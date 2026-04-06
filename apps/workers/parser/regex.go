package parser

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"project-neo/shared/model"
)

var (
	rideTypeRe      = regexp.MustCompile(`(?i)\b(need\s+ride|ride\s+available)\b`)
	fromToRe        = regexp.MustCompile(`(?i)from\s+(.+?)\s+to\s+(.+?)(?:\n|$)`)
	nowRe           = regexp.MustCompile(`(?i)\bnow\b`)
	timeRe          = regexp.MustCompile(`(?i)\b(\d{1,2}[:.]\d{2}\s*(?:AM|PM))\b`)
	inTimeRe        = regexp.MustCompile(`(?i)\bin\s+\d+\s*(?:min|mins|minutes|hour|hours|hr|hrs)\b`)
	costRe          = regexp.MustCompile(`(?i)(?:[$₹£€])(\d+(?:\.\d{1,2})?)|(\d+(?:\.\d{1,2})?)\s*(?:USD|INR|GBP|EUR)`)
	distanceRe      = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(?:km|miles|mi)\b`)
	locationTrailRe = regexp.MustCompile(`(?i)\s+(?:` +
		`at\s+\d+[:.]\d+` + // "at 3:30pm" or "at 3.30pm"
		`|at\s+\d{1,2}\s+\d{2}\s*(?:am|pm)` + // "at 2 40pm" (malformed time with space)
		`|[@([]` + // "@", "(", "["
		`|\d+(?:\.\d+)?\s*(?:km|miles|mi|mile)\b` + // distance (added "mile" singular)
		`|[$₹£€]\d` + // currency prefix
		`|(?:today|tomorrow|tonight|yesterday)\b` + // temporal keywords
		`|on\s+(?:monday|tuesday|wednesday|thursday|friday|saturday|sunday|\d)` + // date phrase
		`|for\s+\d+\s*(?:ppl|people|members?|persons?|passengers?)\b` + // seat phrasing
		`|(?:anyone|going|available|open|free|ok|interested|still)\?` + // conversational trailers
		`).*$`)
)

// cleanLocationText strips trailing time/cost/distance metadata absorbed into a location match.
// e.g. "Royal Spices at 9:55am (2.5 miles $5)" → "Royal Spices"
func cleanLocationText(s string) string {
	return strings.TrimSpace(locationTrailRe.ReplaceAllString(s, ""))
}

// parseDepartureTime populates IsImmediate and DepartureTime on parsed from content.
func parseDepartureTime(content string, parsed *ParsedRide) {
	if nowRe.MatchString(content) {
		parsed.IsImmediate = true
		return
	}
	if inTimeRe.MatchString(content) {
		// Relative time ("in 30 mins") — no absolute DepartureTime stored
		return
	}
	m := timeRe.FindString(content)
	if m == "" {
		return
	}
	normalized := strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(m)), ".", ":")
	for _, layout := range []string{"3:04 PM", "3:04PM"} {
		if t, err := time.Parse(layout, normalized); err == nil {
			now := time.Now()
			dep := time.Date(now.Year(), now.Month(), now.Day(),
				t.Hour(), t.Minute(), 0, 0, now.Location())
			parsed.DepartureTime = &dep
			break
		}
	}
}

// extractWithRegex attempts structured extraction from content.
// Returns (parsed, true) on a hit (ride type + at least one location found),
// or (nil, false) on a miss.
func extractWithRegex(content string) (*ParsedRide, bool) {
	parsed := &ParsedRide{}

	// Ride type
	if m := rideTypeRe.FindString(content); m != "" {
		if strings.Contains(strings.ToLower(m), "need") {
			parsed.RideType = model.RideTypeNeedRide
		} else {
			parsed.RideType = model.RideTypeRideAvailable
		}
	}

	// From / To locations
	if m := fromToRe.FindStringSubmatch(content); len(m) == 3 {
		from := cleanLocationText(m[1])
		to := cleanLocationText(m[2])
		parsed.FromLocationText = &from
		parsed.ToLocationText = &to
	}

	// Departure time
	parseDepartureTime(content, parsed)

	// Cost
	if m := costRe.FindStringSubmatch(content); m != nil {
		raw := m[1]
		if raw == "" {
			raw = m[2]
		}
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			parsed.Cost = &v
		}
	}

	// Distance
	if m := distanceRe.FindStringSubmatch(content); len(m) >= 2 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			parsed.Distance = &v
		}
	}

	// Hit = ride type AND at least one location
	hit := parsed.RideType != "" &&
		(parsed.FromLocationText != nil || parsed.ToLocationText != nil)
	return parsed, hit
}
