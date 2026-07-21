package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const columnNDistinctQuery = `
SELECT n_distinct
FROM pg_stats
WHERE schemaname = 'public'
  AND tablename = $1
  AND attname = $2
`

// ColumnSelectivityCollector reads pg_stats.n_distinct for a single
// column — the raw input the domain layer needs to judge whether a
// suspected column is selective enough for an index to actually help.
type ColumnSelectivityCollector struct {
	pool *pgxpool.Pool
}

func NewColumnSelectivityCollector(pool *pgxpool.Pool) *ColumnSelectivityCollector {
	return &ColumnSelectivityCollector{pool: pool}
}

// Fetch returns pg_stats.n_distinct for the given column. found is false
// if the column has never been analyzed (no row in pg_stats yet).
func (c *ColumnSelectivityCollector) Fetch(ctx context.Context, table, column string) (nDistinct float64, found bool, err error) {
	err = c.pool.QueryRow(ctx, columnNDistinctQuery, table, column).Scan(&nDistinct)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, err
	}
	return nDistinct, true, nil
}
