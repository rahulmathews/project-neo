package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"project-neo/graphql-api/graph/generated"
	"project-neo/graphql-api/graph/resolvers"
	"project-neo/graphql-api/internal/auth"
	"project-neo/graphql-api/internal/httpx"
	ipostgres "project-neo/graphql-api/internal/postgres"
	"project-neo/shared/postgres"
	"project-neo/shared/repository"

	"github.com/uptrace/bun"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck("http://localhost:8082/health"))
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	jwtSecret := os.Getenv("SUPABASE_JWT_SECRET")
	if jwtSecret == "" {
		return fmt.Errorf("SUPABASE_JWT_SECRET is required")
	}
	isProd := strings.EqualFold(os.Getenv("ENV"), "production")
	cfg := loadHTTPConfig()

	db, err := postgres.NewDB(dsn)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("close database", "error", err)
		}
	}()

	broker := ipostgres.NewBroker()

	rideRepo := postgres.NewRideRepository(db)
	matchRepo := postgres.NewMatchRepository(db)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go ipostgres.StartListener(ctx, logger, dsn, rideRepo, matchRepo, broker)

	rootHandler := buildRootHandler(buildResolver(db, broker, rideRepo, matchRepo), jwtSecret, cfg, isProd, logger)

	httpSrv := &http.Server{
		Addr:              ":" + port,
		Handler:           rootHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return serveAndWait(ctx, httpSrv, logger)
}

func buildResolver(
	db *bun.DB,
	broker *ipostgres.Broker,
	rideRepo repository.RideRepository,
	matchRepo repository.MatchRepository,
) *resolvers.Resolver {
	return &resolvers.Resolver{
		Users:     postgres.NewUserRepository(db),
		Rides:     rideRepo,
		Matches:   matchRepo,
		Groups:    postgres.NewGroupRepository(db),
		Locations: postgres.NewLocationRepository(db),
		Broker:    broker,
	}
}

func buildGraphQLServer(resolver *resolvers.Resolver, jwtSecret string, isProd bool) *handler.Server {
	gqlSrv := handler.New(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))
	if !isProd {
		gqlSrv.Use(extension.Introspection{})
	}
	gqlSrv.AddTransport(transport.Options{})
	gqlSrv.AddTransport(transport.GET{})
	gqlSrv.AddTransport(transport.POST{})
	gqlSrv.AddTransport(transport.MultipartForm{})
	gqlSrv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		InitFunc:              websocketInitFunc(jwtSecret),
	})
	return gqlSrv
}

func buildRootHandler(
	resolver *resolvers.Resolver,
	jwtSecret string,
	cfg httpConfig,
	isProd bool,
	logger *slog.Logger,
) http.Handler {
	gqlSrv := buildGraphQLServer(resolver, jwtSecret, isProd)
	mux := http.NewServeMux()
	if isProd {
		mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		})
	} else {
		mux.Handle("/", playground.Handler("GraphQL Playground", "/query"))
	}
	mux.Handle("/query", httpx.Chain(
		gqlSrv,
		auth.Middleware(jwtSecret),
		httpx.RateLimit(cfg.rateLimitRPS, cfg.rateLimitBurst),
		httpx.BodyLimit(cfg.maxBodyBytes),
	))
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"ok","service":"graphql-api"}`)
	})
	return httpx.Chain(
		mux,
		httpx.Recover(logger),
		httpx.RequestLog(logger, "/health"),
		httpx.CORS(cfg.allowedOrigins),
	)
}

func serveAndWait(ctx context.Context, srv *http.Server, logger *slog.Logger) error {
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("graphql-api listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received, draining")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown", "error", err)
	}
	if err := <-serverErr; err != nil {
		return err
	}
	logger.Info("graphql-api stopped cleanly")
	return nil
}

func websocketInitFunc(jwtSecret string) transport.WebsocketInitFunc {
	return func(ctx context.Context, initPayload transport.InitPayload) (context.Context, *transport.InitPayload, error) {
		token := ""
		for _, key := range []string{"Authorization", "authorization"} {
			if v := initPayload.GetString(key); strings.HasPrefix(v, "Bearer ") {
				token = strings.TrimSpace(strings.TrimPrefix(v, "Bearer "))
				break
			}
		}
		if token == "" {
			for _, key := range []string{"token", "access_token"} {
				if v := strings.TrimSpace(initPayload.GetString(key)); v != "" {
					token = strings.TrimSpace(strings.TrimPrefix(v, "Bearer "))
					break
				}
			}
		}
		if token != "" {
			ctx = auth.ContextWithToken(ctx, token, jwtSecret)
		}
		return ctx, nil, nil
	}
}

type httpConfig struct {
	allowedOrigins []string
	maxBodyBytes   int64
	rateLimitRPS   int
	rateLimitBurst int
}

func loadHTTPConfig() httpConfig {
	cfg := httpConfig{
		allowedOrigins: []string{"*"},
		maxBodyBytes:   1 << 20, // 1 MiB
		rateLimitRPS:   50,
		rateLimitBurst: 100,
	}
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		parts := strings.Split(v, ",")
		out := parts[:0]
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		if len(out) > 0 {
			cfg.allowedOrigins = out
		}
	}
	if v := os.Getenv("MAX_BODY_BYTES"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			cfg.maxBodyBytes = n
		}
	}
	if v := os.Getenv("RATE_LIMIT_RPS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.rateLimitRPS = n
		}
	}
	if v := os.Getenv("RATE_LIMIT_BURST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.rateLimitBurst = n
		}
	}
	return cfg
}

func runHealthcheck(url string) int {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck failed: %v\n", err)
		return 1
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "healthcheck close body error: %v\n", closeErr)
		}
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		fmt.Fprintf(os.Stderr, "healthcheck failed: unexpected status %d\n", resp.StatusCode)
		return 1
	}

	fmt.Println("ok")
	return 0
}
