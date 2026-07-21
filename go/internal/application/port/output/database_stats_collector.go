package output

import (
	"context"

	"github.com/fayupable/pgscope/internal/domain"
)

// IDatabaseStatsCollectorPort is implemented by each database engine adapter
// to expose database-wide activity rates (commits/rollbacks per second).
// Kept separate from ISessionCollectorPort since it is a distinct concern
// (database-level health, not per-session detail) — Interface Segregation.
type IDatabaseStatsCollectorPort interface {
	FetchDatabaseStats(ctx context.Context) (domain.DatabaseActivityStats, error)
}
