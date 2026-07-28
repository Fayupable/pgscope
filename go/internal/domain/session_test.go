package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSession_IsBlocked(t *testing.T) {
	tests := []struct {
		name    string
		session Session
		want    bool
	}{
		{
			name:    "no blockers is not blocked",
			session: Session{BlockedBy: nil},
			want:    false,
		},
		{
			name:    "empty blocker slice is not blocked",
			session: Session{BlockedBy: []string{}},
			want:    false,
		},
		{
			name:    "one blocker is blocked",
			session: Session{BlockedBy: []string{"123"}},
			want:    true,
		},
		{
			name:    "multiple blockers is blocked",
			session: Session{BlockedBy: []string{"123", "456"}},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.IsBlocked(); got != tt.want {
				t.Errorf("IsBlocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSession_MarshalJSON(t *testing.T) {
	started := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	session := Session{
		ID:              "42",
		User:            "app",
		ApplicationName: "web",
		ClientAddress:   "10.0.0.5",
		State:           SessionStateActive,
		WaitEventType:   WaitEventTypeLock,
		WaitEvent:       "transactionid",
		Query:           "SELECT 1",
		Operation:       QueryOperationSelect,
		QueryStarted:    started,
		Duration:        2500 * time.Millisecond,
		BlockedBy:       []string{"7"},
		Locks:           []LockedObject{{NativeMode: "AccessExclusiveLock", Severity: LockSeverityExclusive, Granted: true}},
	}

	raw, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("failed to unmarshal produced JSON: %v", err)
	}

	if _, exists := decoded["duration"]; exists {
		t.Error(`decoded JSON has a raw "duration" field, want it replaced by "durationSeconds"`)
	}

	gotDurationSeconds, ok := decoded["durationSeconds"].(float64)
	if !ok {
		t.Fatalf("durationSeconds is missing or not a number: %v", decoded["durationSeconds"])
	}
	if gotDurationSeconds != 2.5 {
		t.Errorf("durationSeconds = %v, want 2.5", gotDurationSeconds)
	}

	if decoded["id"] != "42" {
		t.Errorf("id = %v, want %q", decoded["id"], "42")
	}
	if decoded["state"] != string(SessionStateActive) {
		t.Errorf("state = %v, want %q", decoded["state"], SessionStateActive)
	}
	if decoded["operation"] != string(QueryOperationSelect) {
		t.Errorf("operation = %v, want %q", decoded["operation"], QueryOperationSelect)
	}

	blockedBy, ok := decoded["blockedBy"].([]interface{})
	if !ok || len(blockedBy) != 1 || blockedBy[0] != "7" {
		t.Errorf("blockedBy = %v, want [\"7\"]", decoded["blockedBy"])
	}
}

func TestSession_MarshalJSON_ZeroDuration(t *testing.T) {
	raw, err := json.Marshal(Session{ID: "1", Duration: 0})
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("failed to unmarshal produced JSON: %v", err)
	}

	if decoded["durationSeconds"] != float64(0) {
		t.Errorf("durationSeconds = %v, want 0", decoded["durationSeconds"])
	}
}
