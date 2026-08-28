package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestCompactDailyAndMonthlyRollups(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "analytics.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	if _, err := s.CreateURL(ctx, "history", "https://example.com"); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	insertClick(t, s, "history", now.AddDate(0, 0, -40))
	insertClick(t, s, "history", now.AddDate(-2, 0, 0))
	insertClick(t, s, "history", now.AddDate(0, 0, -2))
	result, err := s.Compact(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if result.RawClicksRolledUp != 2 || result.DailyPeriodsRolled != 1 {
		t.Fatalf("result = %#v", result)
	}
	stats, err := s.Stats(ctx, "history", 100)
	if err != nil {
		t.Fatal(err)
	}
	if stats.URL.Clicks != 3 || len(stats.Recent) != 1 || len(stats.Rollups) != 2 {
		t.Fatalf("stats = %#v", stats)
	}
	if _, err := s.Compact(ctx, now); err != nil {
		t.Fatal(err)
	}
	again, _ := s.GetURL(ctx, "history")
	if again.Clicks != 3 {
		t.Fatalf("idempotent clicks = %d", again.Clicks)
	}
}

func insertClick(t *testing.T, s *SQLite, slug string, at time.Time) {
	t.Helper()
	if _, err := s.db.Exec(`INSERT INTO clicks(slug,created_at) VALUES(?,?)`, slug, at.Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
}
