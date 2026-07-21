package service

import (
	"sync"
	"time"
)

// Recorder gates whether an activity (live monitoring, or history
// recording) is currently active. It is off by default — activation is an
// explicit, opt-in action, either for a fixed duration or indefinitely
// ("full") until manually stopped.
type Recorder struct {
	mu       sync.Mutex
	active   bool
	stopTime *time.Timer
}

func NewRecorder() *Recorder {
	return &Recorder{}
}

// Start begins the activity. A duration of 0 means "run indefinitely" —
// the caller must call Stop explicitly. A positive duration auto-stops
// after that time elapses.
func (r *Recorder) Start(duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stopExistingTimerLocked()
	r.active = true

	if duration > 0 {
		r.stopTime = time.AfterFunc(duration, func() {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.active = false
		})
	}
}

func (r *Recorder) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stopExistingTimerLocked()
	r.active = false
}

func (r *Recorder) IsActive() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active
}

func (r *Recorder) stopExistingTimerLocked() {
	if r.stopTime != nil {
		r.stopTime.Stop()
		r.stopTime = nil
	}
}
