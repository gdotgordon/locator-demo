// Package store defines an interface for storing metrics to a store,
// such as redis, and it also implments the RedisStore, which is the
// required implmementation for the demo.  Note we abstract the store
// into an interface so we can run unit tests without having to actually
// store to redis, among other reasons.
package store

import (
	"time"

	"github.com/gdotgordon/locator-demo/types"
	"github.com/go-redis/redis"
)

type Store interface {
	StoreLatency(d time.Duration) error
	ClearDatabase() error
}

type RedisStore struct {
	cli *redis.Client
}

func NewRedisStore(cli *redis.Client) Store {
	return &RedisStore{cli: cli}
}

func (rs *RedisStore) ClearDatabase() error {
	return rs.cli.FlushDB().Err()
}

func (rs *RedisStore) StoreLatency(d time.Duration) error {
	return rs.cli.LPush(types.LatencyKey, int64(d)).Err()
}
