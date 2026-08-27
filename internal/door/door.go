package door

import (
	"context"
	"errors"
	"time"

	"platformscreendoor/internal/event"
)

var (
	ErrState         = errors.New("door state does not allow operation")
	ErrLocal         = errors.New("door is under local control")
	ErrObstacle      = errors.New("door obstacle detected")
	ErrConfirm       = errors.New("door confirmation timeout")
	ErrDrive         = errors.New("door drive failure")
	ErrNoGrant       = errors.New("door grant missing")
	ErrGrantMismatch = errors.New("door grant mismatch")
	ErrCancel        = errors.New("door operation cancelled")
)

var (
	DefaultOpenTimeout     = 6 * time.Second
	DefaultCloseTimeout    = 8 * time.Second
	DefaultTrapInterval    = 100 * time.Millisecond
	DefaultHeartbeatTTL    = 2 * time.Second
	DefaultHeartbeatJitter = 500 * time.Millisecond
)

func OpenTimeoutFor(cfg Config) time.Duration {
	if cfg.OpenTimeout > 0 {
		return cfg.OpenTimeout
	}
	return DefaultOpenTimeout
}

func CloseTimeoutFor(cfg Config) time.Duration {
	if cfg.CloseTimeout > 0 {
		return cfg.CloseTimeout
	}
	return DefaultCloseTimeout
}

func TrapIntervalFor(cfg Config) time.Duration {
	if cfg.TrapInterval > 0 {
		return cfg.TrapInterval
	}
	return DefaultTrapInterval
}

func notify(obs Observer, doorID string, state State) {
	if obs != nil {
		obs.OnState(doorID, state)
	}
}

func (d *Door) ValidateOpen() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.state != StateClosed {
		return ErrState
	}
	if d.local {
		return ErrLocal
	}
	if d.obstacle {
		return ErrObstacle
	}
	return nil
}

func (d *Door) ValidateClose() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.state != StateOpen {
		return ErrState
	}
	if d.local {
		return ErrLocal
	}
	return nil
}

func (d *Door) Open(act Actuator, obs Observer, rec Recorder) error {
	session := "open:" + d.ID
	if rec != nil {
		_ = rec.Begin(session)
		defer func() {
			_ = rec.Done(session)
		}()
	}
	if err := d.ValidateOpen(); err != nil {
		return err
	}
	d.mu.Lock()
	d.state = StateOpening
	d.mu.Unlock()
	notify(obs, d.ID, StateOpening)
	if err := act.Drive(d.ID, true); err != nil {
		d.setState(StateClosed)
		if obs != nil {
			obs.OnAlarm(d.ID, "drive_failure")
		}
		return err
	}
	confirmed, err := act.Confirm(d.ID, true)
	if err != nil {
		d.setState(StateClosed)
		if obs != nil {
			obs.OnAlarm(d.ID, "confirm_failure")
		}
		return err
	}
	if !confirmed {
		d.setState(StateClosed)
		return ErrConfirm
	}
	d.setState(StateOpen)
	notify(obs, d.ID, StateOpen)
	if rec != nil {
		_ = rec.Append(event.Event{DoorID: d.ID, Type: event.TypeOpen, Detail: "opened"})
	}
	return nil
}

