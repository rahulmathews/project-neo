package postgres

import (
	"sync"

	"project-neo/graphql-api/internal/model"
)

// RideEvent carries a ride and its group context for subscription fan-out.
type RideEvent struct {
	Ride    *model.Ride
	GroupID string // uuid string for fast comparison
}

// Broker is a thread-safe in-memory pub/sub for GraphQL subscriptions.
// One instance is shared across all active WebSocket connections.
type Broker struct {
	mu           sync.RWMutex
	rideAdded    []chan RideEvent
	rideUpdated  []chan RideEvent
	matchUpdated []chan *model.Match
}

func NewBroker() *Broker {
	return &Broker{}
}

// SubscribeRideAdded registers a channel for new ride notifications.
// The returned cancel func must be called when the subscription ends.
func (b *Broker) SubscribeRideAdded() (<-chan RideEvent, func()) {
	ch := make(chan RideEvent, 4)
	b.mu.Lock()
	b.rideAdded = append(b.rideAdded, ch)
	b.mu.Unlock()
	return ch, func() { b.removeRideAdded(ch) }
}

func (b *Broker) PublishRideAdded(e RideEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.rideAdded {
		select {
		case ch <- e:
		default:
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
			return
		}
	}
}

func (b *Broker) SubscribeRideUpdated() (<-chan RideEvent, func()) {
	ch := make(chan RideEvent, 4)
	b.mu.Lock()
	b.rideUpdated = append(b.rideUpdated, ch)
	b.mu.Unlock()
	return ch, func() { b.removeRideUpdated(ch) }
}

func (b *Broker) PublishRideUpdated(e RideEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.rideUpdated {
		select {
		case ch <- e:
		default:
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
			return
		}
	}
}

func (b *Broker) SubscribeMatchUpdated() (<-chan *model.Match, func()) {
	ch := make(chan *model.Match, 4)
	b.mu.Lock()
	b.matchUpdated = append(b.matchUpdated, ch)
	b.mu.Unlock()
	return ch, func() { b.removeMatchUpdated(ch) }
}

func (b *Broker) PublishMatchUpdated(m *model.Match) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.matchUpdated {
		select {
		case ch <- m:
		default:
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
			return
		}
	}
}
