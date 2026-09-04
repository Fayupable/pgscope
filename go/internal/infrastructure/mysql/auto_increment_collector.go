package mysql

import (
	"context"
	"database/sql"
	"math"
	"strings"

	"github.com/fayupable/pgscope/internal/domain"
)

// autoIncrementQuery reads every table's current AUTO_INCREMENT value
// alongside its column's declared type — MySQL has no separate sequence
// object outside MariaDB, the equivalent overflow risk lives on the
// auto-incrementing column itself. AUTO_INCREMENT here is the next value
// to be assigned, close enough to "current usage" for this purpose (same
// caveat Postgres's own sequence last_value carries).
//
// Caveat verified empirically: unlike Postgres's pg_sequences.last_value,
// information_schema.TABLES.AUTO_INCREMENT is served from InnoDB's
// persistent statistics cache and is not always live — it can lag behind
// the real value until InnoDB refreshes it (this happens automatically
// once roughly 10% of a table's rows change, via innodb_stats_auto_recalc,
// or immediately after an explicit ANALYZE TABLE). In practice this means
// the reported value may occasionally under-report how close a table is
// to overflow on a table that changes rarely.
const autoIncrementQuery = `
SELECT
    t.TABLE_NAME,
    t.AUTO_INCREMENT,
    c.DATA_TYPE,
    c.COLUMN_TYPE
FROM information_schema.TABLES t
JOIN information_schema.COLUMNS c
    ON c.TABLE_SCHEMA = t.TABLE_SCHEMA AND c.TABLE_NAME = t.TABLE_NAME
WHERE t.TABLE_SCHEMA = DATABASE()
  AND t.AUTO_INCREMENT IS NOT NULL
  AND c.EXTRA LIKE '%auto_increment%'
`

// AutoIncrementCollector reads AUTO_INCREMENT usage directly from the
// system catalog. It has no opinion on what's "too close" to the limit —
// that judgment belongs to domain.DetectSequenceOverflowRisks, unchanged
// from the Postgres adapter (MySQL's AUTO_INCREMENT columns are mapped
// onto the same engine-agnostic domain.SequenceUsage shape).
type AutoIncrementCollector struct {
	db *sql.DB
}

func NewAutoIncrementCollector(db *sql.DB) *AutoIncrementCollector {
	return &AutoIncrementCollector{db: db}
}

func (c *AutoIncrementCollector) Fetch(ctx context.Context) ([]domain.SequenceUsage, error) {
	rows, err := c.db.QueryContext(ctx, autoIncrementQuery)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	usages := make([]domain.SequenceUsage, 0)
	for rows.Next() {
		var table, dataType, columnType string
		var current int64
		if err := rows.Scan(&table, &current, &dataType, &columnType); err != nil {
			return nil, err
		}

		usages = append(usages, domain.SequenceUsage{
			Sequence:     table,
			CurrentValue: current,
			MaxValue:     maxValueFor(dataType, strings.Contains(columnType, "unsigned")),
		})
	}

	return usages, rows.Err()
}

// maxValueFor returns the largest value an AUTO_INCREMENT column of this
// MySQL integer type can hold. BIGINT UNSIGNED's true maximum
// (18446744073709551615) exceeds int64's range — domain.SequenceUsage
// uses int64 to match Postgres's sequence values, so it's capped at
// math.MaxInt64 here, a practical simplification: a table realistically
// reaching even that capped value is already deep in overflow territory
// by any definition.
func maxValueFor(dataType string, unsigned bool) int64 {
	switch strings.ToLower(dataType) {
	case "tinyint":
		if unsigned {
			return 255
		}
		return 127
	case "smallint":
		if unsigned {
			return 65535
		}
		return 32767
	case "mediumint":
		if unsigned {
			return 16777215
		}
		return 8388607
	case "int":
		if unsigned {
			return 4294967295
		}
		return 2147483647
	case "bigint":
		if unsigned {
			return math.MaxInt64
		}
		return math.MaxInt64
	default:
		return math.MaxInt64
	}
}
