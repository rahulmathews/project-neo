package resolvers

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *mutationResolver) ensureMatchParticipant(ctx context.Context, matchID, userID uuid.UUID) error {
	match, err := r.Resolver.Matches.GetByID(ctx, matchID)
	if err != nil {
		return err
	}
	if match.RiderID != userID && match.DriverID != userID {
		return fmt.Errorf("forbidden")
	}
	return nil
}
