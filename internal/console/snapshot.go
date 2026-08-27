package console

import (
	"platformscreendoor/internal/record"
)

func (s *Service) DoorViews() []DoorView {
	doors := s.deps.Store.All()
	out := make([]DoorView, 0, len(doors))
	for _, d := range doors {
		snap := d.View()
		out = append(out, DoorView{
			ID:       snap.ID,
			Side:     snap.Side,
			Index:    snap.Index,
			State:    snap.State.String(),
			Locked:   snap.IsLocked,
			Local:    snap.Local,
			Online:   snap.Online,
			Obstacle: snap.Obstacle,
		})
	}
	return out
}

func (s *Service) DoorView(doorID string) (DoorView, bool) {
	d, ok := s.deps.Store.Get(doorID)
	if !ok {
		return DoorView{}, false
	}
	snap := d.View()
	return DoorView{
		ID:       snap.ID,
		Side:     snap.Side,
		Index:    snap.Index,
		State:    snap.State.String(),
		Locked:   snap.IsLocked,
		Local:    snap.Local,
		Online:   snap.Online,
		Obstacle: snap.Obstacle,
	}, true
}

func (s *Service) AlarmViews() []AlarmView {
	alarms := s.deps.Alarms.All()
	out := make([]AlarmView, 0, len(alarms))
	for _, a := range alarms {
		out = append(out, AlarmView{
			DoorID: a.DoorID,
			Kind:   a.Kind,
			Level:  a.Level.String(),
			Seq:    a.Seq,
			At:     a.At,
		})
	}
	return out
}

func (s *Service) AlarmHistory(doorID string) []AlarmView {
	alarms := s.deps.Alarms.History(doorID)
	out := make([]AlarmView, 0, len(alarms))
	for _, a := range alarms {
		out = append(out, AlarmView{
			DoorID: a.DoorID,
			Kind:   a.Kind,
			Level:  a.Level.String(),
			Seq:    a.Seq,
			At:     a.At,
		})
	}
	return out
}

func (s *Service) Events(after uint64, limit int) []EventView {
	records := s.deps.Records.Query(after, limit)
	out := make([]EventView, 0, len(records))
	for _, r := range records {
		out = append(out, EventView{Seq: r.Seq, DoorID: r.DoorID, Type: r.Type, Detail: r.Detail, At: r.At})
	}
	return out
}

func (s *Service) DoorEvents(doorID string, after uint64, limit int) []EventView {
	records := s.deps.Records.ByDoor(doorID, after, limit)
	out := make([]EventView, 0, len(records))
	for _, r := range records {
		out = append(out, EventView{Seq: r.Seq, DoorID: r.DoorID, Type: r.Type, Detail: r.Detail, At: r.At})
	}
	return out
}

func (s *Service) Resume(limit int) []EventView {
	records := record.Replay(s.deps.Records, s.cursor, limit)
	out := make([]EventView, 0, len(records))
	for _, r := range records {
		out = append(out, EventView{Seq: r.Seq, DoorID: r.DoorID, Type: r.Type, Detail: r.Detail, At: r.At})
	}
	return out
}

func (s *Service) Snapshot() Snapshot {
	return Snapshot{
		Doors:  s.DoorViews(),
		Alarms: s.AlarmViews(),
		Events: s.Events(0, 200),
		Latest: s.deps.Bus.Latest(),
	}
}

func (s *Service) Evaluate() []AlarmView {
	out := make([]AlarmView, 0)
	for _, d := range s.deps.Store.All() {
		snap := d.View()
		alarms := s.deps.AlarmHandler.Evaluate(snap, snap.Online)
		for _, a := range alarms {
			out = append(out, AlarmView{DoorID: a.DoorID, Kind: a.Kind, Level: a.Level.String(), Seq: a.Seq, At: a.At})
		}
	}
	return out
}
