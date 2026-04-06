package parser

import (
	"errors"
	"time"

	"project-neo/shared/model"
)

// ErrNotARide is returned by extractWithHaiku when the message is clearly not
// a ride request or offer. The extractor marks parse_status = SKIPPED.
var ErrNotARide = errors.New("not a ride message")

// ParsedRide holds extraction results. GroupID and MessageID are NOT stored here —
// they come from the model.Message row passed through the pipeline.
type ParsedRide struct {
	RideType         model.RideType // NEED_RIDE | RIDE_AVAILABLE
	FromLocationText *string        // nil if not found; stored in rides.from_location_text
	ToLocationText   *string        // nil if not found; stored in rides.to_location_text
	IsImmediate      bool
	DepartureTime    *time.Time
	Cost             *float64
	Currency         *string // nil → writer defaults to "USD"
	Distance         *float64
}
