package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "net/http/httputil"
    "net/url"
    "os"
    "strings"
    "sync"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
    "github.com/sirupsen/logrus"
    "golang.org/x/time/rate"
    "github.com/redis/go-redis/v9"
)

// ----------------------------------
// Service discovery
// ----------------------------------

type serviceInfo struct {
    Name    string `json:"name"`
    Address string `json:"address"`
}

type registryResponse struct {
    Status int            `json:"status"`
    Data   []serviceInfo  `json:"data"`
}

type Discovery struct {
    registryURL     string
    refreshInterval time.Duration

    mu       sync.RWMutex
    services map[string]string // name -> address
    logger   *logrus.Logger
}

func NewDiscovery(registryURL string, refreshInterval time.Duration, logger *logrus.Logger) *Discovery {
    d := &Discovery{
        registryURL:     strings.TrimSuffix(registryURL, "/"),
        refreshInterval: refreshInterval,
        services:        make(map[string]string),
        logger:          logger,
    }
    go d.refreshLoop()
    return d
}

func (d *Discovery) refreshLoop() {
    ticker := time.NewTicker(d.refreshInterval)
    for {
        d.refresh()
        <-ticker.C
    }
}

func (d *Discovery) refresh() {
    resp, err := http.Get(fmt.Sprintf("%s/services", d.registryURL))
    if err != nil {
        d.logger.WithError(err).Warn("Failed to fetch services from registry")
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        d.logger.Warnf("Unexpected status from registry: %d", resp.StatusCode)
        return
    }

    var rr registryResponse
    if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
        d.logger.WithError(err).Warn("Failed to decode registry response")
        return
    }

    m := make(map[string]string)
    for _, svc := range rr.Data {
        m[svc.Name] = strings.TrimSuffix(svc.Address, "/")
    }

    d.mu.Lock()
    d.services = m
    d.mu.Unlock()

    d.logger.Debugf("Service registry refreshed: %+v", m)
}

func (d *Discovery) GetAddress(name string) string {
    d.mu.RLock()
    addr := d.services[name]
    d.mu.RUnlock()
    return addr
}

func (d *Discovery) GetAllServices() map[string]string {
    d.mu.RLock()
    defer d.mu.RUnlock()
    services := make(map[string]string)
    for k, v := range d.services {
        services[k] = v
    }
    return services
}

// ----------------------------------
// Rate limiter per IP
// ----------------------------------

type ipRateLimiter struct {
    visitors map[string]*rate.Limiter
    mu       sync.Mutex
    r        rate.Limit
    b        int
}

func newIPRateLimiter(r rate.Limit, b int) *ipRateLimiter {
    return &ipRateLimiter{
        visitors: make(map[string]*rate.Limiter),
        r:        r,
        b:        b,
    }
}

func (i *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
    i.mu.Lock()
    defer i.mu.Unlock()
    limiter, exists := i.visitors[ip]
    if !exists {
        limiter = rate.NewLimiter(i.r, i.b)
        i.visitors[ip] = limiter
    }
    return limiter
}

// ----------------------------------
// Route mapping
// ----------------------------------

type routeMapping struct {
    Prefix        string
    ServiceName   string
    RewritePrefix string
    AuthRequired  bool
}

// ----------------------------------
// JWT utilities
// ----------------------------------

func parseJWT(tokenString string, secret []byte) (jwt.MapClaims, error) {
    token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
        return secret, nil
    })
    if err != nil {
        return nil, err
    }
    if !token.Valid {
        return nil, fmt.Errorf("token invalid")
    }
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return nil, fmt.Errorf("invalid claims")
    }
    return claims, nil
}

// ----------------------------------
// Health check structures
// ----------------------------------

type HealthStatus struct {
    Status    string            `json:"status"`
    Timestamp time.Time         `json:"timestamp"`
    Services  map[string]string `json:"services"`
    Gateway   GatewayHealth     `json:"gateway"`
}

type GatewayHealth struct {
    Status       string `json:"status"`
    Uptime       string `json:"uptime"`
    Version      string `json:"version"`
    Environment  string `json:"environment"`
}

// ----------------------------------
// Gateway handler factory
// ----------------------------------

