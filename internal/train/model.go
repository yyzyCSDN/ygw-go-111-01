package train

type Train struct {
	ID      string
	LineID  string
	Docked  bool
	Aligned bool
	DoorMap map[int]string
}

func New(id string, lineID string) *Train {
	return &Train{
		ID:      id,
		LineID:  lineID,
		DoorMap: make(map[int]string),
	}
}

func (t *Train) Snapshot() Train {
	copyMap := make(map[int]string, len(t.DoorMap))
	for index, doorID := range t.DoorMap {
		copyMap[index] = doorID
	}
	return Train{
		ID:      t.ID,
		LineID:  t.LineID,
		Docked:  t.Docked,
		Aligned: t.Aligned,
		DoorMap: copyMap,
	}
}

func (t *Train) DoorFor(index int) (string, bool) {
	doorID, ok := t.DoorMap[index]
	return doorID, ok
}

func (t *Train) MapDoor(index int, doorID string) {
	if t.DoorMap == nil {
		t.DoorMap = make(map[int]string)
	}
	t.DoorMap[index] = doorID
}
