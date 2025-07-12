package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/omsurase/blogger_microservices/server/post/internal/models"
)

type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisStore(addr string, ttl time.Duration) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("error connecting to Redis: %v", err)
	}

	return &RedisStore{
		client: client,
		ttl:    ttl,
	}, nil
}

func (s *RedisStore) Set(ctx context.Context, post *models.Post) error {
	key := fmt.Sprintf("post:%s", post.ID.String())
	value, err := json.Marshal(post)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, value, s.ttl).Err()
}

func (s *RedisStore) Get(ctx context.Context, id uuid.UUID) (*models.Post, error) {
	key := fmt.Sprintf("post:%s", id.String())
	value, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var post models.Post
	if err := json.Unmarshal([]byte(value), &post); err != nil {
		return nil, err
	}
	return &post, nil
}

func (s *RedisStore) Delete(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("post:%s", id.String())
	return s.client.Del(ctx, key).Err()
} 