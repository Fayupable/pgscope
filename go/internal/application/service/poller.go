package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/fayupable/pgscope/internal/application/port/output"
	"github.com/fayupable/pgscope/internal/domain"
)

// Poller periodically pulls active sessions and database-wide activity
// stats, pushes them out through the publisher port, and — whenever
// monitoring is active — records history snapshots. There is no separate
// "start recording" concept: recording follows monitoring automatically,
// at its own slower cadence (recordInterval), so a long-running monitoring
// session builds up history in the background without the caller having to
// manage it. Incident snapshots (a new blocking relationship appearing)
// are always recorded immediately, regardless of that cadence.
type Poller struct {
	monitoringService      *MonitoringService
	dbStatsCollector       output.IDatabaseStatsCollectorPort
	publisher              output.IEventPublisherPort
	historyStore           output.IHistoryStorePort
	interval               time.Duration
	recordInterval         time.Duration
	maxSessionsPerSnapshot int

	monitorControl    *Recorder
	previouslyBlocked map[string]bool
	lastRecordedAt    time.Time
}

func NewPoller(
	monitoringService *MonitoringService,
	dbStatsCollector output.IDatabaseStatsCollectorPort,
	publisher output.IEventPublisherPort,
	historyStore output.IHistoryStorePort,
	interval time.Duration,
	recordInterval time.Duration,
	maxSessionsPerSnapshot int,
) *Poller {
	return &Poller{
		monitoringService:      monitoringService,
		dbStatsCollector:       dbStatsCollector,
		publisher:              publisher,
		historyStore:           historyStore,
		interval:               interval,
		recordInterval:         recordInterval,
		maxSessionsPerSnapshot: maxSessionsPerSnapshot,
		monitorControl:         NewRecorder(),
		previouslyBlocked:      make(map[string]bool),
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
}

func (p *Poller) IsMonitoring() bool {
	return p.monitorControl.IsActive()
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
		p.maybeRecordSnapshot(ctx, sessions)
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

// maybeRecordSnapshot classifies the current tick and decides whether it's
// actually written to history. An incident (a new blocking relationship
// appearing) is always recorded immediately — that's exactly the moment
// worth keeping. A periodic tick is only recorded once recordInterval has
// elapsed since the last write, decoupling disk writes from the (much
// faster) live poll interval.
func (p *Poller) maybeRecordSnapshot(ctx context.Context, sessions []domain.Session) {
	trigger := p.classifyTrigger(sessions)

	isDue := trigger == domain.SnapshotTriggerIncident || time.Since(p.lastRecordedAt) >= p.recordInterval
	if !isDue {
		return
	}

	snapshot := domain.Snapshot{
		Sessions:   trimSessions(sessions, trigger, p.maxSessionsPerSnapshot),
		CapturedAt: time.Now(),
		Trigger:    trigger,
	}

	if err := p.historyStore.Append(ctx, snapshot); err != nil {
		slog.Error("failed to record snapshot", "error", err)
		return
	}
	p.lastRecordedAt = time.Now()
}

// trimSessions bounds a periodic snapshot's session list to maxSessions,
// keeping the longest-running sessions when there are more active sessions
// than the cap allows — this keeps a periodic snapshot's size (and
// therefore disk growth) independent of how busy the monitored database
// is. Incident snapshots are never trimmed: understanding a blocking chain
// requires every session involved, not just the longest-running ones.
func trimSessions(sessions []domain.Session, trigger domain.SnapshotTrigger, maxSessions int) []domain.Session {
	if trigger != domain.SnapshotTriggerPeriodic || len(sessions) <= maxSessions {
		return sessions
	}

	sorted := make([]domain.Session, len(sessions))
	copy(sorted, sessions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Duration > sorted[j].Duration
	})

	return sorted[:maxSessions]
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

func (p *Poller) RecentHistory(ctx context.Context, since time.Time) ([]domain.Snapshot, error) {
	return p.historyStore.Recent(ctx, since)
}

func (p *Poller) Incidents(ctx context.Context) ([]domain.Snapshot, error) {
	return p.historyStore.Incidents(ctx)
}
