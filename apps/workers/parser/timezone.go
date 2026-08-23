package parser

import (
	"log/slog"
	"os"
	"time"
)

// parserLoc is the location used to interpret bare clock times in messages
// ("9am"). Groups post in local time; building the departure in the message
// timestamp's location (UTC from whatsmeow/Postgres) shifted every ride.
//
//nolint:gochecknoglobals // written once by ConfigureTimezone at startup, before any parse goroutine reads it
var parserLoc = time.UTC

// ConfigureTimezone resolves PARSER_TIMEZONE (IANA name, e.g.
// "America/Chicago") into parserLoc. Call once at startup, before the
// listener and recovery goroutines begin parsing.
func ConfigureTimezone(logger *slog.Logger) {
	name := os.Getenv("PARSER_TIMEZONE")
	if name == "" {
		return
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		logger.Warn("parser: invalid PARSER_TIMEZONE, using UTC", "value", name, "error", err)
		return
	}
	parserLoc = loc
	logger.Info("parser timezone configured", "timezone", name)
}
