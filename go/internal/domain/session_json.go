package domain

import (
	"encoding/json"
	"time"
)

// sessionJSON mirrors Session for wire serialization, exposing Duration as
// fractional seconds instead of a raw time.Duration (nanoseconds), which is
// awkward for frontend consumers to work with.
type sessionJSON struct {
	ID              string         `json:"id"`
	User            string         `json:"user"`
	ApplicationName string         `json:"applicationName"`
	ClientAddress   string         `json:"clientAddress"`
	State           SessionState   `json:"state"`
	WaitEventType   WaitEventType  `json:"waitEventType"`
	WaitEvent       string         `json:"waitEvent"`
	Query           string         `json:"query"`
	Operation       QueryOperation `json:"operation"`
	QueryStarted    time.Time      `json:"queryStarted"`
	DurationSeconds float64        `json:"durationSeconds"`
	BlockedBy       []string       `json:"blockedBy"`
	Locks           []LockedObject `json:"locks"`
}

func (s Session) MarshalJSON() ([]byte, error) {
	return json.Marshal(sessionJSON{
		ID:              s.ID,
		User:            s.User,
		ApplicationName: s.ApplicationName,
		ClientAddress:   s.ClientAddress,
		State:           s.State,
		WaitEventType:   s.WaitEventType,
		WaitEvent:       s.WaitEvent,
		Query:           s.Query,
		Operation:       s.Operation,
		QueryStarted:    s.QueryStarted,
		DurationSeconds: s.Duration.Seconds(),
		BlockedBy:       s.BlockedBy,
		Locks:           s.Locks,
	})
}
