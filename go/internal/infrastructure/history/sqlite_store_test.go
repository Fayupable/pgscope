package history

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fayupable/pgscope/internal/domain"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pgscope_test.db")
	store, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestSQLiteStore_AppendAndRecent(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	first := domain.Snapshot{
		Sessions:   []domain.Session{{ID: "1", User: "app", Query: "SELECT 1", Duration: 2 * time.Second}},
		CapturedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		Trigger:    domain.SnapshotTriggerPeriodic,
	}
	second := domain.Snapshot{
		Sessions:   []domain.Session{{ID: "2", User: "app", Query: "SELECT 2", Duration: 3 * time.Second}},
		CapturedAt: time.Date(2026, 1, 1, 10, 0, 30, 0, time.UTC),
		Trigger:    domain.SnapshotTriggerPeriodic,
	}

	if err := store.Append(ctx, first); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	if err := store.Append(ctx, second); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}

	got, err := store.Recent(ctx, time.Time{})
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Recent() returned %d snapshots, want 2", len(got))
	}

	if !got[0].CapturedAt.Equal(first.CapturedAt) {
		t.Errorf("Recent()[0].CapturedAt = %v, want %v (chronological order)", got[0].CapturedAt, first.CapturedAt)
	}
	if !got[1].CapturedAt.Equal(second.CapturedAt) {
		t.Errorf("Recent()[1].CapturedAt = %v, want %v (chronological order)", got[1].CapturedAt, second.CapturedAt)
	}
}

func TestSQLiteStore_DurationRoundTrips(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	snapshot := domain.Snapshot{
		Sessions: []domain.Session{
			{ID: "1", Duration: 2500 * time.Millisecond},
		},
		CapturedAt: time.Now().UTC(),
		Trigger:    domain.SnapshotTriggerPeriodic,
	}

	if err := store.Append(ctx, snapshot); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	got, err := store.Recent(ctx, time.Time{})
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}
	if len(got) != 1 || len(got[0].Sessions) != 1 {
		t.Fatalf("Recent() = %+v, want exactly 1 snapshot with 1 session", got)
	}

	gotDuration := got[0].Sessions[0].Duration
	if gotDuration != 2500*time.Millisecond {
		t.Errorf("Duration round-trip = %v, want %v", gotDuration, 2500*time.Millisecond)
	}
}

func TestSQLiteStore_RedactsQueryBeforePersisting(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	snapshot := domain.Snapshot{
		Sessions: []domain.Session{
			{ID: "1", Query: "SELECT * FROM users WHERE email = 'ali@example.com'"},
		},
		CapturedAt: time.Now().UTC(),
		Trigger:    domain.SnapshotTriggerPeriodic,
	}

	if err := store.Append(ctx, snapshot); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	got, err := store.Recent(ctx, time.Time{})
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}
	if len(got) != 1 || len(got[0].Sessions) != 1 {
		t.Fatalf("Recent() = %+v, want exactly 1 snapshot with 1 session", got)
	}

	gotQuery := got[0].Sessions[0].Query
	if strings.Contains(gotQuery, "ali@example.com") {
		t.Errorf("Query = %q, want the literal email redacted before persisting", gotQuery)
	}
	if !strings.Contains(gotQuery, "***") {
		t.Errorf("Query = %q, want it to contain the *** redaction marker", gotQuery)
	}
}

