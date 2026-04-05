package postgres

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"project-neo/graphql-api/internal/repository"
)

// StartListener opens a persistent PostgreSQL LISTEN connection and fans out
// NOTIFY events to the broker. Runs until ctx is cancelled.
// Receives repository interfaces to avoid depending on concrete postgres types.
func StartListener(ctx context.Context, dsn string, rides repository.RideRepository, matches repository.MatchRepository, broker *Broker) {
	listener := pq.NewListener(dsn, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("listener event %d: %v", ev, err)
		}
	})

	if err := listener.Listen("rides_added"); err != nil {
		log.Printf("listen rides_added: %v", err)
	}
	if err := listener.Listen("rides_updated"); err != nil {
		log.Printf("listen rides_updated: %v", err)
	}
	if err := listener.Listen("matches_updated"); err != nil {
		log.Printf("listen matches_updated: %v", err)
	}

	log.Println("postgres listener started")

	for {
		select {
		case <-ctx.Done():
			listener.Close()
			return
		case n := <-listener.Notify:
			if n == nil {
				continue
			}
			switch n.Channel {
			case "rides_added":
				handleRideAdded(ctx, n.Extra, rides, broker)
			case "rides_updated":
				handleRideUpdated(ctx, n.Extra, rides, broker)
			case "matches_updated":
				handleMatchUpdated(ctx, n.Extra, matches, broker)
			}
		}
	}
}

func handleRideAdded(ctx context.Context, idStr string, repo repository.RideRepository, broker *Broker) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return
	}
	ride, err := repo.GetByID(ctx, id)
	if err != nil {
		log.Printf("listener: fetch ride %s: %v", idStr, err)
		return
	}
	broker.PublishRideAdded(RideEvent{Ride: ride, GroupID: ride.GroupID.String()})
}

func handleRideUpdated(ctx context.Context, idStr string, repo repository.RideRepository, broker *Broker) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return
	}
	ride, err := repo.GetByID(ctx, id)
	if err != nil {
		log.Printf("listener: fetch ride %s: %v", idStr, err)
		return
	}
	broker.PublishRideUpdated(RideEvent{Ride: ride, GroupID: ride.GroupID.String()})
}

func handleMatchUpdated(ctx context.Context, idStr string, repo repository.MatchRepository, broker *Broker) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return
	}
	match, err := repo.GetByID(ctx, id)
	if err != nil {
		log.Printf("listener: fetch match %s: %v", idStr, err)
		return
	}
	broker.PublishMatchUpdated(match)
}
