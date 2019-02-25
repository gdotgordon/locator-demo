// Package lock implments a quick and dirty lock/mutex as described
// in the Redis SET command documentation.  The Redlock algorithm as
// implemented in redigo is far superior, but unfortunately I used the
// older go-redis package as that is the one I'm most familiar with.
//
// The flaws of the suggested algorithm are obvious, most noteworthy
// being the fact that a lock holder could have their lock expire before
// they are done with it, so here we've removed the expiration, which is
// arguably not much worse than a held mutex that is never unlocked.
package locking

import (
	"errors"
	"time"

	"github.com/gdotgordon/locator-demo/locator/types"
	"github.com/go-redis/redis"
	"github.com/rs/xid"
)

const (
	retries = 3
	sleep   = 1 * time.Second
)

type Lock struct {
	cli     *redis.Client
	retries int
	expiry  time.Duration
	uniq    string
}

// Create a new lock with the desired settings.
func New(cli *redis.Client, expiry time.Duration, retries int) *Lock {
	return &Lock{cli: cli, expiry: expiry, retries: retries}
}

// Lock uses Redis SET resource-name anystring NX EX max-lock-time to
// set a lock.
func (l *Lock) Lock() error {
	uniq := xid.New().String()
	for i := 0; i < l.retries; i++ {
		s, err := l.cli.SetNX(types.LockKey, uniq, 0).Result()
		if err != nil {
			return err
		}
		if s {
			l.uniq = uniq
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return errors.New("could not acquire lock")
}

func (l *Lock) Unlock() error {
	oval, err := l.cli.Get(types.LockKey).Result()
	if err != nil {
		return err
	}

	// Don't delete the lock if we don't own it anymore.
	if oval != l.uniq {
		return nil
	}
	_, err = l.cli.Del(types.LockKey).Result()
	return err
}