func TestSQLiteStore_Incidents(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	periodic := domain.Snapshot{
		Sessions:   []domain.Session{{ID: "1"}},
		CapturedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		Trigger:    domain.SnapshotTriggerPeriodic,
	}
	incident := domain.Snapshot{
		Sessions:   []domain.Session{{ID: "2"}},
		CapturedAt: time.Date(2026, 1, 1, 10, 5, 0, 0, time.UTC),
		Trigger:    domain.SnapshotTriggerIncident,
	}

	if err := store.Append(ctx, periodic); err != nil {
		t.Fatalf("Append(periodic) error = %v", err)
	}
	if err := store.Append(ctx, incident); err != nil {
		t.Fatalf("Append(incident) error = %v", err)
	}

	got, err := store.Incidents(ctx)
	if err != nil {
		t.Fatalf("Incidents() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Incidents() returned %d snapshots, want 1", len(got))
	}
	if got[0].Trigger != domain.SnapshotTriggerIncident {
		t.Errorf("Incidents()[0].Trigger = %v, want %v", got[0].Trigger, domain.SnapshotTriggerIncident)
	}
	if len(got[0].Sessions) != 1 || got[0].Sessions[0].ID != "2" {
		t.Errorf("Incidents()[0] = %+v, want the incident snapshot, not the periodic one", got[0])
	}
}

func TestSQLiteStore_PruneOlderThan(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	old := domain.Snapshot{
		Sessions:   []domain.Session{{ID: "old"}},
		CapturedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Trigger:    domain.SnapshotTriggerPeriodic,
	}
	recent := domain.Snapshot{
		Sessions:   []domain.Session{{ID: "recent"}},
		CapturedAt: time.Now().UTC(),
		Trigger:    domain.SnapshotTriggerPeriodic,
	}
	oldIncident := domain.Snapshot{
		Sessions:   []domain.Session{{ID: "old_incident"}},
		CapturedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Trigger:    domain.SnapshotTriggerIncident,
	}

	for _, s := range []domain.Snapshot{old, recent, oldIncident} {
		if err := store.Append(ctx, s); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	cutoff := time.Now().Add(-24 * time.Hour)
	if err := store.PruneOlderThan(ctx, cutoff); err != nil {
		t.Fatalf("PruneOlderThan() error = %v", err)
	}

	// Recent() returns every snapshot regardless of trigger (mirroring
	// RingBufferStore's semantics), so the surviving incident snapshot is
	// still expected here — only the pruned periodic snapshot should be gone.
	remaining, err := store.Recent(ctx, time.Time{})
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("Recent() after prune returned %d snapshots, want 2 (recent periodic + old incident)", len(remaining))
	}
	remainingIDs := map[string]bool{}
	for _, snap := range remaining {
		remainingIDs[snap.Sessions[0].ID] = true
	}
	if !remainingIDs["recent"] || !remainingIDs["old_incident"] {
		t.Fatalf("Recent() after prune = %+v, want the recent periodic and old_incident snapshots, not the pruned old one", remaining)
	}
	if remainingIDs["old"] {
		t.Fatalf("Recent() after prune still contains the old periodic snapshot, want it pruned")
	}

	incidents, err := store.Incidents(ctx)
	if err != nil {
		t.Fatalf("Incidents() error = %v", err)
	}
	if len(incidents) != 1 {
		t.Fatalf("Incidents() after prune returned %d, want 1 (incidents are never pruned by age)", len(incidents))
	}
}

