package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/blogging-platform/service-registry/internal/models"
	"github.com/go-redis/redis/v8"
)

const (
	serviceTTL = 60 * time.Second
)

var (
	ErrServiceNotFound = errors.New("service not found")
	ErrServiceExists = errors.New("service already exists")
	ErrInvalidService = errors.New("invalid service data")
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(addr string) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisStore{
		client: client,
	}, nil
}

func (s *RedisStore) RegisterService(ctx context.Context, service *models.Service) error {
	if service == nil || service.Name == "" || service.Address == "" {
		return ErrInvalidService
	}

	exists, err := s.client.Exists(ctx, service.Name).Result()
	if err != nil {
		return err
	}
	if exists == 1 {
		return ErrServiceExists
	}

	data, err := json.Marshal(service)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, service.Name, data, serviceTTL).Err()
}

func (s *RedisStore) UpdateTTL(ctx context.Context, serviceName string) error {
	if serviceName == "" {
		return ErrInvalidService
	}

	exists, err := s.client.Exists(ctx, serviceName).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return ErrServiceNotFound
	}

	return s.client.Expire(ctx, serviceName, serviceTTL).Err()
}

func (s *RedisStore) GetServices(ctx context.Context) ([]*models.Service, error) {
	keys, err := s.client.Keys(ctx, "*").Result()
	if err != nil {
		return nil, err
	}

	services := make([]*models.Service, 0, len(keys))
	for _, key := range keys {
		data, err := s.client.Get(ctx, key).Result()
		if err != nil {
			if err == redis.Nil {
				continue 
			}
			return nil, err
		}

		var service models.Service
		if err := json.Unmarshal([]byte(data), &service); err != nil {
			return nil, err 
		}

		services = append(services, &service)
	}

	return services, nil
}

func (s *RedisStore) Close() error {
	return s.client.Close()
} 