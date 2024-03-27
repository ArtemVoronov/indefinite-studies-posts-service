package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	redisService "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/redis"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"github.com/go-redis/redis/v8"
)

type RedisCacheService struct {
	redisService *redisService.RedisService
	PostsTTL     time.Duration
}

func CreateRedisCacheService() *RedisCacheService {
	postsTTL := utils.EnvVarDurationDefault("CACHE_POSTS_TTL_IN_MINUTES", time.Minute, 10*time.Minute)
	return &RedisCacheService{
		redisService: redisService.CreateRedisService(),
		PostsTTL:     postsTTL,
	}
}

func (s *RedisCacheService) Shutdown() error {
	result := []error{}
	err := s.redisService.Shutdown()
	if err != nil {
		result = append(result, err)
	}
	if len(result) > 0 {
		return errors.Join(result...)
	}
	return nil
}

func (s *RedisCacheService) Get(key string) (string, error) {
	data, err := s.redisService.WithTimeout(func(cli *redis.Client, ctx context.Context, cancel context.CancelFunc) (any, error) {
		return cli.Get(ctx, key).Result()
	})()

	if err != nil && errors.Is(err, redis.Nil) {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	result, ok := data.(string)
	if !ok {
		return "", fmt.Errorf("unable cast to string")
	}
	return result, err
}

func (s *RedisCacheService) Set(key, value string, expiration time.Duration) error {
	return s.redisService.WithTimeoutVoid(func(cli *redis.Client, ctx context.Context, cancel context.CancelFunc) error {
		err := cli.Set(ctx, key, value, expiration).Err()
		if err != nil {
			return err
		}
		return err
	})()
}
