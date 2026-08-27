package console

import (
	"context"
	"errors"
	"sync"
	"time"

	"platformscreendoor/internal/alarm"
	"platformscreendoor/internal/anti"
	"platformscreendoor/internal/door"
	"platformscreendoor/internal/event"
	"platformscreendoor/internal/interlock"
	"platformscreendoor/internal/record"
	"platformscreendoor/internal/train"
)

var (
	ErrDoorNotFound  = errors.New("door not found")
	ErrTrainNotFound = errors.New("train not found")
)

type Deps struct {
	Store        *door.Store
	Interlock    *interlock.Interlock
	Alarms       *alarm.Manager
	AlarmHandler *alarm.Handler
	Monitor      *anti.Monitor
	AntiHandler  *anti.Handler
	Recorder     *record.Recorder
	Records      *record.Store
	Bus          *event.Bus
	Actuator     door.Actuator
	Sampler      door.TrapSampler
	Trains       *train.Controller
	Observer     door.Observer
}

type Service struct {
	deps   Deps
	mu     sync.Mutex
	seq    uint64
	trains map[string]*train.Train
	cursor *event.Topic
	cmds   []Command
}

func NewService(deps Deps) *Service {
	return &Service{
		deps:   deps,
		trains: make(map[string]*train.Train),
		cursor: event.NewTopic("console"),
	}
}

func (s *Service) nextSeq() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return s.seq
}

func (s *Service) logCommand(kind string, doorID string, detail string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	s.cmds = append(s.cmds, Command{Seq: s.seq, Kind: kind, DoorID: doorID, Detail: detail, At: time.Now()})
	if len(s.cmds) > 200 {
		s.cmds = s.cmds[len(s.cmds)-200:]
	}
}

func (s *Service) RecentCommands(limit int) []Command {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 || limit > len(s.cmds) {
		limit = len(s.cmds)
	}
	out := make([]Command, 0, limit)
	for i := len(s.cmds) - limit; i < len(s.cmds); i++ {
		out = append(out, s.cmds[i])
	}
	return out
}

func (s *Service) Stats() map[string]int {
	return map[string]int{
		"records":     s.deps.Records.Size(),
		"alarms":      s.deps.Alarms.ActiveAlarmCount(),
		"handles":     s.deps.Recorder.OpenCount(),
		"streams":     len(s.deps.Recorder.Streams()),
		"grants":      len(s.deps.Interlock.Snapshot()),
		"outstanding": len(s.deps.Interlock.Outstanding()),
		"trapped":     len(s.deps.Monitor.BlockedDoors()),
		"samples":     s.deps.Monitor.SampleCount(),
		"interval_ms": int(s.deps.Monitor.Interval() / time.Millisecond),
		"event_types": len(s.deps.Records.Summary()),
		"subscribers": s.deps.Bus.Subscribers(),
	}
}

func (s *Service) OpenDoor(doorID string, trainID string) error {
	d, ok := s.deps.Store.Get(doorID)
	if !ok {
		return ErrDoorNotFound
	}
	t, ok := s.trains[trainID]
	if !ok {
		return ErrTrainNotFound
	}
	// Acquire the grant atomically with the interlock check. Two trains
	// requesting the same door near-simultaneously cannot both succeed: the
	// second is rejected with "door busy" and the first train's grant is never
	// overwritten. This prevents a later train's request from clobbering an
	// already-released (open) door's grant, which previously made the door
	// flicker between open and closed on the console.
	grant, res := s.deps.Interlock.AcquireGrant(d.View(), t.Snapshot(), s.nextSeq())
	if !res.Allowed {
		return errors.New(res.Reason)
	}
	if err := d.ApplyGrant(grant); err != nil {
		// The door could not transition to opening (e.g. it was opened by
		// another flow between our check and here). Roll the grant back so the
		// door is not left recorded as granted to a train that did not take it,
		// and so a later request is not blocked by a stale grant.
		s.deps.Interlock.ReleaseDoor(doorID)
		return err
	}
	// Only publish the grant event once the door state has actually advanced
	// to opening. Publishing before ApplyGrant caused the console/event stream
	// to show the door as released (open) and then snap back to closed when the
	// apply failed, i.e. the "放行又变回关闭" flicker.
	if s.deps.Bus != nil {
		s.deps.Bus.Publish(event.Event{DoorID: doorID, Type: event.TypeGrant, Detail: trainID})
	}
	s.logCommand("open", doorID, trainID)
	return nil
}

func (s *Service) ConfirmOpen(doorID string) error {
	grant, ok := s.deps.Interlock.GrantFor(doorID)
	if !ok {
		return door.ErrNoGrant
	}
	target, ok := s.deps.Store.Get(grant.DoorID)
	if !ok {
		return ErrDoorNotFound
	}
	if err := target.ConfirmGrant(grant); err != nil {
		if s.deps.Recorder != nil {
			_ = s.deps.Recorder.Append(event.Event{DoorID: target.ID, Type: event.TypeOpenFailed, Detail: grant.TrainID})
		}
		return err
	}
	if s.deps.Recorder != nil {
		_ = s.deps.Recorder.Append(event.Event{DoorID: target.ID, Type: event.TypeOpen, Detail: "granted:" + grant.TrainID})
	}
	s.logCommand("confirm_open", doorID, grant.TrainID)
	return nil
}

