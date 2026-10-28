package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/BrionesBenjamin22/laboratorio-azure/apps/movies-api/internal/config"
	"github.com/BrionesBenjamin22/laboratorio-azure/apps/movies-api/internal/movies"
)

func TestServerUsesConfigurationTimeouts(t *testing.T) {
	cfg := config.Config{
		HTTPPort:         8080,
		HTTPReadTimeout:  time.Second,
		HTTPWriteTimeout: 2 * time.Second,
		HTTPIdleTimeout:  3 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := newServer(cfg, logger, movies.NewService(testProvider{}))

	if server.ReadHeaderTimeout != cfg.HTTPReadTimeout {
		t.Errorf("expected ReadHeaderTimeout %s, got %s", cfg.HTTPReadTimeout, server.ReadHeaderTimeout)
	}
	if server.ReadTimeout != cfg.HTTPReadTimeout {
		t.Errorf("expected ReadTimeout %s, got %s", cfg.HTTPReadTimeout, server.ReadTimeout)
	}
	if server.WriteTimeout != cfg.HTTPWriteTimeout {
		t.Errorf("expected WriteTimeout %s, got %s", cfg.HTTPWriteTimeout, server.WriteTimeout)
	}
	if server.IdleTimeout != cfg.HTTPIdleTimeout {
		t.Errorf("expected IdleTimeout %s, got %s", cfg.HTTPIdleTimeout, server.IdleTimeout)
	}
}

type testProvider struct{}

func (testProvider) Popular(context.Context, int) (movies.Page, error) {
	return movies.Page{}, nil
}

func TestServeShutsDownWhenContextIsCanceled(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	done := make(chan error, 1)

	go func() {
		done <- serve(ctx, server, listener, time.Second, logger)
	}()

	client := &http.Client{Timeout: time.Second}
	response, err := client.Get("http://" + listener.Addr().String())
	if err != nil {
		t.Fatalf("request running server: %v", err)
	}
	response.Body.Close()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve returned an error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop after context cancellation")
	}
}
