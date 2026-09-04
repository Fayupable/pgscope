package mysql

import (
	"context"
	"database/sql"
	"strings"

	"github.com/fayupable/pgscope/internal/domain"
)

// duplicateIndexQuery reads MySQL's own redundant-index analysis from
// sys.schema_redundant_indexes — unlike the Postgres adapter, no column-
// prefix comparison needs to happen here, MySQL's sys schema has already
// determined which index is redundant against which. Requires SELECT
// granted on the sys schema (see go/README.md's MySQL setup instructions).
const duplicateIndexQuery = `
SELECT
    table_name,
    redundant_index_name,
    redundant_index_columns,
    dominant_index_name,
    dominant_index_columns
FROM sys.schema_redundant_indexes
WHERE table_schema = DATABASE()
`

// DuplicateIndexCollector reads MySQL's precomputed redundant-index
// findings from the sys schema. No further judgment is needed here — see
// domain.NewDuplicateIndex, which only formats the explanation text.
type DuplicateIndexCollector struct {
	db *sql.DB
}

func NewDuplicateIndexCollector(db *sql.DB) *DuplicateIndexCollector {
	return &DuplicateIndexCollector{db: db}
}

func (c *DuplicateIndexCollector) Fetch(ctx context.Context) ([]domain.DuplicateIndex, error) {
	rows, err := c.db.QueryContext(ctx, duplicateIndexQuery)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make([]domain.DuplicateIndex, 0)
	for rows.Next() {
		var table, redundantIndex, redundantColumns, dominantIndex, dominantColumns string
		if err := rows.Scan(&table, &redundantIndex, &redundantColumns, &dominantIndex, &dominantColumns); err != nil {
			return nil, err
		}

		result = append(result, domain.NewDuplicateIndex(
			table,
			redundantIndex,
			dominantIndex,
			splitColumns(redundantColumns),
			splitColumns(dominantColumns),
		))
	}

	return result, rows.Err()
}

// splitColumns parses sys.schema_redundant_indexes' comma-separated column
// list (e.g. "col_a,col_b") into a slice, trimming any incidental spaces.
func splitColumns(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}
