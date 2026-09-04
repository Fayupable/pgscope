package mysql

import (
	"context"
	"database/sql"

	"github.com/fayupable/pgscope/internal/domain"
)

const largestTablesLimit = 10

// databaseSizeQuery sums each table's on-disk footprint (data + indexes)
// for the current schema — MySQL has no single built-in function like
// Postgres's pg_database_size, so this is computed the same way
// largestTablesQuery derives per-table size, just summed.
const databaseSizeQuery = `
SELECT COALESCE(SUM(DATA_LENGTH + INDEX_LENGTH), 0)
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = DATABASE()
`

const largestTablesQuery = `
SELECT TABLE_NAME, (DATA_LENGTH + INDEX_LENGTH) AS total_bytes
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = DATABASE()
ORDER BY total_bytes DESC
LIMIT ?
`

// DatabaseSizeCollector reads overall and per-table storage size directly
// from information_schema — a purely descriptive fact with no judgment
// attached, same role as its Postgres counterpart.
type DatabaseSizeCollector struct {
	db *sql.DB
}

func NewDatabaseSizeCollector(db *sql.DB) *DatabaseSizeCollector {
	return &DatabaseSizeCollector{db: db}
}

func (c *DatabaseSizeCollector) Fetch(ctx context.Context) (domain.DatabaseSizeInfo, error) {
	var totalBytes int64
	if err := c.db.QueryRowContext(ctx, databaseSizeQuery).Scan(&totalBytes); err != nil {
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
	rows, err := c.db.QueryContext(ctx, largestTablesQuery, largestTablesLimit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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
