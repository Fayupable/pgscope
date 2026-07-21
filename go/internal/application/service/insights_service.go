package service

import (
	"context"

	"github.com/fayupable/pgscope/internal/application/port/output"
	"github.com/fayupable/pgscope/internal/domain"
)

// InsightsService is the use case orchestrating advisory insight
// generation (query performance, index health, database health, and
// more). It depends only on the output port interface, never on a
// concrete database adapter.
type InsightsService struct {
	insights output.IInsightsPort
}

func NewInsightsService(insights output.IInsightsPort) *InsightsService {
	return &InsightsService{insights: insights}
}

func (s *InsightsService) GetInsights(ctx context.Context) (domain.Insights, error) {
	return s.insights.FetchInsights(ctx)
}
