package mysql

import (
	"context"
	"database/sql"
)

const queryTextsPerTableLimit = 50

// queryTextsForTableQuery mirrors the Postgres adapter's approach:
// find every distinct normalized query shape mentioning the given
// table, ranked by call count so the most common shapes are seen first,
// but never filtered down to only the top N by cost — a rarely-run but
// relevant match must not be silently dropped.
const queryTextsForTableQuery = `
SELECT DIGEST_TEXT
FROM performance_schema.events_statements_summary_by_digest
WHERE SCHEMA_NAME = DATABASE()
  AND DIGEST_TEXT LIKE CONCAT('%', ?, '%')
ORDER BY COUNT_STAR DESC
LIMIT ?
`

// QueryTextCollector finds every distinct normalized query shape that
// mentions a given table, used to guess which columns a table is most
// often filtered on.
type QueryTextCollector struct {
	db *sql.DB
}

func NewQueryTextCollector(db *sql.DB) *QueryTextCollector {
	return &QueryTextCollector{db: db}
}

func (c *QueryTextCollector) FetchForTable(ctx context.Context, table string) ([]string, error) {
	rows, err := c.db.QueryContext(ctx, queryTextsForTableQuery, table, queryTextsPerTableLimit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	texts := make([]string, 0)
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return nil, err
		}
		texts = append(texts, text)
	}

	return texts, rows.Err()
}
