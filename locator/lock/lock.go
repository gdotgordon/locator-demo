// Package lock implments a quick and dirty lock/mutex as described
// in the Redis SET command documentation.  The Redlock algorithm as
// implemented in redigo is far superior, but unfrotunately I used the
// older go-redis package as that is the one I'm most familiar with.
// The flaws of the algorithm shown here are obvious, most noteworthy
// being the fact that a lock holder could have their lock expire before
// they are done with it.
package lock

import (
	"errors"
	"fmt"
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
	expiry  time.Duration
	retries int
	uniq    string
}

func New(cli *redis.Client, expiry time.Duration, reries int) *Lock {
	return &Lock{cli: cli, expiry: expiry, retries: retries}
}

// Lock uses Redis SET resource-name anystring NX EX max-lock-time to
// set a lock.
func (l *Lock) Lock() error {
	uniq := xid.New().String()
	fmt.Printf("unique: %s\n", uniq)
	for i := 0; i < l.retries; i++ {
		s, err := l.cli.SetNX(types.LockKey, uniq, l.expiry).Result()
		fmt.Printf("lock res: %v, err %v\n", s, err)
		if err != nil {
			return err
		}
		if s {
			l.uniq = uniq
			return nil
		}
		time.Sleep(500 * time.Millisecond)
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
	res, err := l.cli.Del(types.LockKey).Result()
	fmt.Printf("unlock key; %s, res: %v, err %v\n", l.uniq, res, err)
	return err
}
