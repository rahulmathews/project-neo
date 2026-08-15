// Package errtrack wraps optional Sentry reporting. Every Capture* call is a
// no-op unless Init found a SENTRY_DSN, so services run identically with
// error tracking on or off (local-first: nothing is sent unless configured).
package errtrack

import (
	"log/slog"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

// Init configures Sentry when SENTRY_DSN is set. Returns a flush func to call
// on shutdown (always non-nil, no-op when disabled).
func Init(service string, logger *slog.Logger) func() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		return func() {}
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:         dsn,
		Environment: environment(),
		ServerName:  service,
	})
	if err != nil {
		logger.Warn("sentry init failed — error tracking disabled", "error", err)
		return func() {}
	}
	logger.Info("sentry error tracking enabled", "service", service)
	return func() { sentry.Flush(2 * time.Second) }
}

func environment() string {
	if env := os.Getenv("ENV"); env != "" {
		return env
	}
	return "development"
}

// CaptureErr reports err with optional tags. No-op when disabled or err is nil.
func CaptureErr(err error, tags map[string]string) {
	if err == nil {
		return
	}
	hub := sentry.CurrentHub()
	hub.WithScope(func(scope *sentry.Scope) {
		if len(tags) > 0 {
			scope.SetTags(tags)
		}
		hub.CaptureException(err)
	})
}

// CapturePanic reports a recovered panic value with optional tags.
// No-op when disabled.
func CapturePanic(p any, tags map[string]string) {
	hub := sentry.CurrentHub()
	hub.WithScope(func(scope *sentry.Scope) {
		if len(tags) > 0 {
			scope.SetTags(tags)
		}
		hub.Recover(p)
	})
}