func (s *Service) CloseDoor(ctx context.Context, doorID string) error {
	d, ok := s.deps.Store.Get(doorID)
	if !ok {
		return ErrDoorNotFound
	}
	timeout := door.CloseTimeoutFor(d.Config)
	if d.Config.CloseTimeout <= 0 {
		timeout = s.deps.Interlock.CloseBudget()
	}
	if err := d.Close(ctx, s.deps.Actuator, s.deps.Sampler, s.deps.Interlock, s.deps.Observer, s.deps.Recorder, timeout); err != nil {
		if s.deps.Recorder != nil {
			_ = s.deps.Recorder.Append(event.Event{DoorID: d.ID, Type: event.TypeCloseFailed, Detail: "close"})
		}
		return err
	}
	s.logCommand("close", doorID, "auto")
	return nil
}

func (s *Service) CloseBatch(ctx context.Context, doorIDs []string) []error {
	doors := make([]*door.Door, 0, len(doorIDs))
	for _, id := range doorIDs {
		d, ok := s.deps.Store.Get(id)
		if !ok {
			continue
		}
		doors = append(doors, d)
	}
	s.deps.Interlock.BeginChain()
	errs := door.CloseBatch(ctx, doors, s.deps.Actuator, s.deps.Sampler, s.deps.Interlock, s.deps.Observer, s.deps.Recorder, door.DefaultCloseTimeout)
	s.deps.Interlock.ReleaseChain()
	for _, d := range doors {
		if s.deps.Recorder != nil {
			_ = s.deps.Recorder.Append(event.Event{DoorID: d.ID, Type: event.TypeRelease, Detail: "chain"})
		}
		s.logCommand("close_batch", d.ID, "auto")
	}
	return errs
}

func (s *Service) SetLocal(doorID string, local bool) error {
	d, ok := s.deps.Store.Get(doorID)
	if !ok {
		return ErrDoorNotFound
	}
	d.SetLocal(local)
	kind := event.TypeLocalOff
	if local {
		kind = event.TypeLocalOn
	}
	if s.deps.Bus != nil {
		s.deps.Bus.Publish(event.Event{DoorID: doorID, Type: kind, Detail: "console"})
	}
	s.logCommand("local", doorID, "console")
	return nil
}

func (s *Service) ResetDoor(doorID string) error {
	d, ok := s.deps.Store.Get(doorID)
	if !ok {
		return ErrDoorNotFound
	}
	if s.deps.AntiHandler != nil {
		s.deps.AntiHandler.ClearTrap(doorID)
	}
	d.SetObstacle(false)
	if err := d.Reset(s.deps.Observer); err != nil {
		return err
	}
	if s.deps.Recorder != nil {
		_ = s.deps.Recorder.Append(event.Event{DoorID: doorID, Type: event.TypeReset, Detail: "console"})
	}
	s.deps.Alarms.ResolveAll(doorID)
	s.logCommand("reset", doorID, "console")
	return nil
}

func (s *Service) ResetAll(doorIDs []string) []error {
	errs := make([]error, 0)
	if s.deps.AntiHandler != nil {
		s.deps.AntiHandler.ReleaseAll(doorIDs)
	}
	for _, doorID := range doorIDs {
		if err := s.ResetDoor(doorID); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (s *Service) Heartbeat(doorID string, at time.Time) {
	if s.deps.Alarms == nil {
		return
	}
	last, _ := s.deps.Alarms.LastHeartbeat(doorID)
	s.deps.Alarms.Heartbeat(doorID, at)
	d, ok := s.deps.Store.Get(doorID)
	if !ok {
		return
	}
	online := d.OnlineCheck(last, at)
	d.SetOnline(online)
	if s.deps.Bus != nil {
		kind := event.TypeOffline
		if online {
			kind = event.TypeOnline
		}
		s.deps.Bus.Publish(event.Event{DoorID: doorID, Type: kind, Detail: "heartbeat"})
	}
	s.logCommand("heartbeat", doorID, "beat")
}

func (s *Service) TrainDocked(t *train.Train, aligned bool) {
	copyTrain := t.Snapshot()
	existing, ok := s.trains[copyTrain.ID]
	if !ok {
		existing = train.New(copyTrain.ID, copyTrain.LineID)
		s.trains[copyTrain.ID] = existing
	}
	existing.Docked = copyTrain.Docked
	existing.Aligned = aligned
	existing.DoorMap = copyTrain.DoorMap
	if s.deps.Trains != nil {
		s.deps.Trains.Dock(existing, aligned)
	}
	s.logCommand("train_dock", existing.ID, existing.LineID)
}

func (s *Service) TrainLeave(id string) {
	if t, ok := s.trains[id]; ok {
		if s.deps.Trains != nil {
			s.deps.Trains.Leave(t)
		}
		s.logCommand("train_leave", id, t.LineID)
	}
}

type SimActuator struct {
	mu    sync.Mutex
	delay time.Duration
}

func NewSimActuator(delay time.Duration) *SimActuator {
	return &SimActuator{delay: delay}
}

func (a *SimActuator) Drive(id string, open bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return nil
}

func (a *SimActuator) Confirm(id string, open bool) (bool, error) {
	time.Sleep(a.delay)
	return true, nil
}
