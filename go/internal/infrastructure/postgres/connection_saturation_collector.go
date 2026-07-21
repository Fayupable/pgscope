package postgres

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

const activeConnectionsQuery = `
SELECT COUNT(*)
FROM pg_stat_activity
WHERE datname = current_database()
`

const maxConnectionsSettingQuery = `SELECT setting FROM pg_settings WHERE name = 'max_connections'`

// ConnectionSaturationCollector reads the current connection count and
// the configured limit from the system catalog. It has no opinion on
// whether this is "too many" — that judgment belongs to
// domain.NewConnectionSaturation.
type ConnectionSaturationCollector struct {
	pool *pgxpool.Pool
}

func NewConnectionSaturationCollector(pool *pgxpool.Pool) *ConnectionSaturationCollector {
	return &ConnectionSaturationCollector{pool: pool}
}

func (c *ConnectionSaturationCollector) Fetch(ctx context.Context) (domain.ConnectionSaturation, error) {
	var active int
	if err := c.pool.QueryRow(ctx, activeConnectionsQuery).Scan(&active); err != nil {
		return domain.ConnectionSaturation{}, err
	}

	var maxConnSetting string
	if err := c.pool.QueryRow(ctx, maxConnectionsSettingQuery).Scan(&maxConnSetting); err != nil {
		return domain.ConnectionSaturation{}, err
	}

	max, err := strconv.Atoi(maxConnSetting)
	if err != nil {
		return domain.ConnectionSaturation{}, err
	}

	return domain.NewConnectionSaturation(active, max), nil
}
