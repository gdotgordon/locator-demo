package analyzer

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gdotgordon/locator-demo/types"
	"github.com/go-redis/redis"
)

type Analyzer struct {
	cli *redis.Client
}

func New(cli *redis.Client) (*Analyzer, error) {
	return &Analyzer{cli: cli}, nil
}

func (a *Analyzer) Run(ctx context.Context) {
	topic := fmt.Sprintf("__keyspace@0__:%s*", types.KeyPrefix)
	sub := a.cli.PSubscribe(topic)
	// Wait for confirmation that subscription is created before publishing anything.
	m, err := sub.Receive()
	if err != nil {
		panic(err)
	}
	fmt.Printf("received: %+v\n", m)
	eventChan := sub.Channel()

	go func() {
		defer sub.Close()
		for {
			select {
			case <-ctx.Done():
				break
			case msg := <-eventChan:
				fmt.Printf(" go routine received message: %+v\n", msg)
				fmt.Printf("with %v, %v, payload: '%v'\n", msg.Channel, msg.Pattern, msg.Payload)
				ndx := strings.Index(msg.Channel, ":")
				key := msg.Channel[ndx+1:]
				fmt.Println("key: ", key)
				res, err := a.cli.LRange(types.LatencyKey, 0, 100).Result()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error getting key: '%s'\n", err)
					os.Exit(1)
				}
				fmt.Printf("value is '%s'\n", res)
			}
		}
	}()

}

func (a *Analyzer) GetMedianLatency() (time.Duration, error) {
	res, err := a.cli.LRange(types.LatencyKey, 0, 100).Result()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting keys: '%s'\n", err)
		os.Exit(1)
	}
	var sum int
	for _, v := range res {
		f, err := strconv.Atoi(v)
		if err != nil {
			panic(err)
		}
		sum += f
	}
	//ret := sum/len(res)
	return 0, nil
}
