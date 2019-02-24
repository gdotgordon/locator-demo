package lock

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis"
)

func TestLock(t *testing.T) {
	cli, err := NewClient()
	if err != nil {
		t.Fatalf("error creating redis client")
	}

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)

		i := i
		go func() {
			defer wg.Done()

			lock := New(cli, 5*time.Second, 10)

			err := lock.Lock()
			if err != nil {
				t.Fatalf("%d: error creating lock: %v\n", i, err)
			}
			fmt.Printf("%d: got lock\n", i)
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("%d: unlocking\n", i)
			err = lock.Unlock()
			if err != nil {
				t.Fatalf("error unlocking: %v", err)
			}
			fmt.Printf("%d: unlock\n", i)
		}()
	}

	wg.Wait()
}

func NewClient() (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	_, err := client.Ping().Result()
	if err != nil {
		return nil, err
	}
	return client, nil
}
