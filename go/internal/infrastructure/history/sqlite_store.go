package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/fayupable/pgscope/internal/domain"
)

// SQLiteStore implements output.IHistoryStorePort backed by a single local
// SQLite file. Unlike RingBufferStore, history survives a process restart.
// Query text is redacted (redactQueryLiterals) before it's ever written to
// disk, since pg_stat_activity reports queries with their actual bound
// values, which may include sensitive data.
//
// This type's methods are split across three files by responsibility:
//   - sqlite_store.go (this file): setup and writes (NewSQLiteStore, Close,
//     Append)
//   - sqlite_store_query.go: reads (Recent, Incidents, downsampling)
//   - sqlite_store_maintenance.go: pruning and disk-size enforcement
type SQLiteStore struct {
	db   *sql.DB
	path string
}

// NewSQLiteStore opens (creating if necessary) the SQLite file at path and
// ensures its schema exists. SQLite does not handle concurrent writers
// well, so the connection pool is capped at one connection — acceptable
// at this project's write volume (one snapshot per poll interval).
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("create history db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open history db: %w", err)
	}
	db.SetMaxOpenConns(1)

	// SQLite doesn't shrink the file on DELETE by default — deleted rows
	// just become free pages reused later, so the file's on-disk size never
	// drops on its own. Incremental auto-vacuum (only takes full effect on
	// a database with no tables yet, i.e. a fresh file) lets EnforceMaxSize
	// actually reclaim that space via PRAGMA incremental_vacuum after a
	// prune, instead of only ever freeing pages internally.
	//
	// This MUST run before journal_mode=WAL below — empirically verified
	// that setting journal_mode first silently prevents auto_vacuum from
	// ever taking effect (auto_vacuum stays reported as NONE regardless),
	// even though the database still has no tables at that point. Swapping
	// the order fixes it; this isn't documented anywhere obvious, so don't
	// reorder these two without re-verifying.
	if _, err := db.Exec(`PRAGMA auto_vacuum=INCREMENTAL`); err != nil {
		return nil, fmt.Errorf("set auto_vacuum mode: %w", err)
	}

	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	// A single connection serializes every read and write; without a busy
	// timeout, an Append racing a long-running prune batch would fail
	// immediately with "database is locked" instead of just waiting its turn.
	if _, err := db.Exec(`PRAGMA busy_timeout=5000`); err != nil {
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			captured_at TIMESTAMP NOT NULL,
			trigger TEXT NOT NULL,
			payload TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_snapshots_trigger_captured ON snapshots(trigger, captured_at);
	`); err != nil {
		return nil, fmt.Errorf("create history schema: %w", err)
	}

	return &SQLiteStore{db: db, path: path}, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) Append(ctx context.Context, snapshot domain.Snapshot) error {
	redacted := redactSnapshot(snapshot)

	payload, err := json.Marshal(redacted)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO snapshots (captured_at, trigger, payload) VALUES (?, ?, ?)`,
		redacted.CapturedAt, string(redacted.Trigger), string(payload),
	)
	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}
	return nil
}

// redactSnapshot returns a copy of snapshot with every session's Query
// field redacted, leaving the original (used live, in memory) untouched.
func redactSnapshot(snapshot domain.Snapshot) domain.Snapshot {
	redactedSessions := make([]domain.Session, len(snapshot.Sessions))
	for i, session := range snapshot.Sessions {
		session.Query = redactQueryLiterals(session.Query)
		redactedSessions[i] = session
	}

	return domain.Snapshot{
		Sessions:   redactedSessions,
		CapturedAt: snapshot.CapturedAt,
		Trigger:    snapshot.Trigger,
	}
}
