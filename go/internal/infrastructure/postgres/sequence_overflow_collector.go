package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

// sequenceUsageQuery reads non-cycling sequences only — a cycling
// sequence wraps around by design, so approaching its max value is
// expected behavior, not a risk. last_value can be NULL if the sequence
// has never been used yet, which COALESCE treats as 0 (no risk).
const sequenceUsageQuery = `
SELECT
    sequencename,
    COALESCE(last_value, 0),
    max_value
FROM pg_sequences
WHERE schemaname = 'public'
  AND NOT cycle
`

// SequenceOverflowCollector reads sequence usage directly from the
// system catalog. It has no opinion on what counts as "too close to
// the limit" — that judgment belongs to domain.DetectSequenceOverflowRisks.
type SequenceOverflowCollector struct {
	pool *pgxpool.Pool
}

func NewSequenceOverflowCollector(pool *pgxpool.Pool) *SequenceOverflowCollector {
	return &SequenceOverflowCollector{pool: pool}
}

func (c *SequenceOverflowCollector) Fetch(ctx context.Context) ([]domain.SequenceUsage, error) {
	rows, err := c.pool.Query(ctx, sequenceUsageQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sequences := make([]domain.SequenceUsage, 0)
	for rows.Next() {
		var s domain.SequenceUsage
		if err := rows.Scan(&s.Sequence, &s.CurrentValue, &s.MaxValue); err != nil {
			return nil, err
		}
		sequences = append(sequences, s)
	}

	return sequences, rows.Err()
}
