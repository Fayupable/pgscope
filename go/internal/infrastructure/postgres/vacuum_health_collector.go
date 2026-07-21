package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

const vacuumStatsQuery = `
SELECT
    relname,
    n_live_tup,
    n_dead_tup,
    last_autovacuum
FROM pg_stat_user_tables
`

// VacuumHealthCollector reads dead-tuple and autovacuum statistics
// directly from pg_stat_user_tables. It has no opinion on what counts
// as "too many dead tuples" — that judgment belongs to
// domain.DetectVacuumHealthWarnings.
type VacuumHealthCollector struct {
	pool *pgxpool.Pool
}

func NewVacuumHealthCollector(pool *pgxpool.Pool) *VacuumHealthCollector {
	return &VacuumHealthCollector{pool: pool}
}

func (c *VacuumHealthCollector) Fetch(ctx context.Context) ([]domain.TableVacuumStats, error) {
	rows, err := c.pool.Query(ctx, vacuumStatsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]domain.TableVacuumStats, 0)
	for rows.Next() {
		var s domain.TableVacuumStats
		if err := rows.Scan(&s.Table, &s.LiveTuples, &s.DeadTuples, &s.LastAutovacuum); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, rows.Err()
}
