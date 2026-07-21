package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

// indexCatalogQuery reads simple (non-expression) index definitions
// directly from the system catalog — no text parsing of pg_get_indexdef
// output, so it can't be fooled by formatting differences. Expression
// indexes (indexprs IS NOT NULL) are excluded: comparing them correctly
// would require structural expression matching, which is out of scope
// for an advisory signal.
const indexCatalogQuery = `
SELECT
    t.relname AS table_name,
    ix.relname AS index_name,
    am.amname AS access_method,
    i.indisunique AS is_unique,
    i.indisprimary AS is_primary,
    i.indpred IS NOT NULL AS is_partial,
    COALESCE(i.indpred::text, '') AS predicate,
    array_agg(a.attname ORDER BY k.ord) AS columns
FROM pg_index i
JOIN pg_class t ON t.oid = i.indrelid
JOIN pg_class ix ON ix.oid = i.indexrelid
JOIN pg_am am ON am.oid = ix.relam
JOIN pg_namespace n ON n.oid = t.relnamespace
JOIN LATERAL unnest(i.indkey) WITH ORDINALITY AS k(attnum, ord) ON true
JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = k.attnum
WHERE n.nspname = 'public'
  AND i.indisvalid
  AND i.indexprs IS NULL
GROUP BY t.relname, ix.relname, am.amname, i.indisunique, i.indisprimary, i.indpred
ORDER BY t.relname, ix.relname
`

// DuplicateIndexCollector reads index shape (columns, access method,
// uniqueness, partial predicate) straight from the system catalog. It
// has no opinion on which indexes are redundant — that judgment belongs
// to domain.DetectDuplicateIndexes.
type DuplicateIndexCollector struct {
	pool *pgxpool.Pool
}

func NewDuplicateIndexCollector(pool *pgxpool.Pool) *DuplicateIndexCollector {
	return &DuplicateIndexCollector{pool: pool}
}

func (c *DuplicateIndexCollector) FetchIndexes(ctx context.Context) ([]domain.IndexInfo, error) {
	rows, err := c.pool.Query(ctx, indexCatalogQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexes := make([]domain.IndexInfo, 0)
	for rows.Next() {
		var idx domain.IndexInfo
		if err := rows.Scan(&idx.Table, &idx.Name, &idx.AccessMethod, &idx.Unique, &idx.Primary, &idx.Partial, &idx.Predicate, &idx.Columns); err != nil {
			return nil, err
		}
		indexes = append(indexes, idx)
	}

	return indexes, rows.Err()
}
