package alarm

import (
	"sync"
	"time"

	"platformscreendoor/internal/door"
	"platformscreendoor/internal/event"
	"platformscreendoor/internal/record"
)

type Policy struct {
	Levels map[string]Level
}

func DefaultPolicy() *Policy {
	return &Policy{Levels: map[string]Level{
		"trap":            LevelCritical,
		"drive_failure":   LevelCritical,
		"confirm_failure": LevelWarn,
		"offline":         LevelWarn,
	}}
}

func (p *Policy) LevelFor(kind string) Level {
	if level, ok := p.Levels[kind]; ok {
		return level
	}
	return LevelInfo
}

type Manager struct {
	mu         sync.Mutex
	suppressed map[string]bool
	history    []Alarm
	seq        uint64
	heartbeats map[string]time.Time
	rec        *record.Recorder
	dispatch   *Dispatcher
	policy     *Policy
}

func NewManager(rec *record.Recorder, dispatch *Dispatcher, policy *Policy) *Manager {
	return &Manager{
		suppressed: make(map[string]bool),
		heartbeats: make(map[string]time.Time),
		rec:        rec,
		dispatch:   dispatch,
		policy:     policy,
	}
}

func (m *Manager) Raise(doorID string, kind string) error {
	m.mu.Lock()
	key := doorID + ":" + kind
	if m.suppressed[key] {
		m.mu.Unlock()
		return nil
	}
	m.suppressed[key] = true
	m.seq++
	a := Alarm{DoorID: doorID, Kind: kind, Level: m.policy.LevelFor(kind), Seq: m.seq, At: time.Now()}
	m.history = append(m.history, a)
	m.mu.Unlock()
	if m.dispatch != nil {
		m.dispatch.Dispatch(a)
	}
	if m.rec != nil {
		_ = m.rec.Append(event.Event{DoorID: doorID, Type: event.TypeAlarm, Detail: kind})
	}
	return nil
}

func (m *Manager) Clear(doorID string, kind string) {
	m.mu.Lock()
	delete(m.suppressed, doorID+":"+kind)
	m.mu.Unlock()
	if m.dispatch != nil {
		m.dispatch.Dispatch(Alarm{DoorID: doorID, Kind: "clear_" + kind, Seq: m.seq, At: time.Now()})
	}
}

func (m *Manager) Heartbeat(doorID string, at time.Time) {
	m.mu.Lock()
	m.heartbeats[doorID] = at
	m.mu.Unlock()
}

func (m *Manager) LastHeartbeat(doorID string) (time.Time, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	last, ok := m.heartbeats[doorID]
	return last, ok
}

func (m *Manager) Offline(doorID string, now time.Time, ttl time.Duration, jitter time.Duration) bool {
	m.mu.Lock()
	last, ok := m.heartbeats[doorID]
	m.mu.Unlock()
	if !ok {
		return true
	}
	return now.Sub(last) > ttl
}

func (m *Manager) History(doorID string) []Alarm {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Alarm, 0)
	for _, a := range m.history {
		if a.DoorID == doorID {
			out = append(out, a)
		}
	}
	return out
}

func (m *Manager) Count(kind string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, a := range m.history {
		if a.Kind == kind {
			count++
		}
	}
	return count
}

func (m *Manager) All() []Alarm {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Alarm, len(m.history))
	copy(out, m.history)
	return out
}

func (m *Manager) DefaultJitter() time.Duration {
	return door.DefaultHeartbeatJitter
}

func (m *Manager) ActiveAlarmCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.suppressed)
}

func (m *Manager) ResolveAll(doorID string) {
	m.mu.Lock()
	for key := range m.suppressed {
		if hasDoorPrefix(key, doorID) {
			delete(m.suppressed, key)
		}
	}
	m.mu.Unlock()
}

func hasDoorPrefix(key string, doorID string) bool {
	return len(key) > len(doorID) && key[:len(doorID)] == doorID && key[len(doorID)] == ':'
}

func (m *Manager) OnState(doorID string, state door.State) {
	if m.dispatch != nil {
		m.dispatch.Dispatch(Alarm{DoorID: doorID, Kind: "state_" + state.String(), Seq: m.seq, At: time.Now()})
	}
}

func (m *Manager) OnTrap(doorID string) {
	_ = m.Raise(doorID, "trap")
}

func (m *Manager) OnAlarm(doorID string, kind string) {
	_ = m.Raise(doorID, kind)
}

func (m *Manager) OnAlarmClear(doorID string) {
	m.Clear(doorID, "trap")
	m.Clear(doorID, "drive_failure")
	m.Clear(doorID, "confirm_failure")
	m.Clear(doorID, "offline")
}
