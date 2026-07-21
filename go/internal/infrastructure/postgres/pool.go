package postgres

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PoolConfig struct {
	ConnString string
}

func NewPool(ctx context.Context, cfg PoolConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.ConnString)
	if err != nil {
		return nil, fmt.Errorf("parse connection string: %w", err)
	}

	applyReadOnlySessionParams(poolConfig)
	applyDialTimeout(poolConfig)
	applyPoolLimits(poolConfig)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := verifyConnectivity(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

// applyReadOnlySessionParams enforces read-only transactions and aggressive
// statement/lock timeouts at the session level, independent of any role-level
// enforcement already configured in Postgres itself.
func applyReadOnlySessionParams(poolConfig *pgxpool.Config) {
	if poolConfig.ConnConfig.RuntimeParams == nil {
		poolConfig.ConnConfig.RuntimeParams = make(map[string]string)
	}
	poolConfig.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
	poolConfig.ConnConfig.RuntimeParams["statement_timeout"] = "250"
	poolConfig.ConnConfig.RuntimeParams["lock_timeout"] = "100"
}

// applyDialTimeout ensures a network partition or unresponsive server never
// leaves the collector hanging on connection establishment.
func applyDialTimeout(poolConfig *pgxpool.Config) {
	poolConfig.ConnConfig.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := &net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 15 * time.Second,
		}
		return dialer.DialContext(ctx, network, addr)
	}
}

// applyPoolLimits keeps this pool small and isolated from any application
// connection pool. This pool is shared by the continuous session/lock
// Poller and the on-demand InsightsCollector — Insights now runs 15+
// sequential queries per fetch, so a single concurrent Insights request
// plus the Poller's own tick can use up to 2 of these 3 connections at
// once. Fine for a single local user today; if pgscope is ever used by
// multiple concurrent viewers, this limit (and whether Insights should
// get its own pool) is worth revisiting — see README for the current
// single-user assumption.
func applyPoolLimits(poolConfig *pgxpool.Config) {
	poolConfig.MaxConns = 3
	poolConfig.MinConns = 1
	poolConfig.MaxConnIdleTime = 30 * time.Second
	poolConfig.MaxConnLifetime = 30 * time.Minute
}

func verifyConnectivity(ctx context.Context, pool *pgxpool.Pool) error {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}
