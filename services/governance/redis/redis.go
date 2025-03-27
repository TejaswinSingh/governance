package redis

import (
	"context"
	"os"
	"time"

	"github.com/TejaswinSingh/version0/services/governance/errors"
	"github.com/go-kit/log"
	"github.com/redis/go-redis/v9"
)

type RedisService struct {
	client *redis.Client
}

func NewService(addr, password string, db int) *RedisService {
	return &RedisService{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
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

func (r *RedisService) Set(ctx context.Context, key string, value any, expiration time.Duration) *errors.ServiceError {
	err := r.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		logger.Log("Error", err.Error())
		return errors.ErrRedis
	}
	return nil
}

func (r *RedisService) Get(ctx context.Context, key string) (string, *errors.ServiceError) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", errors.ErrRedisKeyNotExist
		}
		logger.Log("Error", err.Error())
		return "", errors.ErrRedis
	}
	return val, nil
}

var logger log.Logger

func init() {
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
}
