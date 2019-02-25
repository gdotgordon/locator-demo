// Package main runs the analyzerr microservice.  It spins up an http
// server to handle requests, which are handled by the api package.
// It also spins up a 'receiver' instance (package receiver), which
// contains both the keystore receive events, plus has APIs to retrieve
// statistics.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gdotgordon/locator-demo/analyzer/api"
	"github.com/gdotgordon/locator-demo/analyzer/receiver"
	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

var (
	numWorkers = flag.Int("numWorkrs", 3, "Number of reciever workers")
)

func main() {
	flag.Parse()

	var err error
	cli, err := NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating redis client: '%s'\n", err)
		os.Exit(1)
	}

	// We'll propagate the context with cancel thorughout the program,
	// such as http clients, server methods we implement, and other
	// loops using channels.
	ctx, _ := context.WithCancel(context.Background())
	//defer cancel()

	// Create and run the receiver, which is the analyzer of the Redis events
	// from Redis actions of the other code, such as the Locator.
	receiver, err := receiver.New(cli)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating receiver: '%s'\n", err)
		os.Exit(1)
	}
	receiver.Run(ctx, *numWorkers)

	// Create the server to handle stats requests.  The API module will
	// set up the routes, as we don't need to know the details in the
	// main program.
	r := mux.NewRouter()
	if err = api.Init(ctx, r, receiver); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting api: '%s'\n", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Handler:      r,
		Addr:         ":8090",
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
	<-interruptChan

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	srv.Shutdown(ctx)

	log.Println("Shutting down")
}