func (d *Door) Close(ctx context.Context, act Actuator, sampler TrapSampler, rel GrantReleaser, obs Observer, rec Recorder, timeout time.Duration) error {
	session := "close:" + d.ID
	if rec != nil {
		_ = rec.Begin(session)
		defer func() {
			_ = rec.Done(session)
		}()
	}
	if err := d.ValidateClose(); err != nil {
		return err
	}
	d.mu.Lock()
	d.state = StateClosing
	d.isLocked = false
	d.mu.Unlock()
	notify(obs, d.ID, StateClosing)
	if err := act.Drive(d.ID, false); err != nil {
		d.setState(StateOpen)
		return err
	}
	interval := TrapIntervalFor(d.Config)
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			d.setState(StateStopped)
			if rel != nil {
				rel.ReleaseDoor(d.ID)
			}
			notify(obs, d.ID, StateStopped)
			return ErrCancel
		case <-timer.C:
			d.setState(StateStopped)
			if rel != nil {
				rel.ReleaseDoor(d.ID)
			}
			notify(obs, d.ID, StateStopped)
			return ErrConfirm
		case <-tick.C:
			if sampler != nil && sampler.Sample(ctx, d.ID) {
				d.setState(StateStopped)
				d.mu.Lock()
				d.obstacle = true
				d.mu.Unlock()
				if obs != nil {
					obs.OnTrap(d.ID)
					obs.OnAlarm(d.ID, "trap")
				}
				if rel != nil {
					rel.ReleaseDoor(d.ID)
				}
				notify(obs, d.ID, StateStopped)
				return ErrObstacle
			}
			if err := act.Drive(d.ID, false); err != nil {
				d.setState(StateOpen)
				return err
			}
			locked, err := act.Confirm(d.ID, false)
			if err != nil {
				d.setState(StateOpen)
				return err
			}
			if locked {
				d.mu.Lock()
				d.state = StateClosed
				d.isLocked = true
				d.mu.Unlock()
				if rel != nil {
					rel.ReleaseDoor(d.ID)
				}
				notify(obs, d.ID, StateClosed)
				if rec != nil {
					_ = rec.Append(event.Event{DoorID: d.ID, Type: event.TypeLocked, Detail: "closed"})
				}
				return nil
			}
		}
	}
}

func (d *Door) Stop() {
	d.setState(StateStopped)
}

func (d *Door) Reset(obs Observer) error {
	d.mu.Lock()
	if d.state != StateStopped && d.state != StateFault {
		d.mu.Unlock()
		return ErrState
	}
	if d.obstacle {
		d.mu.Unlock()
		return ErrObstacle
	}
	d.state = StateClosed
	d.isLocked = false
	d.obstacle = false
	d.mu.Unlock()
	if obs != nil {
		obs.OnAlarmClear(d.ID)
		notify(obs, d.ID, StateClosed)
	}
	return nil
}

func (d *Door) ApplyGrant(g Grant) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if g.DoorID != d.ID {
		return ErrGrantMismatch
	}
	if d.state != StateClosed {
		return ErrState
	}
	d.state = StateOpening
	return nil
}

func (d *Door) ConfirmGrant(g Grant) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if g.DoorID != d.ID {
		return ErrGrantMismatch
	}
	if d.state == StateOpening {
		d.state = StateOpen
	}
	return nil
}

func (d *Door) ApplyEvent(e event.Event) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e.Seq <= d.lastSeq {
		return nil
	}
	d.lastSeq = e.Seq
	switch e.Type {
	case event.TypeOpen:
		if d.state == StateClosed || d.state == StateStopped {
			d.state = StateOpen
		}
	case event.TypeClose:
		if d.state == StateOpen || d.state == StateOpening {
			d.state = StateClosed
			d.isLocked = true
		}
	case event.TypeStopped:
		d.state = StateStopped
	case event.TypeReset:
		if d.state == StateStopped || d.state == StateFault {
			d.state = StateClosed
			d.isLocked = false
			d.obstacle = false
		}
	case event.TypeTrap:
		d.obstacle = true
		d.state = StateStopped
	case event.TypeAlarmClear:
		d.obstacle = false
	}
	return nil
}

func (d *Door) OnlineCheck(last time.Time, now time.Time) bool {
	ttl := DefaultHeartbeatTTL
	jitter := DefaultHeartbeatJitter
	if d.Config.HeartbeatTTL > 0 {
		ttl = d.Config.HeartbeatTTL
	}
	if d.Config.HeartbeatJitter > 0 {
		jitter = d.Config.HeartbeatJitter
	}
	return now.Sub(last) <= ttl+jitter
}

func CloseBatch(ctx context.Context, doors []*Door, act Actuator, sampler TrapSampler, rel GrantReleaser, obs Observer, rec Recorder, timeout time.Duration) []error {
	for _, d := range doors {
		if err := d.Close(ctx, act, sampler, rel, obs, rec, timeout); err != nil {
			return []error{err}
		}
	}
	return nil
}
