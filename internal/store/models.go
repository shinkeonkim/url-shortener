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
	URL    URL     `json:"url"`
	Recent []Click `json:"recent_clicks"`
}
