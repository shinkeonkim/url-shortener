package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/shinkeonkim/url-shortener/internal/store"
)

func maintain(ctx context.Context, database *store.SQLite) {
	runCompaction(ctx, database)
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runCompaction(ctx, database)
		}
	}
}

func runCompaction(ctx context.Context, database *store.SQLite) {
	result, err := database.Compact(ctx, time.Now())
	if err != nil {
		slog.Error("analytics compaction failed", "error", err)
		return
	}
	if result.RawClicksRolledUp > 0 || result.DailyPeriodsRolled > 0 {
		slog.Info("analytics compacted", "raw_clicks", result.RawClicksRolledUp, "daily_periods", result.DailyPeriodsRolled)
	}
}