func makeProxyHandler(mapping routeMapping, disc *Discovery, logger *logrus.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        // No JWT handling here – performed globally.

        // Lookup service
        targetAddr := disc.GetAddress(mapping.ServiceName)
        if targetAddr == "" {
            logger.Errorf("service %s not found in registry", mapping.ServiceName)
            c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
            return
        }

        // Build reverse proxy
        targetURL, err := url.Parse(targetAddr)
        if err != nil {
            logger.WithError(err).Error("invalid service address")
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid service address"})
            return
        }

        // Build director function
        director := func(req *http.Request) {
            req.URL.Scheme = targetURL.Scheme
            req.URL.Host = targetURL.Host
            // rewrite path
            incomingPath := req.URL.Path
            trimmed := strings.TrimPrefix(incomingPath, mapping.Prefix)
            newPath := mapping.RewritePrefix + trimmed
            if !strings.HasPrefix(newPath, "/") {
                newPath = "/" + newPath
            }
            req.URL.Path = newPath
            req.URL.RawPath = newPath
            // retain query
            // headers already present
            req.Host = targetURL.Host
        }

        proxy := &httputil.ReverseProxy{
            Director: director,
            Transport: &http.Transport{
                Proxy:               http.ProxyFromEnvironment,
                ResponseHeaderTimeout: 10 * time.Second,
            },
            ErrorHandler: func(rw http.ResponseWriter, r *http.Request, e error) {
                logger.WithError(e).Error("proxy error")
                rw.WriteHeader(http.StatusBadGateway)
            },
        }

        // Apply per-request timeout
        ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
        defer cancel()
        c.Request = c.Request.WithContext(ctx)

        proxy.ServeHTTP(c.Writer, c.Request)
    }
}

// ----------------------------------
// Global JWT middleware (validate all requests except login/signup)
// ----------------------------------

func jwtAuthMiddleware(secret []byte, redisClient *redis.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        path := c.Request.URL.Path

        // Skip authentication for login, signup, and health endpoints
        if strings.HasPrefix(path, "/api/v1/auth/login") || 
           strings.HasPrefix(path, "/api/v1/auth/signup") || 
           strings.HasPrefix(path, "/api/v1/auth/register") ||
           strings.HasPrefix(path, "/health") ||
           strings.HasSuffix(path, "/health") ||
           strings.HasPrefix(path, "/service-registry/health") {
            c.Next()
            return
        }

        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenString == authHeader {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
            c.Abort()
            return
        }

        claims, err := parseJWT(tokenString, secret)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        // Blacklist check using Redis (jti)
        if jti, ok := claims["jti"]; ok {
            jtiStr := fmt.Sprintf("%v", jti)
            ctx := context.Background()
            if _, err := redisClient.Get(ctx, jtiStr).Result(); err == nil {
                // jti found in blacklist
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has been revoked"})
                c.Abort()
                return
            } else if err != nil && err != redis.Nil {
                // Redis error
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
                c.Abort()
                return
            }
        }

        // Propagate user info to downstream services via headers
        if uid, ok := claims["user_id"]; ok {
            c.Request.Header.Set("X-User-ID", fmt.Sprintf("%v", uid))
            c.Set("user_id", uid)
        }
        if email, ok := claims["email"]; ok {
            c.Request.Header.Set("X-User-Email", fmt.Sprintf("%v", email))
        }

        c.Next()
    }
}

// ----------------------------------
// Main
// ----------------------------------

var startTime = time.Now()

