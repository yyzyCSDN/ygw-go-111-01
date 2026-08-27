package alarm

import "platformscreendoor/internal/event"

type Dispatcher struct {
	bus *event.Bus
}

func NewDispatcher(bus *event.Bus) *Dispatcher {
	return &Dispatcher{bus: bus}
}

func (d *Dispatcher) Dispatch(a Alarm) {
	if d.bus == nil {
		return
	}
	d.bus.Publish(event.Event{DoorID: a.DoorID, Type: event.TypeAlarm, Detail: a.Kind})
}
