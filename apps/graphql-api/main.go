package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"project-neo/graphql-api/graph/generated"
	"project-neo/graphql-api/graph/resolvers"
	"project-neo/graphql-api/internal/auth"
	ipostgres "project-neo/graphql-api/internal/postgres"
	"project-neo/shared/postgres"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck("http://localhost:8082/health"))
	}

	if err := run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func run() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	jwtSecret := os.Getenv("SUPABASE_JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("SUPABASE_JWT_SECRET is required")
	}

	db, err := postgres.NewDB(dsn)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	broker := ipostgres.NewBroker()

	rideRepo := postgres.NewRideRepository(db)
	matchRepo := postgres.NewMatchRepository(db)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go ipostgres.StartListener(ctx, dsn, rideRepo, matchRepo, broker)

	resolver := &resolvers.Resolver{
		Users:     postgres.NewUserRepository(db),
		Rides:     rideRepo,
		Matches:   matchRepo,
		Groups:    postgres.NewGroupRepository(db),
		Locations: postgres.NewLocationRepository(db),
		Broker:    broker,
	}

	gqlSrv := handler.New(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))
	gqlSrv.Use(extension.Introspection{})
	gqlSrv.AddTransport(transport.Options{})
	gqlSrv.AddTransport(transport.GET{})
	gqlSrv.AddTransport(transport.POST{})
	gqlSrv.AddTransport(transport.MultipartForm{})
	gqlSrv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})

	mux := http.NewServeMux()
	mux.Handle("/", playground.Handler("GraphQL Playground", "/query"))
	mux.Handle("/query", auth.Middleware(jwtSecret)(gqlSrv))
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"ok","service":"graphql-api"}`)
	})

	httpSrv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("graphql-api listening on :%s", port)
	return httpSrv.ListenAndServe()
}

func runHealthcheck(url string) int {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("healthcheck failed: %v", err)
		return 1
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("healthcheck close body error: %v", closeErr)
		}
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		log.Printf("healthcheck failed: unexpected status %d", resp.StatusCode)
		return 1
	}

	fmt.Println("ok")
	return 0
}
