// Package main runs the locator microservice.  It spins up an http
// server to handle requests, which are handled by the api package.
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

	"github.com/gdotgordon/locator-demo/locator/api"
	"github.com/gdotgordon/locator-demo/locator/store"
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

	// We'll propagate the context with cancel thorughout the program,
	// such as http clients, server methods we implement, and other
	// loops using channels.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
