package output

import (
	"context"

	"github.com/fayupable/pgscope/internal/domain"
)

// IHistoryStorePort is implemented by the infrastructure layer to retain a
// rolling window of periodic snapshots plus a longer-lived set of incident
// snapshots (moments a new blocking relationship appeared). The application
// layer depends only on this interface, never on how history is actually
// stored (in-memory ring buffer, database, ...).
type IHistoryStorePort interface {
	Append(ctx context.Context, snapshot domain.Snapshot) error
	Recent(ctx context.Context) ([]domain.Snapshot, error)
	Incidents(ctx context.Context) ([]domain.Snapshot, error)
}
