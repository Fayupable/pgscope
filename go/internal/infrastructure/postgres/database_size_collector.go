package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

const largestTablesLimit = 10

const databaseSizeQuery = `SELECT pg_database_size(current_database())`

// largestTablesQuery reports each table's total on-disk footprint
// (heap + indexes + TOAST combined) via pg_total_relation_size, which
// is the number that actually matches "how much disk space is this
// table responsible for" rather than just the heap alone.
const largestTablesQuery = `
SELECT
    c.relname,
    pg_total_relation_size(c.oid) AS total_bytes
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'r'
  AND n.nspname = 'public'
ORDER BY total_bytes DESC
LIMIT $1
`

// DatabaseSizeCollector reads overall and per-table storage size
// directly from the system catalog — a purely descriptive fact with no
// judgment attached.
type DatabaseSizeCollector struct {
	pool *pgxpool.Pool
}

func NewDatabaseSizeCollector(pool *pgxpool.Pool) *DatabaseSizeCollector {
	return &DatabaseSizeCollector{pool: pool}
}

func (c *DatabaseSizeCollector) Fetch(ctx context.Context) (domain.DatabaseSizeInfo, error) {
	var totalBytes int64
	if err := c.pool.QueryRow(ctx, databaseSizeQuery).Scan(&totalBytes); err != nil {
		return domain.DatabaseSizeInfo{}, err
	}

	tables, err := c.fetchLargestTables(ctx)
	if err != nil {
		return domain.DatabaseSizeInfo{}, err
	}

	return domain.DatabaseSizeInfo{
		TotalBytes:    totalBytes,
		LargestTables: tables,
	}, nil
}

func (c *DatabaseSizeCollector) fetchLargestTables(ctx context.Context) ([]domain.TableSize, error) {
	rows, err := c.pool.Query(ctx, largestTablesQuery, largestTablesLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make([]domain.TableSize, 0)
	for rows.Next() {
		var t domain.TableSize
		if err := rows.Scan(&t.Table, &t.TotalBytes); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}

	return tables, rows.Err()
}
