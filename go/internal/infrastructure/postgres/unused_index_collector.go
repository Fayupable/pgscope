package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

const unusedIndexLimit = 20

// unusedIndexQuery mirrors pgHero's proven approach: non-unique indexes
// (primary keys and unique constraints are always unique, so they're
// excluded automatically) with very few scans, ordered by wasted disk
// space first since that's the more actionable signal than scan count
// alone.
const unusedIndexQuery = `
SELECT
    relname AS table_name,
    indexrelname AS index_name,
    pg_relation_size(ui.indexrelid) AS size_bytes,
    idx_scan
FROM pg_stat_user_indexes ui
JOIN pg_index i ON ui.indexrelid = i.indexrelid
WHERE NOT i.indisunique
  AND idx_scan <= $1
ORDER BY pg_relation_size(ui.indexrelid) DESC
LIMIT $2
`

// UnusedIndexCollector reads raw index size/scan-count statistics from
// the system catalog. It has no opinion on whether an index is truly
// unused — that judgment (including the observation-window gate)
// belongs to domain.DetectUnusedIndexes.
type UnusedIndexCollector struct {
	pool *pgxpool.Pool
}

func NewUnusedIndexCollector(pool *pgxpool.Pool) *UnusedIndexCollector {
	return &UnusedIndexCollector{pool: pool}
}

func (c *UnusedIndexCollector) FetchCandidates(ctx context.Context) ([]domain.UnusedIndexInfo, error) {
	rows, err := c.pool.Query(ctx, unusedIndexQuery, domain.MaxScansForUnused, unusedIndexLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexes := make([]domain.UnusedIndexInfo, 0)
	for rows.Next() {
		var idx domain.UnusedIndexInfo
		if err := rows.Scan(&idx.Table, &idx.Index, &idx.SizeBytes, &idx.IndexScans); err != nil {
			return nil, err
		}
		indexes = append(indexes, idx)
	}

	return indexes, rows.Err()
}
