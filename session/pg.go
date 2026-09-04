package session

import (
	"database/sql"
	"encoding/json"
	"errors"

	_ "github.com/lib/pq"
)

// PostgresStore is a durable Store backed by PostgreSQL — Agent.md §20's
// planned long-term store. Each learner's session snapshot is stored as one
// JSONB row, so a session survives process restarts exactly as FileStore does,
// but in a real database. It implements the same Store contract as MemoryStore
// and FileStore, so the tutor, API, and web client are unchanged.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore opens and pings the database at dsn, ensuring the sessions
// table exists.
func NewPostgresStore(dsn string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS sessions (
		learner    TEXT PRIMARY KEY,
		snapshot   JSONB NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &PostgresStore{db: db}, nil
}

// Close closes the underlying database connection.
func (p *PostgresStore) Close() error {
	return p.db.Close()
}

// Save upserts the session's authoritative state under its learner name.
func (p *PostgresStore) Save(s *Session) error {
	data, err := json.Marshal(captureSnapshot(s))
	if err != nil {
		return err
	}
	_, err = p.db.Exec(
		`INSERT INTO sessions (learner, snapshot) VALUES ($1, $2)
		 ON CONFLICT (learner) DO UPDATE SET snapshot = EXCLUDED.snapshot, updated_at = now()`,
		s.LearnerName(), data)
	return err
}

// Load returns a fresh Session rebuilt from the stored snapshot for name, or
// ErrSessionNotFound if the learner has no row.
func (p *PostgresStore) Load(name string) (*Session, error) {
	var data []byte
	err := p.db.QueryRow(`SELECT snapshot FROM sessions WHERE learner = $1`, name).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	var snap sessionSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	return restoreSnapshot(snap), nil
}
