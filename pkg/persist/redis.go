package persist

import (
	"context"
	"fmt"
	"time"

	"github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/redis/go-redis/v9"
)

type RedisConfiguration struct {
	Host     string
	Port     string
	Password string
	Database int
}

type RedisService struct {
	client *redis.Client
}

func NewRedisService(config *RedisConfiguration) *RedisService {
	return &RedisService{
		client: redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", config.Host, config.Port),
			Password: config.Password,
			DB:       config.Database,
			// PoolSize: 100, default: 10 for each CPU
		})}
}

func (r *RedisService) Ping(ctx context.Context) (string, error) {
	s, err := r.client.Ping(ctx).Result()
	if err != nil {
		return s, err
	}
	return s, nil
}

func (r *RedisService) Close() error {
	err := r.client.Close()
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisService) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	err := r.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return errors.ErrRedis.WithInternalErr(err)
	}
	return nil
}

func (r *RedisService) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", errors.ErrRedisKeyNotExist.WithInternalErr(err)
		}
		return "", errors.ErrRedis.WithInternalErr(err)
	}
	return val, nil
}
