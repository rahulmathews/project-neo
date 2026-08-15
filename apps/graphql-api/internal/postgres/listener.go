package postgres

import (
	"context"
	"log/slog"
	"time"

	"project-neo/shared/repository"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const (
	listenerBaseBackoff = 5 * time.Second
	listenerMaxBackoff  = time.Minute
)

// StartListener opens a persistent PostgreSQL LISTEN connection and fans out
// NOTIFY events to the broker. Runs until ctx is cancelled.
// Receives repository interfaces to avoid depending on concrete postgres types.
//
// The listener is supervised: a Listen failure rebuilds the whole listener
// with backoff instead of leaving subscriptions permanently dead behind a
// healthy-looking service. onUp/onDown feed the /health endpoint and also
// track pq's internal reconnects (pq re-LISTENs registered channels itself
// after a drop; a nil notification signals the re-established connection).
func StartListener(
	ctx context.Context,
	logger *slog.Logger,
	dsn string,
	rides repository.RideRepository,
	matches repository.MatchRepository,
	broker *Broker,
	onUp func(),
	onDown func(error),
) {
	backoff := listenerBaseBackoff
	for {
		listener := pq.NewListener(dsn, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
			if err != nil {
				logger.Error("listener event", "event", ev, "error", err)
			}
			switch ev {
			case pq.ListenerEventConnected, pq.ListenerEventReconnected:
				onUp()
			case pq.ListenerEventDisconnected, pq.ListenerEventConnectionAttemptFailed:
				onDown(err)
			}
		})

		if listenAll(listener, logger, onDown) {
			backoff = listenerBaseBackoff
			onUp()
			logger.Info("postgres listener started")
			dispatch(ctx, logger, listener, rides, matches, broker)
			if ctx.Err() != nil {
				closeListener(listener, logger)
				return
			}
		}

		closeListener(listener, logger)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, listenerMaxBackoff)
	}
}

// listenAll registers every channel; any failure aborts so the caller rebuilds.
func listenAll(listener *pq.Listener, logger *slog.Logger, onDown func(error)) bool {
	for _, channel := range []string{"rides_added", "rides_updated", "matches_updated"} {
		if err := listener.Listen(channel); err != nil {
			logger.Error("listen channel failed, rebuilding listener", "channel", channel, "error", err)
			onDown(err)
			return false
		}
	}
	return true
}

// dispatch consumes notifications until ctx is cancelled.
func dispatch(
	ctx context.Context,
	logger *slog.Logger,
	listener *pq.Listener,
	rides repository.RideRepository,
	matches repository.MatchRepository,
	broker *Broker,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case n := <-listener.Notify:
			if n == nil {
				// Connection re-established after a drop; pq has re-issued
				// LISTEN for all registered channels already.
				continue
			}
			switch n.Channel {
			case "rides_added":
				handleRideAdded(ctx, logger, n.Extra, rides, broker)
			case "rides_updated":
				handleRideUpdated(ctx, logger, n.Extra, rides, broker)
			case "matches_updated":
				handleMatchUpdated(ctx, logger, n.Extra, matches, broker)
			}
		}
	}
}

func closeListener(listener *pq.Listener, logger *slog.Logger) {
	if err := listener.Close(); err != nil {
		logger.Error("close listener", "error", err)
	}
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