func main() {
    // Logger
    logger := logrus.New()
    logger.SetFormatter(&logrus.JSONFormatter{})

    // Env vars
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    registryURL := os.Getenv("SERVICE_REGISTRY_URL")
    if registryURL == "" {
        log.Fatal("SERVICE_REGISTRY_URL env required")
    }

    jwtSecret := []byte(os.Getenv("JWT_SECRET_KEY"))
    if len(jwtSecret) == 0 {
        // not fatal, but warn
        logger.Warn("JWT_SECRET_KEY not set; JWT validation will fail")
    }

    environment := os.Getenv("ENVIRONMENT")
    if environment == "" {
        environment = "development"
    }

    version := os.Getenv("VERSION")
    if version == "" {
        version = "1.0.0"
    }

    // Init discovery
    discovery := NewDiscovery(registryURL, 20*time.Second, logger)

    // Rate limiter: 60 req/min => 1 req per second burst 60
    rl := newIPRateLimiter(1, 60) // limit 1 token per second, burst 60

    // Route mappings
    mappings := []routeMapping{
        {
            Prefix:        "/api/v1/auth",
            ServiceName:   "auth-service",
            RewritePrefix: "/auth",
            AuthRequired:  false,
        },
        {
            Prefix:        "/api/v1/posts",
            ServiceName:   "post-service",
            RewritePrefix: "/post",
            AuthRequired:  false, // specific endpoints will check internally
        },
        {
            Prefix:        "/api/v1/comments",
            ServiceName:   "comment-service",
            RewritePrefix: "/comment",
            AuthRequired:  false,
        },
        {
            Prefix:        "/api/v1/profile",
            ServiceName:   "user-profile-service",
            RewritePrefix: "/profile",
            AuthRequired:  true,
        },
    }

    router := gin.New()
    router.Use(gin.Recovery())

    // Initialize Redis client for blacklist checks
    redisAddr := os.Getenv("REDIS_ADDR")
    if redisAddr == "" {
        redisAddr = "redis:6379"
    }

    redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})

    // Global authentication middleware
    router.Use(jwtAuthMiddleware(jwtSecret, redisClient))

    // Logging middleware
    router.Use(func(c *gin.Context) {
        start := time.Now()
        c.Next()
        latency := time.Since(start)
        logger.WithFields(logrus.Fields{
            "status":   c.Writer.Status(),
            "method":   c.Request.Method,
            "path":     c.Request.URL.Path,
            "ip":       c.ClientIP(),
            "latency":  latency.String(),
            "userAgent": c.Request.UserAgent(),
        }).Info("request completed")
    })

    // Rate limiting middleware
    router.Use(func(c *gin.Context) {
        ip := c.ClientIP()
        limiter := rl.getLimiter(ip)
        if !limiter.Allow() {
            c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
            c.Abort()
            return
        }
        c.Next()
    })

    // Enhanced health endpoint for gateway
    router.GET("/health", func(c *gin.Context) {
        uptime := time.Since(startTime)
        
        healthStatus := HealthStatus{
            Status:    "healthy",
            Timestamp: time.Now(),
            Services:  discovery.GetAllServices(),
            Gateway: GatewayHealth{
                Status:      "healthy",
                Uptime:      uptime.String(),
                Version:     version,
                Environment: environment,
            },
        }
        
        c.JSON(http.StatusOK, healthStatus)
    })

    // Detailed health endpoint
    router.GET("/health/detailed", func(c *gin.Context) {
        uptime := time.Since(startTime)
        services := discovery.GetAllServices()
        
        // Check if critical services are available
        criticalServices := []string{"auth-service", "post-service", "comment-service", "user-profile-service"}
        missingServices := []string{}
        
        for _, service := range criticalServices {
            if _, exists := services[service]; !exists {
                missingServices = append(missingServices, service)
            }
        }
        
        status := "healthy"
        if len(missingServices) > 0 {
            status = "degraded"
        }
        
        healthStatus := gin.H{
            "status":    status,
            "timestamp": time.Now(),
            "gateway": gin.H{
                "status":      "healthy",
                "uptime":      uptime.String(),
                "version":     version,
                "environment": environment,
            },
            "services": gin.H{
                "available": services,
                "missing":   missingServices,
                "count":     len(services),
            },
            "rate_limiter": gin.H{
                "enabled": true,
                "limit":   "60 requests per minute",
            },
        }
        
        if status == "degraded" {
            c.JSON(http.StatusServiceUnavailable, healthStatus)
        } else {
            c.JSON(http.StatusOK, healthStatus)
        }
    })

    // Register proxy handlers
    for _, m := range mappings {
        // path with wildcard
        pattern := m.Prefix + "/*action"
        router.Any(pattern, makeProxyHandler(m, discovery, logger))
        // Add root path handler
        if m.Prefix == "/api/v1/posts" {
            router.Any(m.Prefix, makeProxyHandler(m, discovery, logger))
        }
    }

    // Start server
    addr := fmt.Sprintf(":%s", port)
    logger.Infof("API Gateway listening on %s", addr)
    if err := router.Run(addr); err != nil {
        logger.WithError(err).Fatal("Failed to start API Gateway")
    }
}