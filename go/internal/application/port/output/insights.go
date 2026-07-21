package output

import (
	"context"

	"github.com/fayupable/pgscope/internal/domain"
)

// IInsightsPort is implemented by each database engine adapter to surface
// advisory suggestions (query performance, index health, database health,
// and more) derived from system statistics. Purely read-only and purely
// advisory — no method on this interface ever executes DDL/DML.
type IInsightsPort interface {
	FetchInsights(ctx context.Context) (domain.Insights, error)
}
