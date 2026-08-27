package anti

import (
	"context"
	"sync"
	"time"

	"platformscreendoor/internal/alarm"
)

type Monitor struct {
	mu       sync.Mutex
	interval time.Duration
	alarms   *alarm.Manager
	blocked  map[string]bool
	samples  []Sample
}

func NewMonitor(interval time.Duration, alarms *alarm.Manager) *Monitor {
	return &Monitor{
		interval: interval,
		alarms:   alarms,
		blocked:  make(map[string]bool),
	}
}

func (m *Monitor) Interval() time.Duration {
	return m.interval
}

func (m *Monitor) SetTrap(doorID string, trapped bool) {
	m.mu.Lock()
	m.blocked[doorID] = trapped
	m.mu.Unlock()
}

func (m *Monitor) Release(doorID string) {
	m.mu.Lock()
	m.blocked[doorID] = false
	m.mu.Unlock()
}

func (m *Monitor) Sample(ctx context.Context, doorID string) bool {
	select {
	case <-ctx.Done():
		return false
	default:
	}
	m.mu.Lock()
	trapped := m.blocked[doorID]
	m.samples = append(m.samples, Sample{DoorID: doorID, Blocked: trapped, At: time.Now()})
	m.mu.Unlock()
	if trapped {
		m.alarms.Raise(doorID, "trap")
	}
	return trapped
}

func (m *Monitor) Snapshot() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	state := NewState()
	state.Count = len(m.samples)
	for doorID, blocked := range m.blocked {
		state.Blocked[doorID] = blocked
	}
	return *state
}

func (m *Monitor) BlockedDoors() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0)
	for doorID, blocked := range m.blocked {
		if blocked {
			out = append(out, doorID)
		}
	}
	return out
}

func (m *Monitor) SampleCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.samples)
}
