package validation

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"project-neo/shared/model"
)

const (
	maxNameLen        = 200
	maxDescriptionLen = 2000
	maxLocationLen    = 500
	maxAliasLen       = 100
	maxCurrencyLen    = 5
	minCurrencyLen    = 3
	maxCost           = 1_000_000
	maxDistance       = 100_000 // km
	maxSeats          = 50
)

// loose international phone — allows +, digits, spaces, dashes, parens.
var phoneRe = regexp.MustCompile(`^\+?[0-9 \-()]{6,20}$`)

// FieldError carries a per-field validation failure.
type FieldError struct {
	Field   string
	Message string
}

func (e FieldError) Error() string {
	return fmt.Sprintf("invalid input: %s: %s", e.Field, e.Message)
}

// Errors collects multiple FieldError values; satisfies error when non-empty.
type Errors []FieldError

func (e Errors) Error() string {
	if len(e) == 0 {
		return ""
	}
	parts := make([]string, len(e))
	for i, fe := range e {
		parts[i] = fe.Error()
	}
	return strings.Join(parts, "; ")
}

func (e Errors) OrNil() error {
	if len(e) == 0 {
		return nil
	}
	return e
}

func (e *Errors) add(field, msg string) {
	*e = append(*e, FieldError{Field: field, Message: msg})
}

// ValidateUpsertUserInput enforces bounds on UpsertUserInput.
func ValidateUpsertUserInput(in model.UpsertUserInput) error {
	var errs Errors
	if name := strings.TrimSpace(in.Name); name == "" {
		errs.add("name", "is required")
	} else if len(name) > maxNameLen {
		errs.add("name", fmt.Sprintf("max %d characters", maxNameLen))
	}
	if in.Phone != nil {
		p := strings.TrimSpace(*in.Phone)
		if p != "" && !phoneRe.MatchString(p) {
			errs.add("phone", "must be 6-20 digits, may start with +")
		}
	}
	if in.AvatarURL != nil {
		if a := strings.TrimSpace(*in.AvatarURL); a != "" {
			u, err := url.Parse(a)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				errs.add("avatarUrl", "must be a valid http(s) URL")
			}
		}
	}
	return errs.OrNil()
}

// ValidateCreateRideInput enforces bounds and cross-field rules on CreateRideInput.
func ValidateCreateRideInput(in model.CreateRideInput) error {
	var errs Errors
	validateRideLocations(&errs, in)
	validateRideTiming(&errs, in)
	validateRidePricing(&errs, in.Cost, in.Distance, in.SeatsAvailable, in.Currency)
	return errs.OrNil()
}

func validateRideLocations(errs *Errors, in model.CreateRideInput) {
	hasFrom := in.FromLocationContextID != nil ||
		(in.FromLocationText != nil && strings.TrimSpace(*in.FromLocationText) != "")
	hasTo := in.ToLocationContextID != nil ||
		(in.ToLocationText != nil && strings.TrimSpace(*in.ToLocationText) != "")
	if !hasFrom {
		errs.add("from", "either fromLocationContextId or fromLocationText is required")
	}
	if !hasTo {
		errs.add("to", "either toLocationContextId or toLocationText is required")
	}
	if in.FromLocationText != nil && len(*in.FromLocationText) > maxLocationLen {
		errs.add("fromLocationText", fmt.Sprintf("max %d characters", maxLocationLen))
	}
	if in.ToLocationText != nil && len(*in.ToLocationText) > maxLocationLen {
		errs.add("toLocationText", fmt.Sprintf("max %d characters", maxLocationLen))
	}
}

func validateRideTiming(errs *Errors, in model.CreateRideInput) {
	if !in.IsImmediate && in.DepartureTime == nil {
		errs.add("departureTime", "required when isImmediate is false")
	}
}

func validateRidePricing(errs *Errors, cost, distance *float64, seats *int, currency *string) {
	if cost != nil && (*cost < 0 || *cost > maxCost) {
		errs.add("cost", fmt.Sprintf("must be 0..%d", maxCost))
	}
	if distance != nil && (*distance < 0 || *distance > maxDistance) {
		errs.add("distance", fmt.Sprintf("must be 0..%d km", maxDistance))
	}
	if seats != nil && (*seats < 1 || *seats > maxSeats) {
		errs.add("seatsAvailable", fmt.Sprintf("must be 1..%d", maxSeats))
	}
	if currency != nil {
		c := strings.TrimSpace(*currency)
		if c != "" && (len(c) < minCurrencyLen || len(c) > maxCurrencyLen) {
			errs.add("currency", fmt.Sprintf("must be %d..%d characters", minCurrencyLen, maxCurrencyLen))
		}
	}
}

// ValidateUpdateRideInput enforces bounds on UpdateRideInput.
func ValidateUpdateRideInput(in model.UpdateRideInput) error {
	var errs Errors
	if in.Cost != nil && (*in.Cost < 0 || *in.Cost > maxCost) {
		errs.add("cost", fmt.Sprintf("must be 0..%d", maxCost))
	}
	if in.SeatsAvailable != nil && (*in.SeatsAvailable < 1 || *in.SeatsAvailable > maxSeats) {
		errs.add("seatsAvailable", fmt.Sprintf("must be 1..%d", maxSeats))
	}
	return errs.OrNil()
}

// ValidateCreateGroupInput enforces bounds on CreateGroupInput.
func ValidateCreateGroupInput(in model.CreateGroupInput) error {
	var errs Errors
	if name := strings.TrimSpace(in.Name); name == "" {
		errs.add("name", "is required")
	} else if len(name) > maxNameLen {
		errs.add("name", fmt.Sprintf("max %d characters", maxNameLen))
	}
	if in.Description != nil && len(*in.Description) > maxDescriptionLen {
		errs.add("description", fmt.Sprintf("max %d characters", maxDescriptionLen))
	}
	return errs.OrNil()
}

// ValidateUpsertLocationContextInput enforces bounds on UpsertLocationContextInput.
func ValidateUpsertLocationContextInput(in model.UpsertLocationContextInput) error {
	var errs Errors
	if alias := strings.TrimSpace(in.LocationAlias); alias == "" {
		errs.add("locationAlias", "is required")
	} else if len(alias) > maxAliasLen {
		errs.add("locationAlias", fmt.Sprintf("max %d characters", maxAliasLen))
	}
	if name := strings.TrimSpace(in.LocationName); name == "" {
		errs.add("locationName", "is required")
	} else if len(name) > maxNameLen {
		errs.add("locationName", fmt.Sprintf("max %d characters", maxNameLen))
	}
	return errs.OrNil()
}
