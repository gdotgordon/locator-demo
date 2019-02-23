package store

import (
	"time"

	"github.com/gdotgordon/locator-demo/types"
	"github.com/go-redis/redis"
)

type Store interface {
	StoreLatency(d time.Duration) error
}

type RedisStore struct {
	cli *redis.Client
}

func NewRedisStore(cli *redis.Client) Store {
	return &RedisStore{cli: cli}
}

func (rs *RedisStore) StoreLatency(d time.Duration) error {
	return rs.cli.LPush(types.LatencyKey, int64(d)).Err()
}
