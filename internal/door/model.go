package door

import (
	"context"
	"sync"
	"time"

	"platformscreendoor/internal/event"
)

type State int

const (
	StateClosed State = iota
	StateOpening
	StateOpen
	StateClosing
	StateStopped
	StateFault
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpening:
		return "opening"
	case StateOpen:
		return "open"
	case StateClosing:
		return "closing"
	case StateStopped:
		return "stopped"
	default:
		return "fault"
	}
}

type Config struct {
	OpenTimeout     time.Duration
	CloseTimeout    time.Duration
	TrapInterval    time.Duration
	HeartbeatTTL    time.Duration
	HeartbeatJitter time.Duration
}

func NormalizeConfig(cfg Config) Config {
	if cfg.OpenTimeout <= 0 {
		cfg.OpenTimeout = DefaultOpenTimeout
	}
	if cfg.CloseTimeout <= 0 {
		cfg.CloseTimeout = DefaultCloseTimeout
	}
	if cfg.TrapInterval <= 0 {
		cfg.TrapInterval = DefaultTrapInterval
	}
	if cfg.HeartbeatTTL <= 0 {
		cfg.HeartbeatTTL = DefaultHeartbeatTTL
	}
	if cfg.HeartbeatJitter <= 0 {
		cfg.HeartbeatJitter = DefaultHeartbeatJitter
	}
	return cfg
}

type Grant struct {
	DoorID  string
	TrainID string
	Seq     uint64
}

type Observer interface {
	OnState(doorID string, state State)
	OnTrap(doorID string)
	OnAlarm(doorID string, kind string)
	OnAlarmClear(doorID string)
}

type Recorder interface {
	Begin(session string) error
	Append(event.Event) error
	Done(session string) error
}

type Actuator interface {
	Drive(id string, open bool) error
	Confirm(id string, open bool) (bool, error)
}

type TrapSampler interface {
	Sample(ctx context.Context, doorID string) bool
}

type GrantReleaser interface {
	ReleaseDoor(doorID string)
}

type Door struct {
	ID       string
	Side     int
	Index    int
	Config   Config
	mu       sync.RWMutex
	state    State
	isLocked bool
	obstacle bool
	local    bool
	online   bool
	lastSeq  uint64
}

type View struct {
	ID       string
	Side     int
	Index    int
	Config   Config
	State    State
	IsLocked bool
	Obstacle bool
	Local    bool
	Online   bool
}

func New(id string, side, index int, cfg Config) *Door {
	return &Door{
		ID:    id,
		Side:  side,
		Index: index,
		Config: cfg,
		state: StateClosed,
		online: true,
	}
}

func (d *Door) State() State {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.state
}

func (d *Door) IsLocked() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.isLocked
}

func (d *Door) Obstacle() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.obstacle
}

func (d *Door) Local() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.local
}

func (d *Door) Online() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.online
}

func (d *Door) View() View {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return View{
		ID:       d.ID,
		Side:     d.Side,
		Index:    d.Index,
		Config:   d.Config,
		State:    d.state,
		IsLocked: d.isLocked,
		Obstacle: d.obstacle,
		Local:    d.local,
		Online:   d.online,
	}
}

func (d *Door) setState(state State) {
	d.mu.Lock()
	d.state = state
	d.mu.Unlock()
}

func (d *Door) SetLocal(local bool) {
	d.mu.Lock()
	d.local = local
	d.mu.Unlock()
}

func (d *Door) SetOnline(online bool) {
	d.mu.Lock()
	d.online = online
	d.mu.Unlock()
}

func (d *Door) SetObstacle(obstacle bool) {
	d.mu.Lock()
	d.obstacle = obstacle
	d.mu.Unlock()
}
