package train

import "platformscreendoor/internal/event"

type Controller struct {
	bus *event.Bus
}

func NewController(bus *event.Bus) *Controller {
	return &Controller{bus: bus}
}

func (c *Controller) Dock(t *Train, aligned bool) {
	c.ApplySignal(t, true, aligned)
}

func (c *Controller) Leave(t *Train) {
	c.ApplySignal(t, false, false)
}

func (c *Controller) SetAligned(t *Train, aligned bool) {
	t.Aligned = aligned
}

func (c *Controller) ApplySignal(t *Train, docked bool, aligned bool) {
	t.Docked = docked
	c.SetAligned(t, aligned)
	if c.bus != nil {
		kind := event.TypeTrainLeave
		if docked {
			kind = event.TypeTrainDock
		}
		c.bus.Publish(event.Event{DoorID: t.ID, Type: kind, Detail: t.LineID})
	}
}

type Planner struct {
	LineDoors map[string][]string
}

func NewPlanner() *Planner {
	return &Planner{LineDoors: make(map[string][]string)}
}

func (p *Planner) RegisterLine(lineID string, doorIDs []string) {
	p.LineDoors[lineID] = append([]string(nil), doorIDs...)
}

func (p *Planner) Plan(lineID string) map[int]string {
	ids := p.LineDoors[lineID]
	out := make(map[int]string, len(ids))
	for index, doorID := range ids {
		out[index] = doorID
	}
	return out
}

func (p *Planner) LineCount() int {
	return len(p.LineDoors)
}

func (p *Planner) HasLine(lineID string) bool {
	_, ok := p.LineDoors[lineID]
	return ok
}

func (p *Planner) Lines() []string {
	out := make([]string, 0, len(p.LineDoors))
	for lineID := range p.LineDoors {
		out = append(out, lineID)
	}
	return out
}
