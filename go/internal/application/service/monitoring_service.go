package service

import (
	"context"

	"github.com/fayupable/pgscope/internal/application/port/output"
	"github.com/fayupable/pgscope/internal/domain"
)

// MonitoringService is the use case orchestrating session collection. It
// depends only on the output port interface, never on a concrete database
// adapter — Postgres, MySQL, or anything else is interchangeable here.
type MonitoringService struct {
	sessionCollector output.ISessionCollectorPort
}

func NewMonitoringService(sessionCollector output.ISessionCollectorPort) *MonitoringService {
	return &MonitoringService{sessionCollector: sessionCollector}
}

func (s *MonitoringService) GetActiveSessions(ctx context.Context) ([]domain.Session, error) {
	return s.sessionCollector.FetchActiveSessions(ctx)
}
