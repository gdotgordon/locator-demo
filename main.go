package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gdotgordon/locator-demo/analyzer"
	"github.com/gdotgordon/locator-demo/api"
	"github.com/gdotgordon/locator-demo/store"
	"github.com/gdotgordon/locator-demo/types"
	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

func main() {
	var err error
	cli, err := NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating redis client: '%s'\n", err)
		os.Exit(1)
	}
	if cli.Del(types.LatencyKey).Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting key: '%s'\n", err)
		os.Exit(1)
	}
	defer cli.Del(types.LatencyKey)

	// We'll propagate the context with cancel thorughout the program,
	// such as http clients, server methods we implement, and other
	// loops using channels.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create and run the analyzer, which is the receiver of the Redis events
	// from Redis actions of the other code, such as the Locator.
	analyz, err := analyzer.New(cli)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating analyzer: '%s'\n", err)
		os.Exit(1)
	}
	analyz.Run(ctx)

	// Create the server to handle geocode lookups.  The API module will
	// set up the routes, as we don't need to know the details in the
	// main program.
	r := mux.NewRouter()
	if err = api.Init(ctx, r, store.NewRedisStore(cli)); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting api: '%s'\n", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Handler:      r,
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start Server
	go func() {
		log.Println("Starting Server")
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	//time.Sleep(5 * time.Second)
	fmt.Println("*****************Setting value!")
	err = cli.LPush(types.LatencyKey, 3, 4, 5.7789, 19).Err()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting key: '%s'\n", err)
		os.Exit(1)
	}

	// Block until we shutdown.
	waitForShutdown(ctx, srv)
}

func NewClient() (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	_, err := client.Ping().Result()
	if err != nil {
		return nil, err
	}

	if client.ConfigSet("notify-keyspace-events", "KEA").Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting keyspace notify: '%s'\n", err)
		os.Exit(1)
	}
	return client, nil
}

func waitForShutdown(ctx context.Context, srv *http.Server) {
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	fmt.Println("blocking on signal ...")
	sig := <-interruptChan
	fmt.Printf("received signal: %v\n", sig)

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	srv.Shutdown(ctx)

	log.Println("Shutting down")
}
