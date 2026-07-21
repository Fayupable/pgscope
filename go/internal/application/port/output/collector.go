package output

import (
	"context"

	"github.com/fayupable/pgscope/internal/domain"
)

// ISessionCollectorPort is implemented by each database engine adapter
// (Postgres, MySQL, ...) to expose active session data in a normalized form.
// The application layer depends only on this interface, never on a concrete
// driver or SQL dialect.
//
// Contract: on success, implementations must return a non-nil slice (empty
// if there are no active sessions) — callers distinguish "fetch failed"
// from "fetch succeeded with zero sessions" by checking the error, not by
// checking the slice for nil.
type ISessionCollectorPort interface {
	FetchActiveSessions(ctx context.Context) ([]domain.Session, error)
}
