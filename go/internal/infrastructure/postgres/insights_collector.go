package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

// InsightsCollector implements output.IInsightsPort by composing
// focused, single-purpose collectors. It holds no SQL itself — its only
// job is orchestration: fetch each piece, hand raw numbers to the
// domain layer for judgment, and assemble the result. Every suggestion
// produced downstream is advisory — never a certainty, never an
// automatic action.
type InsightsCollector struct {
	topQueries           *TopQueryCollector
	candidates           *IndexCandidateCollector
	queryTexts           *QueryTextCollector
	selectivity          *ColumnSelectivityCollector
	statsReset           *StatsResetReader
	duplicateIndexes     *DuplicateIndexCollector
	unusedIndexes        *UnusedIndexCollector
	functionCosts        *FunctionCostCollector
	pagination           *PaginationCollector
	databaseSize         *DatabaseSizeCollector
	connectionSaturation *ConnectionSaturationCollector
	sequenceOverflow     *SequenceOverflowCollector
	invalidObjects       *InvalidObjectsCollector
	vacuumHealth         *VacuumHealthCollector
	idleInTransaction    *IdleInTransactionCollector
	checkpointHealth     *CheckpointHealthCollector
	replicationLag       *ReplicationLagCollector
	kcache               *KcacheCollector
}

func NewInsightsCollector(pool *pgxpool.Pool) *InsightsCollector {
	return &InsightsCollector{
		topQueries:           NewTopQueryCollector(pool),
		candidates:           NewIndexCandidateCollector(pool),
		queryTexts:           NewQueryTextCollector(pool),
		selectivity:          NewColumnSelectivityCollector(pool),
		statsReset:           NewStatsResetReader(pool),
		duplicateIndexes:     NewDuplicateIndexCollector(pool),
		unusedIndexes:        NewUnusedIndexCollector(pool),
		functionCosts:        NewFunctionCostCollector(pool),
		pagination:           NewPaginationCollector(pool),
		databaseSize:         NewDatabaseSizeCollector(pool),
		connectionSaturation: NewConnectionSaturationCollector(pool),
		sequenceOverflow:     NewSequenceOverflowCollector(pool),
		invalidObjects:       NewInvalidObjectsCollector(pool),
		vacuumHealth:         NewVacuumHealthCollector(pool),
		idleInTransaction:    NewIdleInTransactionCollector(pool),
		checkpointHealth:     NewCheckpointHealthCollector(pool),
		replicationLag:       NewReplicationLagCollector(pool),
		kcache:               NewKcacheCollector(pool),
	}
}

