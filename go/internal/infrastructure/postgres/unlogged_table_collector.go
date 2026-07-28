package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

// unloggedTablesQuery reads ordinary tables (relkind = 'r', so views,
// sequences, etc. are excluded) marked UNLOGGED (relpersistence = 'u') in
// the catalog — no judgment needed here, Postgres itself already recorded
// this at CREATE TABLE time, this just surfaces it.
const unloggedTablesQuery = `
SELECT t.relname
FROM pg_class t
JOIN pg_namespace n ON n.oid = t.relnamespace
WHERE t.relkind = 'r'
  AND t.relpersistence = 'u'
  AND n.nspname = 'public'
`

// UnloggedTableCollector reads tables the catalog marks as UNLOGGED.
type UnloggedTableCollector struct {
	pool *pgxpool.Pool
}

func NewUnloggedTableCollector(pool *pgxpool.Pool) *UnloggedTableCollector {
	return &UnloggedTableCollector{pool: pool}
}

func (c *UnloggedTableCollector) Fetch(ctx context.Context) ([]domain.UnloggedTable, error) {
	rows, err := c.pool.Query(ctx, unloggedTablesQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.UnloggedTable, 0)
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		result = append(result, domain.NewUnloggedTable(table))
	}

	return result, rows.Err()
}
