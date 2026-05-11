package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	sharedpostgres "project-neo/shared/postgres"
	workersinternal "project-neo/workers/internal"
	"project-neo/workers/internal/metrics"
	"project-neo/workers/parser"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/uptrace/bun"
)

func run() error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}

	bunDB, err := initDB(databaseURL)
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

	connectors, err := buildConnectors(ctx, bunDB, logger)
	if err != nil {
		return fmt.Errorf("build connectors: %w", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}
	reg := metrics.NewRegistry()
	httpMetrics, parserMetrics := metrics.New(reg)
	health := newRuntimeHealth()
	srv := startHealthServer(port, logger, reg, httpMetrics, health)

	provider := parser.NewLLMProvider(logger)
	go parser.StartRecovery(ctx, bunDB, provider, parserMetrics, logger)
	fatalErr := make(chan error, 1)
	go func() {
		if err := parser.StartListener(ctx, databaseURL, bunDB, provider, parserMetrics, logger, health.markParserReady); err != nil {
			health.markParserFailed(err)
			fatalErr <- err
		}
	}()

	return waitForShutdown(cancel, connectors, srv, fatalErr, logger)
}

func initDB(databaseURL string) (*bun.DB, error) {
	bunDB, err := sharedpostgres.NewDB(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	return bunDB, nil
}

func buildConnectors(ctx context.Context, bunDB *bun.DB, logger *slog.Logger) ([]workersinternal.Connector, error) {
	connectors, err := workersinternal.NewConnectors(ctx, bunDB, logger)
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

func startHealthServer(port string, logger *slog.Logger, reg *prometheus.Registry, httpMetrics *metrics.HTTP, health *runtimeHealth) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/health", instrumentHTTP(httpMetrics, "/health", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if ok, reason := health.snapshot(); !ok {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":          "degraded",
				"service":         "workers",
				"parser_listener": reason,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":          "ok",
			"service":         "workers",
			"parser_listener": "ok",
		})
	})))
	mux.Handle("/metrics", metrics.Handler(reg))
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		logger.Info("workers health server listening", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("health server error", "error", err)
		}
	}()
	return srv
}

type runtimeHealth struct {
	mu          sync.RWMutex
	parserReady bool
	parserErr   string
}

func newRuntimeHealth() *runtimeHealth {
	return &runtimeHealth{}
}

func (h *runtimeHealth) markParserReady() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.parserReady = true
	h.parserErr = ""
}

func (h *runtimeHealth) markParserFailed(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.parserReady = false
	h.parserErr = err.Error()
}

func (h *runtimeHealth) snapshot() (bool, string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.parserErr != "" {
		return false, h.parserErr
	}
	if !h.parserReady {
		return false, "starting"
	}
	return true, "ok"
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func instrumentHTTP(m *metrics.HTTP, route string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		m.RequestsTotal.WithLabelValues(r.Method, route, strconv.Itoa(status)).Inc()
		m.RequestDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	})
}

func waitForShutdown(
	cancel context.CancelFunc,
	connectors []workersinternal.Connector,
	srv *http.Server,
	fatalErr <-chan error,
	logger *slog.Logger,
) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	var runErr error
	select {
	case <-quit:
		logger.Info("shutting down workers service")
	case runErr = <-fatalErr:
		logger.Error("workers service component failed", "error", runErr)
	}

	cancel()

	for _, c := range connectors {
		c.Stop()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if shutdownErr := srv.Shutdown(shutdownCtx); shutdownErr != nil {
		logger.Error("health server shutdown error", "error", shutdownErr)
	}
	return runErr
}
