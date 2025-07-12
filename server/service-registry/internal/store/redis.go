package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/blogging-platform/service-registry/internal/models"
	"github.com/go-redis/redis/v8"
)

const (
	serviceTTL = 60 * time.Second
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
	data, err := json.Marshal(service)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, service.Name, data, serviceTTL).Err()
}

func (s *RedisStore) UpdateTTL(ctx context.Context, serviceName string) error {
	exists, err := s.client.Exists(ctx, serviceName).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return redis.Nil
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
			continue
		}

		var service models.Service
		if err := json.Unmarshal([]byte(data), &service); err != nil {
			continue
		}

		services = append(services, &service)
	}

	return services, nil
}

func (s *RedisStore) Close() error {
	return s.client.Close()
} 