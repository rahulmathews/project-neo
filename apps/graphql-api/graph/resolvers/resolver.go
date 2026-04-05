package resolvers

import (
	"project-neo/graphql-api/internal/postgres"
	"project-neo/shared/repository"
)

// Resolver is the root resolver. All repositories are injected here.
type Resolver struct {
	Users     repository.UserRepository
	Rides     repository.RideRepository
	Matches   repository.MatchRepository
	Groups    repository.GroupRepository
	Locations repository.LocationRepository
	Broker    *postgres.Broker
}
