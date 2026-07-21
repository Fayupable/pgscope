package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/fayupable/pgscope/internal/application/port/output"
	"github.com/fayupable/pgscope/internal/domain"
)

var allowedRecordMinutes = map[int]bool{5: true, 10: true, 15: true, 30: true}

// Poller periodically pulls active sessions and database-wide activity
// stats, pushes them out through the publisher port, and optionally records
// history snapshots. Two independent, opt-in controls gate its behavior:
//   - Monitor: whether the poller queries the database and publishes live
//     updates at all. Off by default.
//   - Record: whether, while monitoring is active, ticks are also written
//     to history. Off by default, always bounded to a fixed set of
//     durations (never indefinite).
type Poller struct {
	monitoringService *MonitoringService
	dbStatsCollector  output.IDatabaseStatsCollectorPort
	publisher         output.IEventPublisherPort
	historyStore      output.IHistoryStorePort
	interval          time.Duration

	monitorControl    *Recorder
	recordControl     *Recorder
	previouslyBlocked map[string]bool
}

func NewPoller(
	monitoringService *MonitoringService,
	dbStatsCollector output.IDatabaseStatsCollectorPort,
	publisher output.IEventPublisherPort,
	historyStore output.IHistoryStorePort,
	interval time.Duration,
) *Poller {
	return &Poller{
		monitoringService: monitoringService,
		dbStatsCollector:  dbStatsCollector,
		publisher:         publisher,
		historyStore:      historyStore,
		interval:          interval,
		monitorControl:    NewRecorder(),
		recordControl:     NewRecorder(),
		previouslyBlocked: make(map[string]bool),
	}
}

// StartMonitoring begins live polling. minutes=0 means run until StopMonitoring
// is called explicitly ("Full"); a positive value auto-stops after that many
// minutes.
func (p *Poller) StartMonitoring(minutes int) error {
	if minutes < 0 {
		return fmt.Errorf("minutes cannot be negative")
	}
	p.monitorControl.Start(time.Duration(minutes) * time.Minute)
	return nil
}

func (p *Poller) StopMonitoring() {
	p.monitorControl.Stop()
	p.recordControl.Stop()
}

func (p *Poller) IsMonitoring() bool {
	return p.monitorControl.IsActive()
}

// StartRecording begins writing ticks to history. Recording is always
// bounded to one of a fixed set of presets — never indefinite — since
// history exists for short-window export/replay, not open-ended growth.
func (p *Poller) StartRecording(minutes int) error {
	if !allowedRecordMinutes[minutes] {
		return fmt.Errorf("minutes must be one of: 5, 10, 15, 30")
	}
	p.recordControl.Start(time.Duration(minutes) * time.Minute)
	return nil
}

func (p *Poller) StopRecording() {
	p.recordControl.Stop()
}

func (p *Poller) IsRecording() bool {
	return p.recordControl.IsActive()
}

func (p *Poller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.tick(ctx)
		}
	}
}

func (p *Poller) tick(ctx context.Context) {
	if !p.monitorControl.IsActive() {
		return
	}

	sessions, ok := p.fetchSessions(ctx)
	if ok {
		p.publishSessions(ctx, sessions)
		if p.recordControl.IsActive() {
			p.recordSnapshot(ctx, sessions)
		}
	}

	p.publishDatabaseStats(ctx)
}

func (p *Poller) fetchSessions(ctx context.Context) ([]domain.Session, bool) {
	sessions, err := p.monitoringService.GetActiveSessions(ctx)
	if err != nil {
		slog.Error("failed to fetch active sessions", "error", err)
		return nil, false
	}
	slog.Info("fetched active sessions", "count", len(sessions))
	return sessions, true
}

func (p *Poller) publishSessions(ctx context.Context, sessions []domain.Session) {
	if err := p.publisher.PublishSessions(ctx, sessions); err != nil {
		slog.Error("failed to publish sessions", "error", err)
	}
}

func (p *Poller) publishDatabaseStats(ctx context.Context) {
	stats, err := p.dbStatsCollector.FetchDatabaseStats(ctx)
	if err != nil {
		slog.Error("failed to fetch database stats", "error", err)
		return
	}

	if err := p.publisher.PublishDatabaseStats(ctx, stats); err != nil {
		slog.Error("failed to publish database stats", "error", err)
	}
}

func (p *Poller) recordSnapshot(ctx context.Context, sessions []domain.Session) {
	snapshot := domain.Snapshot{
		Sessions:   sessions,
		CapturedAt: time.Now(),
		Trigger:    p.classifyTrigger(sessions),
	}

	if err := p.historyStore.Append(ctx, snapshot); err != nil {
		slog.Error("failed to record snapshot", "error", err)
	}
}

// classifyTrigger marks this tick as an incident if any session became
// blocked that was not blocked (or did not exist) in the previous tick —
// i.e. a new blocking relationship just appeared. It also updates the
// remembered "was blocked" state for the next tick.
func (p *Poller) classifyTrigger(sessions []domain.Session) domain.SnapshotTrigger {
	currentlyBlocked := make(map[string]bool, len(sessions))
	trigger := domain.SnapshotTriggerPeriodic

	for _, session := range sessions {
		isBlocked := session.IsBlocked()
		currentlyBlocked[session.ID] = isBlocked

		if isBlocked && !p.previouslyBlocked[session.ID] {
			trigger = domain.SnapshotTriggerIncident
		}
	}

	p.previouslyBlocked = currentlyBlocked
	return trigger
}

func (p *Poller) RecentHistory(ctx context.Context) ([]domain.Snapshot, error) {
	return p.historyStore.Recent(ctx)
}

func (p *Poller) Incidents(ctx context.Context) ([]domain.Snapshot, error) {
	return p.historyStore.Incidents(ctx)
}
