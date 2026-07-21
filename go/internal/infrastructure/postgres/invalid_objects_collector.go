package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

const invalidIndexesQuery = `
SELECT
    t.relname,
    ix.relname
FROM pg_index i
JOIN pg_class t ON t.oid = i.indrelid
JOIN pg_class ix ON ix.oid = i.indexrelid
JOIN pg_namespace n ON n.oid = t.relnamespace
WHERE NOT i.indisvalid
  AND n.nspname = 'public'
`

const unvalidatedConstraintsQuery = `
SELECT
    t.relname,
    c.conname
FROM pg_constraint c
JOIN pg_class t ON t.oid = c.conrelid
JOIN pg_namespace n ON n.oid = t.relnamespace
WHERE NOT c.convalidated
  AND n.nspname = 'public'
`

// InvalidObjectsCollector reads indexes and constraints the catalog
// itself marks as not fully usable — no judgment needed here, Postgres
// has already made the determination (indisvalid/convalidated), this
// just surfaces it.
type InvalidObjectsCollector struct {
	pool *pgxpool.Pool
}

func NewInvalidObjectsCollector(pool *pgxpool.Pool) *InvalidObjectsCollector {
	return &InvalidObjectsCollector{pool: pool}
}

func (c *InvalidObjectsCollector) FetchInvalidIndexes(ctx context.Context) ([]domain.InvalidIndex, error) {
	rows, err := c.pool.Query(ctx, invalidIndexesQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.InvalidIndex, 0)
	for rows.Next() {
		var table, index string
		if err := rows.Scan(&table, &index); err != nil {
			return nil, err
		}
		result = append(result, domain.NewInvalidIndex(table, index))
	}

	return result, rows.Err()
}

func (c *InvalidObjectsCollector) FetchUnvalidatedConstraints(ctx context.Context) ([]domain.UnvalidatedConstraint, error) {
	rows, err := c.pool.Query(ctx, unvalidatedConstraintsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.UnvalidatedConstraint, 0)
	for rows.Next() {
		var table, constraint string
		if err := rows.Scan(&table, &constraint); err != nil {
			return nil, err
		}
		result = append(result, domain.NewUnvalidatedConstraint(table, constraint))
	}

	return result, rows.Err()
}
