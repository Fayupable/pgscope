package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

const paginationCandidateLimit = 50

// paginationCandidatesQuery deliberately does NOT rank by cost or limit
// itself to a "top N" list — it filters directly on the presence of
// OFFSET in the query text, so a cheap or rarely-ranked query shape is
// just as visible as an expensive one. Detection must never be a subset
// of a display-only list.
const paginationCandidatesQuery = `
SELECT query, calls, mean_exec_time, stddev_exec_time, rows
FROM pg_stat_statements
WHERE ` + currentDbFilter + `
  AND query ILIKE '%OFFSET%'
ORDER BY calls DESC
LIMIT $1
`

const statementsTrackSettingQuery = `SELECT setting FROM pg_settings WHERE name = 'pg_stat_statements.track'`

// PaginationCollector reads OFFSET-containing query statistics from
// pg_stat_statements. It has no opinion on whether a pattern is worth
// warning about — that judgment belongs to domain.DetectPaginationWarnings.
type PaginationCollector struct {
	pool *pgxpool.Pool
}

func NewPaginationCollector(pool *pgxpool.Pool) *PaginationCollector {
	return &PaginationCollector{pool: pool}
}

func (c *PaginationCollector) FetchCandidates(ctx context.Context) ([]domain.PaginationSignal, error) {
	rows, err := c.pool.Query(ctx, paginationCandidatesQuery, paginationCandidateLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	signals := make([]domain.PaginationSignal, 0)
	for rows.Next() {
		var s domain.PaginationSignal
		if err := rows.Scan(&s.Query, &s.Calls, &s.MeanExecMs, &s.StddevExecMs, &s.Rows); err != nil {
			return nil, err
		}
		s.ContainsOffset = true // guaranteed by the SQL filter above
		signals = append(signals, s)
	}

	return signals, rows.Err()
}

// StatementsTrackSetting reports pg_stat_statements.track — 'top' (the
// default) means statements run inside functions, procedures, or DO
// blocks are invisible to every text-based advisory here, since only
// the outermost statement gets a row. pgscope never changes this
// setting; it only surfaces it so the user can decide.
func (c *PaginationCollector) StatementsTrackSetting(ctx context.Context) (string, error) {
	var setting string
	err := c.pool.QueryRow(ctx, statementsTrackSettingQuery).Scan(&setting)
	return setting, err
}
