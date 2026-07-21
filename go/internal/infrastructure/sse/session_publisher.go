package sse

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fayupable/pgscope/internal/domain"
)

// SessionPublisher adapts Broadcaster to output.IEventPublisherPort,
// translating domain sessions and database stats into wire-format SSE
// events.
type SessionPublisher struct {
	broadcaster *Broadcaster
}

func NewSessionPublisher(broadcaster *Broadcaster) *SessionPublisher {
	return &SessionPublisher{broadcaster: broadcaster}
}

func (p *SessionPublisher) PublishSessions(_ context.Context, sessions []domain.Session) error {
	payload, err := json.Marshal(sessions)
	if err != nil {
		return fmt.Errorf("marshal sessions: %w", err)
	}

	p.broadcaster.Broadcast(Event{
		Name: "sessions",
		Data: payload,
	})
	return nil
}

func (p *SessionPublisher) PublishDatabaseStats(_ context.Context, stats domain.DatabaseActivityStats) error {
	payload, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("marshal database stats: %w", err)
	}

	p.broadcaster.Broadcast(Event{
		Name: "db_stats",
		Data: payload,
	})
	return nil
}