func TestSQLiteStore_PruneExcessIncidents(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		incident := domain.Snapshot{
			Sessions:   []domain.Session{{ID: string(rune('a' + i))}},
			CapturedAt: base.Add(time.Duration(i) * time.Minute),
			Trigger:    domain.SnapshotTriggerIncident,
		}
		if err := store.Append(ctx, incident); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	if err := store.PruneExcessIncidents(ctx, 3); err != nil {
		t.Fatalf("PruneExcessIncidents() error = %v", err)
	}

	got, err := store.Incidents(ctx)
	if err != nil {
		t.Fatalf("Incidents() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("Incidents() after PruneExcessIncidents(3) returned %d, want 3", len(got))
	}

	// The 3 most recent (i=2,3,4) should survive, oldest two (i=0,1) pruned.
	cutoff := base.Add(2 * time.Minute)
	for _, snap := range got {
		if snap.CapturedAt.Before(cutoff) {
			t.Errorf("survived incident has CapturedAt = %v, want it at or after %v (the 3 most recent)", snap.CapturedAt, cutoff)
		}
	}
}

func TestSQLiteStore_PruneExcessIncidents_NoopWhenUnderLimit(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	incident := domain.Snapshot{
		Sessions:   []domain.Session{{ID: "1"}},
		CapturedAt: time.Now().UTC(),
		Trigger:    domain.SnapshotTriggerIncident,
	}
	if err := store.Append(ctx, incident); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	if err := store.PruneExcessIncidents(ctx, 100); err != nil {
		t.Fatalf("PruneExcessIncidents() error = %v", err)
	}

	got, err := store.Incidents(ctx)
	if err != nil {
		t.Fatalf("Incidents() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Incidents() after no-op prune returned %d, want 1", len(got))
	}
}

func TestSQLiteStore_Recent_FiltersBySince(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tooOld := domain.Snapshot{
		Sessions:   []domain.Session{{ID: "too_old"}},
		CapturedAt: base,
		Trigger:    domain.SnapshotTriggerPeriodic,
	}
	inWindow := domain.Snapshot{
		Sessions:   []domain.Session{{ID: "in_window"}},
		CapturedAt: base.Add(2 * time.Hour),
		Trigger:    domain.SnapshotTriggerPeriodic,
	}

	if err := store.Append(ctx, tooOld); err != nil {
		t.Fatalf("Append(tooOld) error = %v", err)
	}
	if err := store.Append(ctx, inWindow); err != nil {
		t.Fatalf("Append(inWindow) error = %v", err)
	}

	since := base.Add(time.Hour)
	got, err := store.Recent(ctx, since)
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Recent(since=%v) returned %d snapshots, want 1", since, len(got))
	}
	if got[0].Sessions[0].ID != "in_window" {
		t.Errorf("Recent(since=%v)[0] = %+v, want the in_window snapshot", since, got[0])
	}
}

func TestSQLiteStore_Recent_IncludesIncidentsRegardlessOfDownsampling(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		snap := domain.Snapshot{
			Sessions:   []domain.Session{{ID: "periodic"}},
			CapturedAt: base.Add(time.Duration(i) * time.Second),
			Trigger:    domain.SnapshotTriggerPeriodic,
		}
		if err := store.Append(ctx, snap); err != nil {
			t.Fatalf("Append(periodic) error = %v", err)
		}
	}
	incident := domain.Snapshot{
		Sessions:   []domain.Session{{ID: "incident"}},
		CapturedAt: base.Add(10 * time.Second),
		Trigger:    domain.SnapshotTriggerIncident,
	}
	if err := store.Append(ctx, incident); err != nil {
		t.Fatalf("Append(incident) error = %v", err)
	}

	got, err := store.Recent(ctx, base)
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}

	found := false
	for _, snap := range got {
		if snap.Trigger == domain.SnapshotTriggerIncident {
			found = true
		}
	}
	if !found {
		t.Errorf("Recent() = %+v, want the incident snapshot present regardless of periodic volume", got)
	}
}

func TestSQLiteStore_Recent_DownsamplesLargePeriodicWindow(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	total := maxPeriodicRowsPerWindow * 3
	for i := 0; i < total; i++ {
		snap := domain.Snapshot{
			Sessions:   []domain.Session{{ID: "s"}},
			CapturedAt: base.Add(time.Duration(i) * time.Second),
			Trigger:    domain.SnapshotTriggerPeriodic,
		}
		if err := store.Append(ctx, snap); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	got, err := store.Recent(ctx, base)
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}

	if len(got) > maxPeriodicRowsPerWindow {
		t.Errorf("Recent() returned %d snapshots, want at most %d (downsampled)", len(got), maxPeriodicRowsPerWindow)
	}
	if len(got) == 0 {
		t.Fatal("Recent() returned 0 snapshots, want a downsampled but non-empty result")
	}

	first := got[0].CapturedAt
	last := got[len(got)-1].CapturedAt
	spanCoveredFraction := last.Sub(first).Seconds() / float64(total)
	if spanCoveredFraction < 0.9 {
		t.Errorf("downsampled result only spans %.0f%% of the original window, want it spread across nearly the whole thing", spanCoveredFraction*100)
	}
}

