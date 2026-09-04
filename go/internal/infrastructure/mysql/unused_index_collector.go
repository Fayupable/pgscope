package mysql

import (
	"context"
	"database/sql"

	"github.com/fayupable/pgscope/internal/domain"
)

// unusedIndexQuery reads MySQL's own unused-index analysis from
// sys.schema_unused_indexes — like the duplicate-index adapter, no
// threshold logic needs to happen here, MySQL's sys schema already
// determined which indexes have had zero reads. Requires SELECT granted
// on the sys schema (see go/README.md's MySQL setup instructions).
const unusedIndexQuery = `
SELECT object_name, index_name
FROM sys.schema_unused_indexes
WHERE object_schema = DATABASE()
`

// UnusedIndexCollector reads MySQL's precomputed unused-index findings
// from the sys schema. No further judgment is needed here — see
// domain.NewUnusedIndex, which only formats the explanation text.
type UnusedIndexCollector struct {
	db *sql.DB
}

func NewUnusedIndexCollector(db *sql.DB) *UnusedIndexCollector {
	return &UnusedIndexCollector{db: db}
}

func (c *UnusedIndexCollector) Fetch(ctx context.Context) ([]domain.UnusedIndex, error) {
	rows, err := c.db.QueryContext(ctx, unusedIndexQuery)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make([]domain.UnusedIndex, 0)
	for rows.Next() {
		var table, index string
		if err := rows.Scan(&table, &index); err != nil {
			return nil, err
		}
		result = append(result, domain.NewUnusedIndex(table, index))
	}

	return result, rows.Err()
}
