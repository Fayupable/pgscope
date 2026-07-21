package output

import (
	"context"

	"github.com/fayupable/pgscope/internal/domain"
)

// IEventPublisherPort is implemented by the transport layer (SSE, WebSocket,
// ...) to deliver session snapshots and database-wide stats to connected
// clients. The application layer depends only on this interface, never on
// the transport mechanism.
type IEventPublisherPort interface {
	PublishSessions(ctx context.Context, sessions []domain.Session) error
	PublishDatabaseStats(ctx context.Context, stats domain.DatabaseActivityStats) error
}
