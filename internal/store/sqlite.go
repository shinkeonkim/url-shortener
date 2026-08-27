package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("url not found")
var ErrConflict = errors.New("slug already exists")

type SQLite struct{ db *sql.DB }

func Open(path string) (*SQLite, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &SQLite{db: db}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLite) Close() error { return s.db.Close() }

func (s *SQLite) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;
CREATE TABLE IF NOT EXISTS urls (
 slug TEXT PRIMARY KEY, target_url TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS clicks (
 id INTEGER PRIMARY KEY AUTOINCREMENT, slug TEXT NOT NULL REFERENCES urls(slug) ON DELETE CASCADE,
 created_at TEXT NOT NULL, referrer TEXT NOT NULL DEFAULT '', user_agent TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_clicks_slug_created ON clicks(slug, created_at DESC);`)
	return err
}

func (s *SQLite) CreateURL(ctx context.Context, slug, target string) (URL, error) {
	now := time.Now().UTC().Truncate(time.Second)
	_, err := s.db.ExecContext(ctx, `INSERT INTO urls(slug,target_url,created_at) VALUES(?,?,?)`, slug, target, now.Format(time.RFC3339))
	if err != nil {
		if isConstraint(err) {
			return URL{}, ErrConflict
		}
		return URL{}, err
	}
	return URL{Slug: slug, TargetURL: target, CreatedAt: now}, nil
}

func (s *SQLite) GetURL(ctx context.Context, slug string) (URL, error) {
	var u URL
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT u.slug,u.target_url,u.created_at,COUNT(c.id) FROM urls u LEFT JOIN clicks c ON c.slug=u.slug WHERE u.slug=? GROUP BY u.slug`, slug).Scan(&u.Slug, &u.TargetURL, &created, &u.Clicks)
	if errors.Is(err, sql.ErrNoRows) {
		return URL{}, ErrNotFound
	}
	if err != nil {
		return URL{}, err
	}
	u.CreatedAt, err = time.Parse(time.RFC3339, created)
	return u, err
}

func (s *SQLite) DeleteURL(ctx context.Context, slug string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM urls WHERE slug=?`, slug)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrNotFound
	}
	return err
}

func (s *SQLite) RecordClick(ctx context.Context, slug, referrer, userAgent string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO clicks(slug,created_at,referrer,user_agent) VALUES(?,?,?,?)`,
		slug, time.Now().UTC().Format(time.RFC3339), referrer, userAgent)
	if isConstraint(err) {
		return ErrNotFound
	}
	return err
}

func (s *SQLite) Stats(ctx context.Context, slug string, limit int) (Stats, error) {
	u, err := s.GetURL(ctx, slug)
	if err != nil {
		return Stats{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,slug,created_at,referrer,user_agent FROM clicks WHERE slug=? ORDER BY id DESC LIMIT ?`, slug, limit)
	if err != nil {
		return Stats{}, err
	}
	defer rows.Close()
	stats := Stats{URL: u, Recent: []Click{}}
	for rows.Next() {
		var c Click
		var created string
		if err := rows.Scan(&c.ID, &c.Slug, &created, &c.Referrer, &c.UserAgent); err != nil {
			return Stats{}, err
		}
		c.CreatedAt, err = time.Parse(time.RFC3339, created)
		if err != nil {
			return Stats{}, err
		}
		stats.Recent = append(stats.Recent, c)
	}
	return stats, rows.Err()
}

func isConstraint(err error) bool {
	return err != nil && (contains(err.Error(), "constraint") || contains(err.Error(), "UNIQUE"))
}
func contains(s, part string) bool {
	for i := 0; i+len(part) <= len(s); i++ {
		if s[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
