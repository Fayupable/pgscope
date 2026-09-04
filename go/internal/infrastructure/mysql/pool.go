package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type PoolConfig struct {
	DSN string
}

// NewPool opens a MySQL connection pool. Read-only enforcement is expected
// at the database user's grant level (SELECT-only privileges) — the same
// approach the root README documents for the Postgres pgscope_agent role.
// MySQL has no session-wide read-only knob as simple as Postgres's
// default_transaction_read_only RuntimeParam, so that layer isn't
// duplicated here; the grant-level restriction is the actual enforcement.
func NewPool(ctx context.Context, cfg PoolConfig) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open mysql connection: %w", err)
	}

	applyPoolLimits(db)

	if err := verifyConnectivity(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

// applyPoolLimits mirrors the Postgres pool's sizing (see postgres/pool.go)
// — small and isolated, since this pool is shared by the continuous
// session/lock poller and the on-demand insights collector.
func applyPoolLimits(db *sql.DB) {
	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(1)
	db.SetConnMaxIdleTime(30 * time.Second)
	db.SetConnMaxLifetime(30 * time.Minute)
}

func verifyConnectivity(ctx context.Context, db *sql.DB) error {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}
