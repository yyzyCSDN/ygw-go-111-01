package door

type Fanout struct {
	observers []Observer
}

func NewFanout(observers ...Observer) *Fanout {
	return &Fanout{observers: observers}
}

func (f *Fanout) Add(obs Observer) {
	f.observers = append(f.observers, obs)
}

func (f *Fanout) OnState(doorID string, state State) {
	for _, obs := range f.observers {
		if obs != nil {
			obs.OnState(doorID, state)
		}
	}
}

func (f *Fanout) OnTrap(doorID string) {
	for _, obs := range f.observers {
		if obs != nil {
			obs.OnTrap(doorID)
		}
	}
}

func (f *Fanout) OnAlarm(doorID string, kind string) {
	for _, obs := range f.observers {
		if obs != nil {
			obs.OnAlarm(doorID, kind)
		}
	}
}

func (f *Fanout) OnAlarmClear(doorID string) {
	for _, obs := range f.observers {
		if obs != nil {
			obs.OnAlarmClear(doorID)
		}
	}
}
