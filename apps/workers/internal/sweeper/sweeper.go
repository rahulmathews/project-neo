// Package sweeper expires stale rides so the feed only ever shows claimable
// ones. Before this, EXPIRED was unreachable and past rides sat AVAILABLE in
// the feed forever.
package sweeper

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"project-neo/shared/errtrack"
	"project-neo/shared/model"
	"project-neo/workers/internal/metrics"

	"github.com/uptrace/bun"
)

const (
	defaultInterval        = 5 * time.Minute
	defaultImmediateMin    = 60 // "now" rides expire this many minutes after posting
	defaultDepartedGrace   = 30 // scheduled rides linger this long past departure
	defaultUndatedHours    = 24 // rides with no time at all last a day
	expiryUpdateTimeoutSec = 30
)

// Start sweeps immediately and then on a ticker (RIDE_EXPIRY_SWEEP_INTERVAL,
// default 5m). Blocks until ctx is cancelled. Each expired row fires the
// existing rides_updated NOTIFY trigger, so subscribed feeds clear themselves.
func Start(ctx context.Context, db *bun.DB, m *metrics.Sweeper, logger *slog.Logger) {
	interval := durationEnv(logger)
	sweep(ctx, db, m, logger)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweep(ctx, db, m, logger)
		}
	}
}

func sweep(ctx context.Context, db *bun.DB, m *metrics.Sweeper, logger *slog.Logger) {
	immediateMin := intEnv("RIDE_EXPIRY_IMMEDIATE_MIN", defaultImmediateMin, logger)
	departedGrace := intEnv("RIDE_EXPIRY_DEPARTED_GRACE_MIN", defaultDepartedGrace, logger)
	undatedHours := intEnv("RIDE_EXPIRY_UNDATED_HOURS", defaultUndatedHours, logger)

	ctx, cancel := context.WithTimeout(ctx, expiryUpdateTimeoutSec*time.Second)
	defer cancel()

	res, err := db.NewUpdate().
		Model((*model.Ride)(nil)).
		Set("status = ?", model.RideStatusExpired).
		Set("updated_at = now()").
		Where("r.status = ?", model.RideStatusAvailable).
		Where(
			"((r.is_immediate AND r.created_at < now() - make_interval(mins => ?)) "+
				"OR (r.departure_time IS NOT NULL AND r.departure_time < now() - make_interval(mins => ?)) "+
				"OR (NOT r.is_immediate AND r.departure_time IS NULL AND r.created_at < now() - make_interval(hours => ?)))",
			immediateMin, departedGrace, undatedHours,
		).
		Exec(ctx)
	if err != nil {
		if ctx.Err() == nil {
			logger.Error("sweeper: expiry update failed", "error", err)
			errtrack.CaptureErr(err, map[string]string{"component": "ride_sweeper"})
		}
		return
	}

	rows, err := res.RowsAffected()
	if err != nil || rows == 0 {
		return
	}
	m.RidesExpired.Add(float64(rows))
	logger.Info("sweeper: expired stale rides", "count", rows)
}

func durationEnv(logger *slog.Logger) time.Duration {
	raw := os.Getenv("RIDE_EXPIRY_SWEEP_INTERVAL")
	if raw == "" {
		return defaultInterval
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		logger.Warn("sweeper: invalid RIDE_EXPIRY_SWEEP_INTERVAL, using default",
			"value", raw, "default", defaultInterval)
		return defaultInterval
	}
	return d
}

func intEnv(name string, def int, logger *slog.Logger) int {
	raw := os.Getenv(name)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		logger.Warn("sweeper: invalid env value, using default", "name", name, "value", raw, "default", def)
		return def
	}
	return n
}
