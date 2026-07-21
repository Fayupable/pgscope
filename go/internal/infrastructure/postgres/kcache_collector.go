package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

const kcacheExtensionCheckQuery = `SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_kcache')`

const physicalIOLimit = 20

// physicalIOQuery joins pg_stat_kcache (physical I/O and CPU time per
// query) with pg_stat_statements (query text). Written against
// pg_stat_kcache 2.2+, which split reads/writes into plan_*/exec_*
// variants — verify column names if your installed version differs.
const physicalIOQuery = `
SELECT
    s.query,
    k.calls,
    k.exec_reads,
    k.exec_writes,
    k.exec_user_time,
    k.exec_system_time
FROM pg_stat_kcache() k
JOIN pg_stat_statements s ON s.queryid = k.queryid AND s.dbid = k.dbid AND s.userid = k.userid
WHERE k.dbid = (SELECT oid FROM pg_database WHERE datname = current_database())
ORDER BY k.exec_reads DESC
LIMIT $1
`

// KcacheCollector reads physical I/O and CPU statistics from the
// optional pg_stat_kcache extension. pgscope never installs this
// extension itself — it only checks whether it's already present and
// reads from it if so.
type KcacheCollector struct {
	pool *pgxpool.Pool
}

func NewKcacheCollector(pool *pgxpool.Pool) *KcacheCollector {
	return &KcacheCollector{pool: pool}
}

func (c *KcacheCollector) Enabled(ctx context.Context) (bool, error) {
	var enabled bool
	err := c.pool.QueryRow(ctx, kcacheExtensionCheckQuery).Scan(&enabled)
	return enabled, err
}

func (c *KcacheCollector) Fetch(ctx context.Context) ([]domain.QueryPhysicalIO, error) {
	rows, err := c.pool.Query(ctx, physicalIOQuery, physicalIOLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]domain.QueryPhysicalIO, 0)
	for rows.Next() {
		var s domain.QueryPhysicalIO
		if err := rows.Scan(&s.Query, &s.Calls, &s.ExecReads, &s.ExecWrites, &s.UserTimeMs, &s.SystemTimeMs); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, rows.Err()
}
