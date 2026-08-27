package event

type Topic struct {
	Name string
	Last uint64
}

func NewTopic(name string) *Topic {
	return &Topic{Name: name}
}

func (t *Topic) Advance(seq uint64) uint64 {
	if seq > t.Last {
		t.Last = seq
	}
	return t.Last
}

func (t *Topic) Behind(latest uint64) uint64 {
	if latest > t.Last {
		return latest - t.Last
	}
	return 0
}
