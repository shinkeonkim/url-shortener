package store

import "time"

type URL struct {
	Slug      string    `json:"slug"`
	TargetURL string    `json:"target_url"`
	CreatedAt time.Time `json:"created_at"`
	Clicks    int64     `json:"clicks"`
}

type Click struct {
	ID        int64     `json:"id"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	Referrer  string    `json:"referrer,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
}

type Stats struct {
	URL     URL      `json:"url"`
	Recent  []Click  `json:"recent_clicks"`
	Rollups []Rollup `json:"rollups"`
}

type Rollup struct {
	PeriodStart time.Time `json:"period_start"`
	Granularity string    `json:"granularity"`
	Clicks      int64     `json:"clicks"`
}

type StorageStats struct {
	URLs         int64
	TotalClicks  int64
	RawClicks    int64
	DailyRollups int64
	MonthRollups int64
}

type URLClickStat struct {
	Slug   string
	Clicks int64
}

type CompactResult struct {
	RawClicksRolledUp  int64
	DailyPeriodsRolled int64
}
