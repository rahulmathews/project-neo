package repository

import "errors"

// Sentinel errors shared across repositories and resolvers. The GraphQL error
// presenter maps these onto stable client-facing codes; anything not wrapping
// one of them is masked as an internal error.
var (
	// ErrNotFound marks a missing row. Resolvers for nullable fields should
	// convert it to a null result rather than an error.
	ErrNotFound = errors.New("not found")

	// ErrForbidden marks an authorization failure on an operation the caller
	// can name but not touch (someone else's ride, match, etc.).
	ErrForbidden = errors.New("forbidden")

	// ErrConflict marks a uniqueness/consistency conflict (e.g. a ride that
	// already has an active match).
	ErrConflict = errors.New("conflict")

	// ErrInvalidState marks a lifecycle transition the current status does not
	// allow (e.g. claiming a ride that is no longer available).
	ErrInvalidState = errors.New("invalid state")
)
