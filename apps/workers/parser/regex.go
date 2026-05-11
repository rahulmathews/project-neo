package parser

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"project-neo/shared/model"
)

var (
	requestIntentRe = regexp.MustCompile(`(?i)\b(?:need|looking\s+for)\b(?:\s+a)?(?:\s+shared)?\s+ride\b|\bneed\b.*\bfrom\b.*\bto\b`)
	offerIntentRe   = regexp.MustCompile(`(?i)\b(?:ride\s+available|available\s+ride|drop\s+available|available\s+from|to\s+and\s+fro\s+available)\b`)
	nonRideTopicRe  = regexp.MustCompile(`(?i)\b(?:accommod?ation|room|lease|rent|apartment|move\s+in|baby\s*sitter|sitter|cleaning|part\s*time)\b`)
	fromToRe        = regexp.MustCompile(`(?i)\bfrom\s+(.+?)\s+to\s+(.+?)(?:\n|$)`)
	directToRe      = regexp.MustCompile(`(?i)\bneed\s+(?:a\s+)?(?:shared\s+)?ride\s+(?:now\s+)?(?:for\s+\d+(?:\s+\w+)?\s+)?(.+?)\s+to\s+(.+?)(?:\n|$)`)
	nowRe           = regexp.MustCompile(`(?i)\bnow\b`)
	tomorrowRe      = regexp.MustCompile(`(?i)\b(tomorrow|tmr)\b`)
	timeRe          = regexp.MustCompile(`(?i)\b(\d{1,2})(?:(?:\s*[:.]\s*|\s+)(\d{2}))?\s*(am|pm|night)\b`)
	inTimeRe        = regexp.MustCompile(`(?i)\bin\s+\d+\s*(?:min|mins|minutes|hour|hours|hr|hrs)\b`)
	costRe          = regexp.MustCompile(`(?i)(?:[$₹£€&])\s*(\d+(?:\.\d{1,2})?)|(\d+(?:\.\d{1,2})?)\s*(?:USD|INR|GBP|EUR)|(\d+(?:\.\d{1,2})?)\s*[$₹£€&]`)
	distanceRe      = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(?:km|mils?|miles?|mi)\b`)
	seatsRe         = regexp.MustCompile(`(?i)\bfor\s+(\d+)\s*(?:ppl|people|members?|mem|persons?|passengers?)\b`)
	shortSeatsRe    = regexp.MustCompile(`(?i)\bfor\s+(\d+)\b`)
	leadingSeatsRe  = regexp.MustCompile(`(?i)\b(\d+)\s*(?:ppl|people|members?|mem|persons?|passengers?)\b`)
	markdownRe      = regexp.MustCompile(`[*_~` + "`" + `]+`)
	typoFromRe      = regexp.MustCompile(`(?i)\b(?:form|frm|frok|ftom)\b`)
	typoToRe        = regexp.MustCompile(`(?i)\bti\b`)
	locationTrailRe = regexp.MustCompile(`(?i)\s+(?:` +
		`at\s+\d{1,2}\s*[:.]\s*\d{2}\s*(?:am|pm)?` + // "at 3:30pm", "at 10 :10pm"
		`|at\s+\d{1,2}\s*(?:am|pm|night)\b` + // "at 3pm", "at 6 AM"
		`|at\s+\d{1,2}\b` + // "at 10"
		`|at\s+\d{1,2}\s+\d{2}\s*(?:am|pm|night)` + // "at 2 40pm" (malformed time with space)
		`|\d{1,2}\s*[:.]\s*\d{2}\s*(?:am|pm|night)?` + // time without "at"
		`|\d{1,2}\s+\d{2}\s*(?:am|pm|night)` + // "6 45am"
		`|\d{1,2}\s*(?:am|pm|night)\b` + // "3pm"
		`|[@([]` + // "@", "(", "["
		`|\d+(?:\.\d+)?\s*(?:km|mils?|miles?|mi|mile)\b` + // distance
		`|[$₹£€]\d` + // currency prefix
		`|\d+\s*[$₹£€&]` + // currency suffix
		`|(?:now|today|tomorrow|tmr|tonight|yesterday)\b` + // temporal keywords
		`|on\s+(?:monday|tuesday|wednesday|thursday|friday|saturday|sunday|january|february|march|april|may|june|july|august|september|october|november|december|\d{1,2})` + // date phrase
		`|for\s+\d+\s*(?:ppl|people|members?|persons?|passengers?)\b` + // seat phrasing
		`|for\s+\d+\b` + // terse seat phrasing, e.g. "for 2"
		`|(?:anyone|going|available|open|free|ok|interested|still)\?` + // conversational trailers
		`).*$`)
)

func normalizeParserContent(content string) string {
	content = markdownRe.ReplaceAllString(content, "")
	content = typoFromRe.ReplaceAllString(content, "from")
	content = typoToRe.ReplaceAllString(content, "to")
	return model.NormalizeMessageContent(content)
}

func isClearlyNonRide(content string) bool {
	content = normalizeParserContent(content)
	if !nonRideTopicRe.MatchString(content) {
		return false
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "ride") && !strings.Contains(lower, "drop") && !strings.Contains(lower, "to and fro") {
		return true
	}
	return !requestIntentRe.MatchString(content) && !offerIntentRe.MatchString(content)
}

// cleanLocationText strips trailing time/cost/distance metadata absorbed into a location match.
// e.g. "Royal Spices at 9:55am (2.5 miles $5)" → "Royal Spices".
func cleanLocationText(s string) string {
	return strings.TrimSpace(locationTrailRe.ReplaceAllString(s, ""))
}

func cleanOptionalLocationText(s *string) *string {
	if s == nil {
		return nil
	}
	cleaned := cleanLocationText(*s)
	if cleaned == "" {
		return nil
	}
	return &cleaned
}

// parseDepartureTime populates IsImmediate and DepartureTime on parsed from content.
func parseDepartureTime(content string, messageTime time.Time, parsed *ParsedRide) {
	if nowRe.MatchString(content) {
		parsed.IsImmediate = true
		return
	}
	if inTimeRe.MatchString(content) {
		// Relative time ("in 30 mins") — no absolute DepartureTime stored
		return
	}
	clock, ok := parseClockTime(timeRe.FindStringSubmatch(content))
	if !ok {
		return
	}

	base := messageTimeOrNow(messageTime)
	dep := time.Date(base.Year(), base.Month(), base.Day(), clock.hour, clock.minute, 0, 0, base.Location())
	if tomorrowRe.MatchString(content) {
		dep = dep.AddDate(0, 0, 1)
	}
	parsed.DepartureTime = &dep
}

type parsedClockTime struct {
	hour   int
	minute int
}

func parseClockTime(m []string) (parsedClockTime, bool) {
	if len(m) != 4 {
		return parsedClockTime{}, false
	}
	hour, err := strconv.Atoi(m[1])
	if err != nil || hour < 1 || hour > 12 {
		return parsedClockTime{}, false
	}
	minute, ok := parseClockMinute(m[2])
	if !ok {
		return parsedClockTime{}, false
	}
	return parsedClockTime{hour: normalizeClockHour(hour, strings.ToLower(m[3])), minute: minute}, true
}

func parseClockMinute(raw string) (int, bool) {
	if raw == "" {
		return 0, true
	}
	minute, err := strconv.Atoi(raw)
	if err != nil || minute > 59 {
		return 0, false
	}
	return minute, true
}

func normalizeClockHour(hour int, period string) int {
	switch period {
	case "pm":
		if hour != 12 {
			return hour + 12
		}
	case "am":
		if hour == 12 {
			return 0
		}
	case "night":
		return normalizeNightHour(hour)
	}
	return hour
}

func normalizeNightHour(hour int) int {
	switch {
	case hour == 12:
		return 0
	case hour >= 6:
		return hour + 12
	default:
		return hour
	}
}

func messageTimeOrNow(messageTime time.Time) time.Time {
	if messageTime.IsZero() {
		return time.Now()
	}
	return messageTime
}

// extractWithRegex attempts structured extraction from content.
// Returns (parsed, true) on a hit (ride type + at least one location found),
// or (nil, false) on a miss.
func extractWithRegex(content string, messageTime time.Time) (*ParsedRide, bool) {
	content = normalizeParserContent(content)
	parsed := &ParsedRide{}

	if offerIntentRe.MatchString(content) {
		parsed.RideType = model.RideTypeRideAvailable
	} else if requestIntentRe.MatchString(content) {
		parsed.RideType = model.RideTypeNeedRide
	}

	if parsed.RideType == "" {
		return nil, false
	}

	if from, to, ok := extractRoute(content); ok {
		parsed.FromLocationText = cleanOptionalLocationText(&from)
		parsed.ToLocationText = cleanOptionalLocationText(&to)
	}

	// Departure time
	parseDepartureTime(content, messageTime, parsed)

	// Cost
	if m := costRe.FindStringSubmatch(content); m != nil {
		raw := m[1]
		if raw == "" {
			raw = m[2]
		}
		if raw == "" {
			raw = m[3]
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

	parseSeats(content, parsed)

	// Hit = ride type AND both route endpoints.
	hit := parsed.RideType != "" &&
		parsed.FromLocationText != nil && parsed.ToLocationText != nil
	return parsed, hit
}

func extractRoute(content string) (string, string, bool) {
	if m := fromToRe.FindStringSubmatch(content); len(m) == 3 {
		return m[1], m[2], true
	}
	if m := directToRe.FindStringSubmatch(content); len(m) == 3 {
		from := strings.TrimSpace(m[1])
		if !strings.EqualFold(from, "from") {
			return from, m[2], true
		}
	}
	return "", "", false
}

func parseSeats(content string, parsed *ParsedRide) {
	for _, re := range []*regexp.Regexp{seatsRe, leadingSeatsRe, shortSeatsRe} {
		m := re.FindStringSubmatch(content)
		if len(m) < 2 {
			continue
		}
		v, err := strconv.Atoi(m[1])
		if err != nil || v < 1 || v > 8 {
			continue
		}
		parsed.SeatsAvailable = &v
		return
	}
}
