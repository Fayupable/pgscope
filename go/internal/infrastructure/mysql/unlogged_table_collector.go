package mysql

import (
	"context"
	"database/sql"

	"github.com/fayupable/pgscope/internal/domain"
)

// memoryTablesQuery reads tables using the MEMORY storage engine — MySQL
// has no literal "UNLOGGED" table type, but MEMORY tables share the exact
// same risk Postgres's UNLOGGED tables carry: all data is lost on a
// restart or crash, since nothing is ever written to disk.
const memoryTablesQuery = `
SELECT TABLE_NAME
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = DATABASE()
  AND ENGINE = 'MEMORY'
`

// UnloggedTableCollector reads tables the catalog marks as using the
// MEMORY engine. No further judgment is needed here — see
// domain.NewMemoryEngineTable, which carries the same underlying finding
// as the Postgres adapter's domain.NewUnloggedTable but with wording
// specific to MySQL's MEMORY engine.
type UnloggedTableCollector struct {
	db *sql.DB
}

func NewUnloggedTableCollector(db *sql.DB) *UnloggedTableCollector {
	return &UnloggedTableCollector{db: db}
}

func (c *UnloggedTableCollector) Fetch(ctx context.Context) ([]domain.UnloggedTable, error) {
	rows, err := c.db.QueryContext(ctx, memoryTablesQuery)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make([]domain.UnloggedTable, 0)
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		result = append(result, domain.NewMemoryEngineTable(table))
	}

	return result, rows.Err()
}
