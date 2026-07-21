package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

const indexCandidateLimit = 10

const indexCandidateTablesQuery = `
SELECT
    t.relname,
    t.seq_scan,
    t.idx_scan,
    t.n_live_tup,
    COALESCE(t.n_tup_ins, 0) + COALESCE(t.n_tup_upd, 0) AS write_ops,
    COALESCE(idx.index_count, 0) AS index_count
FROM pg_stat_user_tables t
LEFT JOIN (
    SELECT relname, COUNT(*) AS index_count
    FROM pg_stat_user_indexes
    GROUP BY relname
) idx ON idx.relname = t.relname
WHERE t.idx_scan > 0
  AND t.n_live_tup >= $1
  AND (100.0 * t.idx_scan / NULLIF(t.seq_scan + t.idx_scan, 0)) < $2
ORDER BY t.seq_scan DESC
LIMIT $3
`

type rawIndexCandidate struct {
	table         string
	seqScan       int64
	idxScan       int64
	estimatedRows int64
	writeOps      int64
	indexCount    int
}

// IndexCandidateCollector reads raw scan/write statistics per table from
// pg_stat_user_tables — it has no opinion on what the numbers mean,
// that judgment belongs to domain.IndexSignal.
type IndexCandidateCollector struct {
	pool *pgxpool.Pool
}

func NewIndexCandidateCollector(pool *pgxpool.Pool) *IndexCandidateCollector {
	return &IndexCandidateCollector{pool: pool}
}

func (c *IndexCandidateCollector) FetchRaw(ctx context.Context) ([]rawIndexCandidate, error) {
	rows, err := c.pool.Query(
		ctx,
		indexCandidateTablesQuery,
		domain.MinRowsForSuggestion,
		domain.MaxIndexUsagePercent,
		indexCandidateLimit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	raw := make([]rawIndexCandidate, 0)
	for rows.Next() {
		var rc rawIndexCandidate
		if err := rows.Scan(&rc.table, &rc.seqScan, &rc.idxScan, &rc.estimatedRows, &rc.writeOps, &rc.indexCount); err != nil {
			return nil, err
		}
		raw = append(raw, rc)
	}

	return raw, rows.Err()
}
