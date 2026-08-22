package parser

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"project-neo/shared/model"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const rideFingerprintVersion = 1

// rideSemanticFingerprint computes the deduplication key for a ride.
// Same shape as before, but location keys come from location_contexts
// (alias_normalized → location_id) instead of a hardcoded synonym map.
func rideSemanticFingerprint(ctx context.Context, db *bun.DB, msg *model.Message, parsed *ParsedRide) string {
	parts := []string{
		fmt.Sprintf("v%d", rideFingerprintVersion),
		string(parsed.RideType),
		canonicalLocationKey(ctx, db, msg.GroupID, parsed.FromLocationText),
		canonicalLocationKey(ctx, db, msg.GroupID, parsed.ToLocationText),
		rideTimeKey(msg, parsed),
		floatKey(parsed.Cost, 100),
		floatKey(parsed.Distance, 10),
		intKey(parsed.SeatsAvailable),
	}
	return strings.Join(parts, "|")
}

// canonicalLocationKey resolves a free-form location string to a stable key.
// Lookup order:
//  1. Exact match on (group_id, alias_normalized) in location_contexts.
//     Hit + location_id present → "loc:<uuid>" (most authoritative — collapses
//     all aliases sharing a canonical Location row).
//     Hit + location_id absent  → "name:<lowercased location_name>".
//  2. Miss → fall back to the normalized text itself.
func canonicalLocationKey(ctx context.Context, db *bun.DB, groupID uuid.UUID, s *string) string {
	if s == nil || *s == "" {
		return ""
	}
	normalized := normalizeAliasKey(*s)
	if normalized == "" {
		return ""
	}

	var lc model.LocationContext
	err := db.NewSelect().
		Model(&lc).
		Column("location_id", "location_name").
		Where("group_id = ?", groupID).
		Where("alias_normalized = ?", normalized).
		Limit(1).
		Scan(ctx)
	if err != nil {
		// sql.ErrNoRows is the expected miss; other errors fall back too,
		// preferring availability over correctness for an opaque key.
		_ = errors.Is(err, sql.ErrNoRows)
		return normalized
	}

	if lc.LocationID != nil {
		return "loc:" + lc.LocationID.String()
	}
	return "name:" + normalizeAliasKey(lc.LocationName)
}

// normalizeAliasKey mirrors the SQL generated column on location_contexts:
// strip trailing temporal/seat/distance noise, lowercase, drop non-alphanumeric.
func normalizeAliasKey(s string) string {
	cleaned := strings.ToLower(cleanLocationText(s))
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, cleaned)
}

func rideTimeKey(msg *model.Message, parsed *ParsedRide) string {
	if parsed.DepartureTime != nil {
		return "at:" + parsed.DepartureTime.UTC().Format("200601021504")
	}
	if parsed.IsImmediate {
		return "now:" + messageTime(msg).UTC().Format("2006010215")
	}

	tmp := &ParsedRide{}
	parseDepartureTime(msg.Content, messageTime(msg), tmp)
	if tmp.DepartureTime != nil {
		return "at:" + tmp.DepartureTime.UTC().Format("200601021504")
	}
	if tmp.IsImmediate {
		return "now:" + messageTime(msg).UTC().Format("2006010215")
	}

	return "day:" + messageTime(msg).UTC().Format("20060102")
}

func messageTime(msg *model.Message) time.Time {
	if msg.Timestamp.IsZero() {
		return time.Now()
	}
	return msg.Timestamp
}

func floatKey(v *float64, multiplier float64) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%.0f", math.Round(*v*multiplier))
}

func intKey(v *int) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%d", *v)
}
