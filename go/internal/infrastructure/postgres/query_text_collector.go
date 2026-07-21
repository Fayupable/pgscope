package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const queryTextsPerTableLimit = 50

const queryTextsForTableQuery = `
SELECT query
FROM pg_stat_statements
WHERE ` + currentDbFilter + `
  AND query ILIKE '%' || $1 || '%'
ORDER BY calls DESC
LIMIT $2
`

// QueryTextCollector finds every distinct query shape that mentions a
// given table, regardless of how often it has been called — filtering
// by call rank would silently drop relevant but rarely-run matches.
type QueryTextCollector struct {
	pool *pgxpool.Pool
}

func NewQueryTextCollector(pool *pgxpool.Pool) *QueryTextCollector {
	return &QueryTextCollector{pool: pool}
}

func (c *QueryTextCollector) FetchForTable(ctx context.Context, table string) ([]string, error) {
	rows, err := c.pool.Query(ctx, queryTextsForTableQuery, table, queryTextsPerTableLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
