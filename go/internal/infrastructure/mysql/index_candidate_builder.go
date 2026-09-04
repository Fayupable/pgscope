package mysql

import (
	"context"

	"github.com/fayupable/pgscope/internal/domain"
)

// FetchIndexCandidates ties together IndexCandidateCollector's raw scan
// stats with QueryTextCollector's per-table query shapes to build fully
// judged domain.IndexCandidate suggestions — the same three-step flow
// the Postgres adapter's InsightsCollector.fetchIndexCandidates performs.
//
// Unlike the Postgres adapter, no selectivity (domain.IndexSignal.NDistinct)
// is set here: MySQL keeps no free, precomputed cardinality statistic for
// an unindexed column the way Postgres's pg_stats.n_distinct does. The
// only way to get one would be a live SELECT COUNT(DISTINCT column)
// against the actual table, which could be expensive on a large table
// this tool has no business slowing down. Selectivity is left unknown
// (NDistinct stays nil) — domain.IndexSignal already treats that as a
// valid, handled case, so candidates are still built, just without the
// extra "how selective is this column" refinement Postgres provides.
func FetchIndexCandidates(
	ctx context.Context,
	candidates *IndexCandidateCollector,
	queryTexts *QueryTextCollector,
	minRows int64,
	maxIndexUsagePercent float64,
) ([]domain.IndexCandidate, error) {
	raw, err := candidates.FetchRaw(ctx, minRows, maxIndexUsagePercent)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return []domain.IndexCandidate{}, nil
	}

	elapsedSeconds := candidates.BeginCycle()

	result := make([]domain.IndexCandidate, 0, len(raw))
	for _, rc := range raw {
		texts, err := queryTexts.FetchForTable(ctx, rc.Table)
		if err != nil {
			return nil, err
		}

		suspectedColumns := extractSuspectedColumns(texts)

		signal := domain.IndexSignal{
			EstimatedRows: rc.EstimatedRows,
			SeqScan:       rc.SeqScan,
			IdxScan:       rc.IdxScan,
		}

		result = append(result, domain.NewIndexCandidate(
			rc.Table,
			signal,
			suspectedColumns,
			rc.IndexCount,
			candidates.WritesPerSecond(rc.Table, rc.WriteOps, elapsedSeconds),
		))
	}

	return result, nil
}
