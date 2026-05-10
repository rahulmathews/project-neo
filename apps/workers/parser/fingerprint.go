package parser

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"project-neo/shared/model"
)

const rideFingerprintVersion = 1

var locationAliases = map[string]string{
	"lions gate":          "lionsgate",
	"liones gate":         "lionsgate",
	"liongate":            "lionsgate",
	"pointee":             "pointe",
	"pointie":             "pointe",
	"point royal":         "pointe",
	"pointe royal":        "pointe",
	"pointeroyal":         "pointe",
	"pointroyal":          "pointe",
	"pointe royale":       "pointe",
	"oakpark mall":        "oak park mall",
	"state line":          "stateline",
	"state line road":     "stateline",
	"st line road":        "stateline",
	"stateline rd":        "stateline",
	"stateline road":      "stateline",
	"overland park":       "op",
	"overlandpark":        "op",
	"overlandpark kansas": "op",
}

func rideSemanticFingerprint(msg *model.Message, parsed *ParsedRide) string {
	return buildRideFingerprint(msg, parsed, rideTimeKey(msg, parsed))
}

func legacyRideSemanticFingerprint(msg *model.Message, parsed *ParsedRide) string {
	if raw := rawTimeKey(msg); raw != "" && parsed.DepartureTime != nil {
		return buildRideFingerprint(msg, parsed, raw)
	}
	return ""
}

func buildRideFingerprint(msg *model.Message, parsed *ParsedRide, timeKey string) string {
	parts := []string{
		fmt.Sprintf("v%d", rideFingerprintVersion),
		string(parsed.RideType),
		canonicalLocationKey(parsed.FromLocationText),
		canonicalLocationKey(parsed.ToLocationText),
		timeKey,
		floatKey(parsed.Cost, 100),
		floatKey(parsed.Distance, 10),
		intKey(parsed.SeatsAvailable),
	}
	return strings.Join(parts, "|")
}

func canonicalLocationKey(s *string) string {
	if s == nil {
		return ""
	}
	cleaned := strings.ToLower(cleanLocationText(*s))
	cleaned = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return ' '
	}, cleaned)
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if alias, ok := locationAliases[cleaned]; ok {
		return alias
	}
	return cleaned
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

func rawTimeKey(msg *model.Message) string {
	raw := timeRe.FindString(normalizeParserContent(msg.Content))
	if raw == "" {
		return ""
	}
	return "raw:" + messageTime(msg).UTC().Format("20060102") + ":" + compactToken(raw)
}

func compactToken(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
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