func (c *InsightsCollector) FetchInsights(ctx context.Context) (domain.Insights, error) {
	elapsedSeconds, err := c.statsReset.SecondsSinceReset(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	topQueries, err := c.topQueries.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	indexCandidates, err := c.fetchIndexCandidates(ctx, elapsedSeconds)
	if err != nil {
		return domain.Insights{}, err
	}

	duplicateIndexes, err := c.fetchDuplicateIndexes(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	unusedIndexes, err := c.fetchUnusedIndexes(ctx, elapsedSeconds)
	if err != nil {
		return domain.Insights{}, err
	}

	functionCosts, trackFunctionsSetting, err := c.fetchFunctionCosts(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	paginationWarnings, statementsTrackSetting, err := c.fetchPaginationWarnings(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	databaseSize, err := c.databaseSize.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	connectionSaturation, err := c.connectionSaturation.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	sequenceUsages, err := c.sequenceOverflow.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	invalidIndexes, err := c.invalidObjects.FetchInvalidIndexes(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	unvalidatedConstraints, err := c.invalidObjects.FetchUnvalidatedConstraints(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	vacuumStats, err := c.vacuumHealth.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	idleSessions, err := c.idleInTransaction.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}
	checkpointStats, err := c.checkpointHealth.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	replicaLags, err := c.replicationLag.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	physicalIOEnabled, err := c.kcache.Enabled(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	var physicalIOStats []domain.QueryPhysicalIO
	if physicalIOEnabled {
		physicalIOStats, err = c.kcache.Fetch(ctx)
		if err != nil {
			return domain.Insights{}, err
		}
	}
	return domain.Insights{
		TopQueries:                topQueries,
		IndexCandidates:           indexCandidates,
		DuplicateIndexes:          duplicateIndexes,
		UnusedIndexes:             unusedIndexes,
		FunctionCosts:             functionCosts,
		TrackFunctionsEnabled:     trackFunctionsSetting != "none",
		TrackFunctionsSetting:     trackFunctionsSetting,
		PaginationWarnings:        paginationWarnings,
		NestedStatementsTracked:   statementsTrackSetting == "all",
		StatementsTrackSetting:    statementsTrackSetting,
		DatabaseSize:              databaseSize,
		ConnectionSaturation:      connectionSaturation,
		SequenceOverflowRisks:     domain.DetectSequenceOverflowRisks(sequenceUsages),
		InvalidIndexes:            invalidIndexes,
		UnvalidatedConstraints:    unvalidatedConstraints,
		VacuumHealthWarnings:      domain.DetectVacuumHealthWarnings(vacuumStats),
		IdleInTransactionWarnings: domain.DetectIdleInTransactionWarnings(idleSessions),
		CheckpointHealth:          domain.NewCheckpointHealth(checkpointStats),
		ReplicationLagWarnings:    domain.DetectReplicationLagWarnings(replicaLags),
		PhysicalIOEnabled:         physicalIOEnabled,
		PhysicalIOHotspots:        domain.DetectPhysicalIOHotspots(physicalIOStats),
	}, nil
}

func (c *InsightsCollector) fetchPaginationWarnings(ctx context.Context) ([]domain.PaginationWarning, string, error) {
	setting, err := c.pagination.StatementsTrackSetting(ctx)
	if err != nil {
		return nil, "", err
	}

	signals, err := c.pagination.FetchCandidates(ctx)
	if err != nil {
		return nil, "", err
	}

	return domain.DetectPaginationWarnings(signals), setting, nil
}

func (c *InsightsCollector) fetchFunctionCosts(ctx context.Context) ([]domain.FunctionCost, string, error) {
	setting, err := c.functionCosts.TrackFunctionsSetting(ctx)
	if err != nil {
		return nil, "", err
	}

	if setting == "none" {
		return []domain.FunctionCost{}, setting, nil
	}

	stats, err := c.functionCosts.FetchFunctionStats(ctx)
	if err != nil {
		return nil, "", err
	}

	return domain.DetectExpensiveFunctions(stats), setting, nil
}

func (c *InsightsCollector) fetchDuplicateIndexes(ctx context.Context) ([]domain.DuplicateIndex, error) {
	indexes, err := c.duplicateIndexes.FetchIndexes(ctx)
	if err != nil {
		return nil, err
	}

	return domain.DetectDuplicateIndexes(indexes), nil
}

func (c *InsightsCollector) fetchUnusedIndexes(ctx context.Context, elapsedSeconds float64) ([]domain.UnusedIndex, error) {
	candidates, err := c.unusedIndexes.FetchCandidates(ctx)
	if err != nil {
		return nil, err
	}
	return domain.DetectUnusedIndexes(candidates, elapsedSeconds), nil
}

func (c *InsightsCollector) fetchIndexCandidates(ctx context.Context, elapsedSeconds float64) ([]domain.IndexCandidate, error) {
	raw, err := c.candidates.FetchRaw(ctx)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return []domain.IndexCandidate{}, nil
	}

	result := make([]domain.IndexCandidate, 0, len(raw))
	for _, rc := range raw {
		candidate, err := c.buildCandidate(ctx, rc, elapsedSeconds)
		if err != nil {
			return nil, err
		}
		result = append(result, candidate)
	}

	return result, nil
}

func (c *InsightsCollector) buildCandidate(ctx context.Context, rc rawIndexCandidate, elapsedSeconds float64) (domain.IndexCandidate, error) {
	queryTexts, err := c.queryTexts.FetchForTable(ctx, rc.table)
	if err != nil {
		return domain.IndexCandidate{}, err
	}

	suspectedColumns := extractSuspectedColumns(queryTexts)

	signal := domain.IndexSignal{
		EstimatedRows: rc.estimatedRows,
		SeqScan:       rc.seqScan,
		IdxScan:       rc.idxScan,
	}

	if len(suspectedColumns) > 0 {
		nDistinct, found, err := c.selectivity.Fetch(ctx, rc.table, suspectedColumns[0])
		if err != nil {
			return domain.IndexCandidate{}, err
		}
		if found {
			signal.NDistinct = &nDistinct
		}
	}

	return domain.NewIndexCandidate(
		rc.table,
		signal,
		suspectedColumns,
		rc.indexCount,
		writesPerSecond(rc.writeOps, elapsedSeconds),
	), nil
}
