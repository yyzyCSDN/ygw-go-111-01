package record

import "platformscreendoor/internal/event"

func Replay(store *Store, cursor *event.Topic, limit int) []Record {
	records := store.Query(cursor.Last, limit)
	for _, r := range records {
		cursor.Advance(r.Seq)
	}
	return records
}

func Behind(store *Store, cursor *event.Topic) int {
	return store.Size() - int(cursor.Last)
}
