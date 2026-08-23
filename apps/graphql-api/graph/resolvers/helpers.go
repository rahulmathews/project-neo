package resolvers

import (
	"context"
	"errors"
	"fmt"

	"project-neo/shared/repository"

	"github.com/google/uuid"
)

func (r *mutationResolver) ensureMatchParticipant(ctx context.Context, matchID, userID uuid.UUID) error {
	match, err := r.Resolver.Matches.GetByID(ctx, matchID)
	if err != nil {
		return err
	}
	if match.RiderID != userID && match.DriverID != userID {
		return fmt.Errorf("match access: %w", repository.ErrForbidden)
	}
	return nil
}

// nilIfNotFound converts ErrNotFound into a null result. For nullable relation
// fields a dangling reference must not error out the whole parent row.
func nilIfNotFound[T any](v *T, err error) (*T, error) {
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil
	}
	return v, err
}
