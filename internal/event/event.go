package event

import (
	"sync"
	"time"
)

type Type string

const (
	TypeOpen        Type = "open"
	TypeClose       Type = "close"
	TypeOpenFailed  Type = "open_failed"
	TypeCloseFailed Type = "close_failed"
	TypeLocked      Type = "locked"
	TypeStopped     Type = "stopped"
	TypeTrap        Type = "trap"
	TypeReset       Type = "reset"
	TypeLocalOn     Type = "local_on"
	TypeLocalOff    Type = "local_off"
	TypeOnline      Type = "online"
	TypeOffline     Type = "offline"
	TypeAlarm       Type = "alarm"
	TypeAlarmClear  Type = "alarm_clear"
	TypeGrant       Type = "grant"
	TypeRelease     Type = "release"
	TypeTrainDock   Type = "train_dock"
	TypeTrainLeave  Type = "train_leave"
)

type Event struct {
	Seq    uint64
	DoorID string
	Type   Type
	Detail string
	At     time.Time
}

type Bus struct {
	mu     sync.Mutex
	seq    uint64
	subs   []chan Event
	closed bool
}

func New() *Bus {
	return &Bus{}
}

func (b *Bus) Publish(e Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.seq++
	e.Seq = b.seq
	if e.At.IsZero() {
		e.At = time.Now()
	}
	for _, sub := range b.subs {
		select {
		case sub <- e:
		default:
		}
	}
}

func (b *Bus) Subscribe() <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	sub := make(chan Event, 64)
	b.subs = append(b.subs, sub)
	return sub
}

func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for _, sub := range b.subs {
		close(sub)
	}
	b.subs = nil
}

func (b *Bus) Latest() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.seq
}

func (b *Bus) Ordered(items []Event) []Event {
	sorted := make([]Event, len(items))
	copy(sorted, items)
	return sorted
}

func (b *Bus) Subscribers() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs)
}

func (b *Bus) Drain(sub <-chan Event, limit int) []Event {
	out := make([]Event, 0, limit)
	for len(out) < limit {
		select {
		case e, ok := <-sub:
			if !ok {
				return out
			}
			out = append(out, e)
		default:
			return out
		}
	}
	return out
}
