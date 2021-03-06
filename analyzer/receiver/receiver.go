// Package Receiver implemnts the Redis event listener for events
// generated by the locator.  The receiver has the option of running
// as either a singe goroutine or multiple goroutines.  Because th redis-go
// package supplies a channel to read the events, the event stream may
// easily be multiplexed.  The number of workers may be configured, so
// if the nature of the task requires sequential proessing, numWorkers
// should be set to 1 (in the flag in main.go)
package receiver

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gdotgordon/locator-demo/analyzer/types"
	"github.com/go-redis/redis"
)

// Receiver stores some statistics from the received events.
type Receiver struct {
	cli        *redis.Client
	latencyCnt int64
	succCnt    int64
	errCnt     int64
}

// New creates a new event receiver for keyspace events.
func New(cli *redis.Client) (*Receiver, error) {
	return &Receiver{cli: cli}, nil
}

// Run is the main event loop processor.  For each event read, it
// takes the appropriate action, which in our case is to simply store
// them in our instance.  In real life, you might pass the data off
// to a tracking system like Prometheus or a database.
func (r *Receiver) Run(ctx context.Context, numWorkers int) {
	topic := fmt.Sprintf("__keyspace@0__:%s*", types.KeyPrefix)
	sub := r.cli.PSubscribe(topic)
	// Wait for confirmation that subscription is created before publishing anything.
	_, err := sub.Receive()
	if err != nil {
		panic(err)
	}
	eventChan := sub.Channel()

	// This function will be invoked in thr event loop, and is moved
	// out of the loop for readability's sake.
	var wg sync.WaitGroup
	processEvents := func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-eventChan:
				if !ok {
					return
				}
				log.Printf("Received message: +%v\n", msg)
				ndx := strings.Index(msg.Channel, ":")
				key := msg.Channel[ndx+1:]
				log.Println("key: ", key)
				if strings.HasSuffix(msg.Channel, ":latency") &&
					msg.Payload == "lpush" {
					atomic.AddInt64(&r.latencyCnt, 1)
				} else if strings.HasSuffix(msg.Channel, ":success") &&
					msg.Payload == "incrby" {
					atomic.AddInt64(&r.succCnt, 1)
				} else if strings.HasSuffix(msg.Channel, ":error") &&
					msg.Payload == "incrby" {
					atomic.AddInt64(&r.errCnt, 1)
				}
			}
		}
	}

	// Launch the event loops.
	go func() {
		defer sub.Close()

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				processEvents()
			}()
		}
		wg.Wait()
	}()
}

// GetStats returns a statisitcs object with the accumulated local data.
// For the latency we compute an average of the 100 (or max) latest
// events.
func (r *Receiver) GetStats() (*types.StatsResponse, error) {
	res, err := r.cli.LRange(types.LatencyKey, 0, 100).Result()
	if err != nil {
		return nil, err
	}
	var sum int64
	for _, v := range res {
		f, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
		sum += f
	}
	var avg float64
	if len(res) > 0 {
		avg = float64(sum) / float64(len(res))
	}
	davg := time.Duration(int64(math.Round(avg)))
	return &types.StatsResponse{Success: r.succCnt, Error: r.errCnt,
		LatencyCount: r.latencyCnt, Latency: davg.String()}, nil
}

// Resets the counter and db.  Mostly for testing.
func (r *Receiver) Reset() error {
	r.latencyCnt = 0
	r.succCnt = 0
	r.errCnt = 0
	return r.cli.FlushDB().Err()
}
