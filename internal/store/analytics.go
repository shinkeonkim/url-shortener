package store

import (
	"context"
	"database/sql"
	"time"
)

func (s *SQLite) Compact(ctx context.Context, now time.Time) (CompactResult, error) {
	dailyCutoff := dayStart(now.UTC()).AddDate(0, 0, -30)
	monthlyCutoff := dayStart(now.UTC()).AddDate(-1, 0, 0)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CompactResult{}, err
	}
	defer tx.Rollback()
	result := CompactResult{}
	if _, err = tx.ExecContext(ctx, `INSERT INTO click_rollups(slug,period_start,granularity,clicks)
 SELECT slug,substr(created_at,1,10)||'T00:00:00Z','day',COUNT(*) FROM clicks
 WHERE created_at < ? GROUP BY slug,substr(created_at,1,10)
 ON CONFLICT(slug,period_start,granularity) DO UPDATE SET clicks=clicks+excluded.clicks`, dailyCutoff.Format(time.RFC3339)); err != nil {
		return result, err
	}
	deleted, err := tx.ExecContext(ctx, `DELETE FROM clicks WHERE created_at < ?`, dailyCutoff.Format(time.RFC3339))
	if err != nil {
		return result, err
	}
	result.RawClicksRolledUp, _ = deleted.RowsAffected()
	if _, err = tx.ExecContext(ctx, `INSERT INTO click_rollups(slug,period_start,granularity,clicks)
 SELECT slug,substr(period_start,1,7)||'-01T00:00:00Z','month',SUM(clicks) FROM click_rollups
 WHERE granularity='day' AND period_start < ? GROUP BY slug,substr(period_start,1,7)
 ON CONFLICT(slug,period_start,granularity) DO UPDATE SET clicks=clicks+excluded.clicks`, monthlyCutoff.Format(time.RFC3339)); err != nil {
		return result, err
	}
	deleted, err = tx.ExecContext(ctx, `DELETE FROM click_rollups WHERE granularity='day' AND period_start < ?`, monthlyCutoff.Format(time.RFC3339))
	if err != nil {
		return result, err
	}
	result.DailyPeriodsRolled, _ = deleted.RowsAffected()
	return result, tx.Commit()
}

func (s *SQLite) rollups(ctx context.Context, slug string, limit int) ([]Rollup, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT period_start,granularity,clicks FROM click_rollups WHERE slug=? ORDER BY period_start DESC LIMIT ?`, slug, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Rollup{}
	for rows.Next() {
		var rollup Rollup
		var period string
		if err := rows.Scan(&period, &rollup.Granularity, &rollup.Clicks); err != nil {
			return nil, err
		}
		rollup.PeriodStart, err = time.Parse(time.RFC3339, period)
		if err != nil {
			return nil, err
		}
		result = append(result, rollup)
	}
	return result, rows.Err()
}

func (s *SQLite) Overview(ctx context.Context) (StorageStats, error) {
	var stats StorageStats
	err := s.db.QueryRowContext(ctx, `SELECT
	 (SELECT COUNT(*) FROM urls),
	 (SELECT COUNT(*) FROM clicks)+COALESCE((SELECT SUM(clicks) FROM click_rollups),0),(SELECT COUNT(*) FROM clicks),
	 (SELECT COUNT(*) FROM click_rollups WHERE granularity='day'),
	 (SELECT COUNT(*) FROM click_rollups WHERE granularity='month')`).Scan(&stats.URLs, &stats.TotalClicks, &stats.RawClicks, &stats.DailyRollups, &stats.MonthRollups)
	if err == sql.ErrNoRows {
		return StorageStats{}, nil
	}
	return stats, err
}

func (s *SQLite) URLClickStats(ctx context.Context) ([]URLClickStat, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT u.slug,
	 (SELECT COUNT(*) FROM clicks c WHERE c.slug=u.slug)+COALESCE((SELECT SUM(r.clicks) FROM click_rollups r WHERE r.slug=u.slug),0)
	 FROM urls u ORDER BY 2 DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []URLClickStat{}
	for rows.Next() {
		var item URLClickStat
		if err := rows.Scan(&item.Slug, &item.Clicks); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func dayStart(value time.Time) time.Time {
	y, m, d := value.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
