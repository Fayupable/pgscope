package mysql

import (
	"context"
	"database/sql"

	"github.com/fayupable/pgscope/internal/domain"
)

// InsightsCollector implements output.IInsightsPort for MySQL/MariaDB by
// composing focused, single-purpose collectors, the same orchestration
// role the Postgres adapter's InsightsCollector plays. It holds no SQL
// itself. Every suggestion produced downstream is advisory — never a
// certainty, never an automatic action.
//
// domain.Insights is shared across engines, but MySQL doesn't implement
// every field the Postgres adapter does — some Postgres concepts have no
// MySQL equivalent (VACUUM, WAL checkpoints, replication slots) or no
// good one (pg_stat_kcache-style physical I/O, function cost tracking,
// invalid index/constraint catalog entries). Those fields are left at
// their zero value (empty slice, false, zero struct) rather than
// fabricated — the JSON output simply omits/empties them for this
// engine.
type InsightsCollector struct {
	databaseSize         *DatabaseSizeCollector
	connectionSaturation *ConnectionSaturationCollector
	preparedTransactions *PreparedTransactionCollector
	duplicateIndexes     *DuplicateIndexCollector
	unusedIndexes        *UnusedIndexCollector
	idleInTransaction    *IdleInTransactionCollector
	topQueries           *TopQueryCollector
	longRunningQueries   *LongRunningQueryCollector
	autoIncrement        *AutoIncrementCollector
	unloggedTables       *UnloggedTableCollector
	pagination           *PaginationCollector
	indexCandidates      *IndexCandidateCollector
	queryTexts           *QueryTextCollector
	lockWaits            *LockWaitCollector
}

func NewInsightsCollector(db *sql.DB) *InsightsCollector {
	return &InsightsCollector{
		databaseSize:         NewDatabaseSizeCollector(db),
		connectionSaturation: NewConnectionSaturationCollector(db),
		preparedTransactions: NewPreparedTransactionCollector(db),
		duplicateIndexes:     NewDuplicateIndexCollector(db),
		unusedIndexes:        NewUnusedIndexCollector(db),
		idleInTransaction:    NewIdleInTransactionCollector(db),
		topQueries:           NewTopQueryCollector(db),
		longRunningQueries:   NewLongRunningQueryCollector(db),
		autoIncrement:        NewAutoIncrementCollector(db),
		unloggedTables:       NewUnloggedTableCollector(db),
		pagination:           NewPaginationCollector(db),
		indexCandidates:      NewIndexCandidateCollector(db),
		queryTexts:           NewQueryTextCollector(db),
		lockWaits:            NewLockWaitCollector(db),
	}
}

func (c *InsightsCollector) FetchInsights(ctx context.Context) (domain.Insights, error) {
	databaseSize, err := c.databaseSize.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	connectionSaturation, err := c.connectionSaturation.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	preparedTransactions, err := c.preparedTransactions.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	duplicateIndexes, err := c.duplicateIndexes.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	unusedIndexes, err := c.unusedIndexes.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	idleInTransaction, err := c.idleInTransaction.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	topQueries, err := c.topQueries.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	longRunningQueries, err := c.longRunningQueries.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	autoIncrementUsages, err := c.autoIncrement.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	unloggedTables, err := c.unloggedTables.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	paginationSignals, err := c.pagination.FetchCandidates(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	indexCandidates, err := FetchIndexCandidates(
		ctx, c.indexCandidates, c.queryTexts, domain.MinRowsForSuggestion, domain.MaxIndexUsagePercent,
	)
	if err != nil {
		return domain.Insights{}, err
	}

	lockWaits, err := c.lockWaits.Fetch(ctx)
	if err != nil {
		return domain.Insights{}, err
	}

	return domain.Insights{
		TopQueries:                  topQueries,
		IndexCandidates:             indexCandidates,
		DuplicateIndexes:            duplicateIndexes,
		UnusedIndexes:               unusedIndexes,
		PaginationWarnings:          domain.DetectPaginationWarnings(paginationSignals),
		DatabaseSize:                databaseSize,
		ConnectionSaturation:        connectionSaturation,
		SequenceOverflowRisks:       domain.DetectSequenceOverflowRisks(autoIncrementUsages),
		IdleInTransactionWarnings:   domain.DetectIdleInTransactionWarnings(idleInTransaction),
		PreparedTransactionWarnings: domain.DetectPreparedTransactionWarnings(preparedTransactions),
		LongRunningQueryWarnings:    domain.DetectLongRunningQueryWarnings(longRunningQueries),
		UnloggedTables:              unloggedTables,
		LockWaitWarnings:            domain.DetectLockWaitWarnings(lockWaits),
	}, nil
}
