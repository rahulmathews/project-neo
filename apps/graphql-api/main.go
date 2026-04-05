package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"project-neo/graphql-api/graph/generated"
	"project-neo/graphql-api/graph/resolvers"
	"project-neo/graphql-api/internal/auth"
	ipostgres "project-neo/graphql-api/internal/postgres"
	"project-neo/shared/postgres"
)

func main() {
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
		log.Fatalf("connect to database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	broker := ipostgres.NewBroker()

	rideRepo := postgres.NewRideRepository(db)
	matchRepo := postgres.NewMatchRepository(db)

	// Start LISTEN/NOTIFY listener in background — receives repo interfaces, not concrete types
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

	srv := handler.New(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})

	mux := http.NewServeMux()
	mux.Handle("/", playground.Handler("GraphQL Playground", "/query"))
	mux.Handle("/query", auth.Middleware(jwtSecret)(srv))
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"ok","service":"graphql-api"}`)
	})

	log.Printf("graphql-api listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
