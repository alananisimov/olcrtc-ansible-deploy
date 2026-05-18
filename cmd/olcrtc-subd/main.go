package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openlibrecommunity/olcrtc/internal/subscription"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	statePath := flag.String("state", "/config/state.yaml", "path to generated subscription state YAML")
	listen := flag.String("listen", ":8080", "HTTP listen address")
	flag.Parse()

	state, err := subscription.LoadState(*statePath)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              *listen,
		Handler:           subscription.Handler(state),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("olcrtc-subd listening on %s", *listen)
		errCh <- server.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("received %s, shutting down", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}
