package postgres

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"project-neo/shared/repository"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// StartListener opens a persistent PostgreSQL LISTEN connection and fans out
// NOTIFY events to the broker. Runs until ctx is cancelled.
// Receives repository interfaces to avoid depending on concrete postgres types.
func StartListener(
	ctx context.Context,
	logger *slog.Logger,
	dsn string,
	rides repository.RideRepository,
	matches repository.MatchRepository,
	broker *Broker,
) {
	listener := pq.NewListener(dsn, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			logger.Error("listener event", "event", ev, "error", err)
		}
	})

	for _, channel := range []string{"rides_added", "ride_occurrences_added", "rides_updated", "matches_updated"} {
		if err := listener.Listen(channel); err != nil {
			logger.Error("listen channel", "channel", channel, "error", err)
		}
	}

	logger.Info("postgres listener started")

	for {
		select {
		case <-ctx.Done():
			if err := listener.Close(); err != nil {
				logger.Error("close listener", "error", err)
			}
			return
		case n := <-listener.Notify:
			if n == nil {
				continue
			}
			switch n.Channel {
			case "rides_added":
				handleRideAdded(ctx, logger, n.Extra, rides, broker)
			case "ride_occurrences_added":
				handleRideOccurrenceAdded(ctx, logger, n.Extra, rides, broker)
			case "rides_updated":
				handleRideUpdated(ctx, logger, n.Extra, rides, broker)
			case "matches_updated":
				handleMatchUpdated(ctx, logger, n.Extra, matches, broker)
			}
		}
	}
}

type rideOccurrencePayload struct {
	RideID  string `json:"ride_id"`
	GroupID string `json:"group_id"`
}

func handleRideOccurrenceAdded(ctx context.Context, logger *slog.Logger, payload string, repo repository.RideRepository, broker *Broker) {
	var p rideOccurrencePayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		logger.Warn("listener: invalid ride occurrence payload", "payload", payload, "error", err)
		return
	}
	id, err := uuid.Parse(p.RideID)
	if err != nil {
		return
	}
	ride, err := repo.GetByID(ctx, id)
	if err != nil {
		logger.Error("listener: fetch occurrence ride", "id", p.RideID, "error", err)
		return
	}
	groupID, err := uuid.Parse(p.GroupID)
	if err == nil {
		ride.GroupID = groupID
	}
	broker.PublishRideAdded(RideEvent{Ride: ride, GroupID: p.GroupID})
}

func handleRideAdded(ctx context.Context, logger *slog.Logger, idStr string, repo repository.RideRepository, broker *Broker) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return
	}
	ride, err := repo.GetByID(ctx, id)
	if err != nil {
		logger.Error("listener: fetch ride", "id", idStr, "error", err)
		return
	}
	broker.PublishRideAdded(RideEvent{Ride: ride, GroupID: ride.GroupID.String()})
}

func handleRideUpdated(ctx context.Context, logger *slog.Logger, idStr string, repo repository.RideRepository, broker *Broker) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return
	}
	ride, err := repo.GetByID(ctx, id)
	if err != nil {
		logger.Error("listener: fetch ride", "id", idStr, "error", err)
		return
	}
	broker.PublishRideUpdated(RideEvent{Ride: ride, GroupID: ride.GroupID.String()})
}

func handleMatchUpdated(ctx context.Context, logger *slog.Logger, idStr string, repo repository.MatchRepository, broker *Broker) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return
	}
	match, err := repo.GetByID(ctx, id)
	if err != nil {
		logger.Error("listener: fetch match", "id", idStr, "error", err)
		return
	}
	broker.PublishMatchUpdated(match)
}
