package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BrionesBenjamin22/laboratorio-azure/apps/movies-api/internal/config"
	"github.com/BrionesBenjamin22/laboratorio-azure/apps/movies-api/internal/httpapi"
	"github.com/BrionesBenjamin22/laboratorio-azure/apps/movies-api/internal/movies"
	"github.com/BrionesBenjamin22/laboratorio-azure/apps/movies-api/internal/tmdb"
)

func main() {
	if err := run(); err != nil {
		slog.Error("movies-api stopped with an error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	listener, err := net.Listen("tcp", cfg.HTTPAddress())
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.HTTPAddress(), err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	provider, err := tmdb.New(tmdb.Config{
		BaseURL:  cfg.TMDBBaseURL,
		APIToken: cfg.TMDBAPIToken,
		Language: cfg.TMDBLanguage,
		Timeout:  cfg.TMDBTimeout,
	}, logger)
	if err != nil {
		return fmt.Errorf("initialize TMDB provider: %w", err)
	}
	movieService := movies.NewService(provider)

	server := newServer(cfg, logger, movieService)
	return serve(ctx, server, listener, cfg.ShutdownTimeout, logger)
}

func newLogger(level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

func newServer(cfg config.Config, logger *slog.Logger, movieService *movies.Service) *http.Server {
	return &http.Server{
		Addr:              cfg.HTTPAddress(),
		Handler:           httpapi.NewHandler(logger, movieService),
		ReadHeaderTimeout: cfg.HTTPReadTimeout,
		ReadTimeout:       cfg.HTTPReadTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}
}

func serve(
	ctx context.Context,
	server *http.Server,
	listener net.Listener,
	shutdownTimeout time.Duration,
	logger *slog.Logger,
) error {
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Serve(listener)
	}()

	logger.Info("movies-api started", "address", listener.Addr().String())

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		logger.Info("movies-api shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Warn("graceful shutdown deadline reached", "error", err)
		if closeErr := server.Close(); closeErr != nil {
			return fmt.Errorf("force server close after shutdown error %v: %w", err, closeErr)
		}
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	if err := <-serverErrors; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP during shutdown: %w", err)
	}

	logger.Info("movies-api stopped")
	return nil
}
