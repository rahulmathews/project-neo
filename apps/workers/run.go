package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	sharedpostgres "project-neo/shared/postgres"
	workersinternal "project-neo/workers/internal"

	"github.com/uptrace/bun"
)

func run() error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}

	bunDB, sqlDB, err := initDB(databaseURL)
	if err != nil {
		return fmt.Errorf("database init: %w", err)
	}
	defer func() {
		if closeErr := bunDB.Close(); closeErr != nil {
			logger.Error("failed to close database", "error", closeErr)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connectors, err := buildConnectors(ctx, bunDB, sqlDB, logger)
	if err != nil {
		return fmt.Errorf("build connectors: %w", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}
	srv := startHealthServer(port, logger)

	waitForShutdown(cancel, connectors, srv, logger)
	return nil
}

func initDB(databaseURL string) (*bun.DB, *sql.DB, error) {
	bunDB, err := sharedpostgres.NewDB(databaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("open postgres: %w", err)
	}
	sqlDB := bunDB.DB
	return bunDB, sqlDB, nil
}

func buildConnectors(ctx context.Context, bunDB *bun.DB, sqlDB *sql.DB, logger *slog.Logger) ([]workersinternal.Connector, error) {
	connectors, err := workersinternal.NewConnectors(ctx, bunDB, sqlDB, logger)
	if err != nil {
		return nil, fmt.Errorf("new connectors: %w", err)
	}
	for _, c := range connectors {
		if err := c.Start(ctx); err != nil {
			return nil, fmt.Errorf("start connector: %w", err)
		}
	}
	return connectors, nil
}

func startHealthServer(port string, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"ok","service":"workers"}`)
	})
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		logger.Info("workers health server listening", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("health server error", "error", err)
		}
	}()
	return srv
}

func waitForShutdown(
	cancel context.CancelFunc,
	connectors []workersinternal.Connector,
	srv *http.Server,
	logger *slog.Logger,
) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down workers service")

	cancel()

	for _, c := range connectors {
		c.Stop()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("health server shutdown error", "error", err)
	}
}
