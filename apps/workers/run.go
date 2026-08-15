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

	// Order matters: the health endpoint must be reachable before anything
	// that can take long (or wait indefinitely, like WhatsApp QR pairing).
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

	// Connectors last, supervised: connection failures and pending QR pairing
	// must never crash the service — the parser pipeline works regardless.
	sup := workersinternal.StartConnectorSupervisor(ctx, bunDB, logger, health.setWhatsApp)

	return waitForShutdown(cancel, sup, srv, fatalErr, logger)
}

func initDB(databaseURL string) (*bun.DB, error) {
	bunDB, err := sharedpostgres.NewDB(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	return bunDB, nil
}

func startHealthServer(port string, logger *slog.Logger, reg *prometheus.Registry, httpMetrics *metrics.HTTP, health *runtimeHealth) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/health", instrumentHTTP(httpMetrics, "/health", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		parserOK, parserReason, waStatus := health.snapshot()
		status := "ok"
		if !parserOK {
			// The parser listener is the service's core — its failure is a
			// real fault and makes the container unhealthy.
			w.WriteHeader(http.StatusServiceUnavailable)
			status = "degraded"
		} else if waStatus != "connected" {
			// WhatsApp pairing is an operator step, not a fault: report
			// degraded but stay 200 so `docker compose up` is green on a
			// fresh machine while the QR waits in the logs.
			status = "degraded"
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":          status,
			"service":         "workers",
			"parser_listener": parserReason,
			"whatsapp":        waStatus,
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
	waStatus    string
}

func newRuntimeHealth() *runtimeHealth {
	return &runtimeHealth{waStatus: "starting"}
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

func (h *runtimeHealth) setWhatsApp(status string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.waStatus = status
}

func (h *runtimeHealth) snapshot() (parserOK bool, parserReason, waStatus string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	waStatus = h.waStatus
	if h.parserErr != "" {
		return false, h.parserErr, waStatus
	}
	if !h.parserReady {
		return false, "starting", waStatus
	}
	return true, "ok", waStatus
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
	sup *workersinternal.Supervisor,
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

	sup.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if shutdownErr := srv.Shutdown(shutdownCtx); shutdownErr != nil {
		logger.Error("health server shutdown error", "error", shutdownErr)
	}
	return runErr
}