func appendPaddedHistory(t *testing.T, ctx context.Context, store *SQLiteStore, base time.Time, periodicCount int) {
	t.Helper()
	for i := 0; i < periodicCount; i++ {
		snap := domain.Snapshot{
			Sessions: []domain.Session{
				{ID: "s", Query: strings.Repeat("x", 500)}, // pad each row so the file grows measurably
			},
			CapturedAt: base.Add(time.Duration(i) * time.Second),
			Trigger:    domain.SnapshotTriggerPeriodic,
		}
		if err := store.Append(ctx, snap); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}
	incident := domain.Snapshot{
		Sessions:   []domain.Session{{ID: "incident", Query: strings.Repeat("y", 500)}},
		CapturedAt: base.Add(time.Duration(periodicCount+1) * time.Second),
		Trigger:    domain.SnapshotTriggerIncident,
	}
	if err := store.Append(ctx, incident); err != nil {
		t.Fatalf("Append(incident) error = %v", err)
	}
}

func countByTrigger(snapshots []domain.Snapshot) (periodic, incident int) {
	for _, snap := range snapshots {
		if snap.Trigger == domain.SnapshotTriggerIncident {
			incident++
		} else {
			periodic++
		}
	}
	return periodic, incident
}

func TestSQLiteStore_EnforceMaxSize_PrunesPeriodicFirstAndLeavesIncidentsAlone(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// pruneBatchSize is 500 — use more than that so a single batch can't
	// possibly delete every periodic row at once, letting a "moderate cap"
	// scenario (satisfied by only partial pruning) actually be exercised.
	appendPaddedHistory(t, ctx, store, base, 1200)

	sizeBefore, err := store.fileSize()
	if err != nil {
		t.Fatalf("fileSize() error = %v", err)
	}

	// A moderate cap: enough pruning is needed to trigger enforcement, but
	// deleting only part of the periodic set should already bring the file
	// back under it — incidents should never be touched in this case.
	moderateCap := sizeBefore * 3 / 4
	if err := store.EnforceMaxSize(ctx, moderateCap); err != nil {
		t.Fatalf("EnforceMaxSize() error = %v", err)
	}

	remaining, err := store.Recent(ctx, base)
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}
	periodicRemaining, incidentRemaining := countByTrigger(remaining)

	if periodicRemaining >= 1200 {
		t.Errorf("periodic snapshots remaining = %d, want fewer than the original 1200 (enforcement should have pruned some)", periodicRemaining)
	}
	if incidentRemaining != 1 {
		t.Errorf("incident snapshots remaining = %d, want 1 (a moderate cap should never need to touch incidents)", incidentRemaining)
	}
}

func TestSQLiteStore_EnforceMaxSize_FallsBackToIncidentsWhenPeriodicAloneIsNotEnough(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	appendPaddedHistory(t, ctx, store, base, 20)

	// An impossibly small cap: even deleting every periodic snapshot can't
	// satisfy it, so EnforceMaxSize must fall back to pruning incidents too.
	const impossiblyTinyCap = 1
	if err := store.EnforceMaxSize(ctx, impossiblyTinyCap); err != nil {
		t.Fatalf("EnforceMaxSize() error = %v", err)
	}

	remaining, err := store.Recent(ctx, base)
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}
	periodicRemaining, incidentRemaining := countByTrigger(remaining)

	if periodicRemaining != 0 {
		t.Errorf("periodic snapshots remaining = %d, want 0 (an impossibly tiny cap should exhaust periodic first)", periodicRemaining)
	}
	if incidentRemaining != 0 {
		t.Errorf("incident snapshots remaining = %d, want 0 (once periodic is exhausted, an impossibly tiny cap must fall back to pruning incidents)", incidentRemaining)
	}
}
