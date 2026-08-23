package postgres

import (
	"log/slog"
	"sync"

	"project-neo/graphql-api/internal/metrics"
	"project-neo/shared/model"
)

// subscriberBuffer sizes each subscriber's channel. A full buffer drops the
// event for that subscriber (never blocks the publisher) — drops are counted
// and logged so a slow client is visible instead of silently missing rides.
const subscriberBuffer = 16

// RideEvent carries a ride and its group context for subscription fan-out.
type RideEvent struct {
	Ride    *model.Ride
	GroupID string // uuid string for fast comparison
}

// Broker is a thread-safe in-memory pub/sub for GraphQL subscriptions.
// One instance is shared across all active WebSocket connections.
type Broker struct {
	mu           sync.RWMutex
	logger       *slog.Logger
	metrics      *metrics.Subscriptions
	rideAdded    []chan RideEvent
	rideUpdated  []chan RideEvent
	matchUpdated []chan *model.Match
}

func NewBroker(logger *slog.Logger, m *metrics.Subscriptions) *Broker {
	return &Broker{logger: logger, metrics: m}
}

// dropped records a subscriber that missed an event because its buffer was full.
func (b *Broker) dropped(channel string) {
	b.metrics.DroppedEvents.WithLabelValues(channel).Inc()
	b.logger.Warn("subscription event dropped", "channel", channel)
}

// SubscribeRideAdded registers a channel for new ride notifications.
// The returned cancel func must be called when the subscription ends.
func (b *Broker) SubscribeRideAdded() (<-chan RideEvent, func()) {
	ch := make(chan RideEvent, subscriberBuffer)
	b.mu.Lock()
	b.rideAdded = append(b.rideAdded, ch)
	b.mu.Unlock()
	b.metrics.Active.WithLabelValues("ride_added").Inc()
	return ch, func() { b.removeRideAdded(ch) }
}

func (b *Broker) PublishRideAdded(e RideEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.rideAdded {
		select {
		case ch <- e:
		default:
			b.dropped("ride_added")
		}
	}
}

func (b *Broker) removeRideAdded(target chan RideEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, ch := range b.rideAdded {
		if ch == target {
			b.rideAdded = append(b.rideAdded[:i], b.rideAdded[i+1:]...)
			close(ch)
			b.metrics.Active.WithLabelValues("ride_added").Dec()
			return
		}
	}
}

func (b *Broker) SubscribeRideUpdated() (<-chan RideEvent, func()) {
	ch := make(chan RideEvent, subscriberBuffer)
	b.mu.Lock()
	b.rideUpdated = append(b.rideUpdated, ch)
	b.mu.Unlock()
	b.metrics.Active.WithLabelValues("ride_updated").Inc()
	return ch, func() { b.removeRideUpdated(ch) }
}

func (b *Broker) PublishRideUpdated(e RideEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.rideUpdated {
		select {
		case ch <- e:
		default:
			b.dropped("ride_updated")
		}
	}
}

func (b *Broker) removeRideUpdated(target chan RideEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, ch := range b.rideUpdated {
		if ch == target {
			b.rideUpdated = append(b.rideUpdated[:i], b.rideUpdated[i+1:]...)
			close(ch)
			b.metrics.Active.WithLabelValues("ride_updated").Dec()
			return
		}
	}
}

func (b *Broker) SubscribeMatchUpdated() (<-chan *model.Match, func()) {
	ch := make(chan *model.Match, subscriberBuffer)
	b.mu.Lock()
	b.matchUpdated = append(b.matchUpdated, ch)
	b.mu.Unlock()
	b.metrics.Active.WithLabelValues("match_updated").Inc()
	return ch, func() { b.removeMatchUpdated(ch) }
}

func (b *Broker) PublishMatchUpdated(m *model.Match) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.matchUpdated {
		select {
		case ch <- m:
		default:
			b.dropped("match_updated")
		}
	}
}

func (b *Broker) removeMatchUpdated(target chan *model.Match) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, ch := range b.matchUpdated {
		if ch == target {
			b.matchUpdated = append(b.matchUpdated[:i], b.matchUpdated[i+1:]...)
			close(ch)
			b.metrics.Active.WithLabelValues("match_updated").Dec()
			return
		}
	}
}
