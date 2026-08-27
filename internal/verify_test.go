package internal_test

import (
	"testing"
	"time"

	"platformscreendoor/internal/alarm"
	"platformscreendoor/internal/door"
	"platformscreendoor/internal/event"
	"platformscreendoor/internal/record"
)

func TestBug10_Heartbeatmisjudge(t *testing.T) {
	bus := event.New()
	records := record.NewStore(50)
	rec := record.NewRecorder(nil, t.TempDir(), records)
	dispatch := alarm.NewDispatcher(bus)
	alarms := alarm.NewManager(rec, dispatch, alarm.DefaultPolicy())
	ttl := 1 * time.Second
	jitter := 500 * time.Millisecond
	now := time.Now()
	alarms.Heartbeat("PSD10", now)
	late := now.Add(1200 * time.Millisecond)
	if alarms.Offline("PSD10", late, ttl, jitter) {
		t.Fatal("200ms-late heartbeat misjudged as offline")
	}
	d := door.New("PSD10", 1, 9, door.Config{HeartbeatTTL: ttl, HeartbeatJitter: jitter})
	if !d.OnlineCheck(now, late) {
		t.Fatal("door online check misjudged late heartbeat as offline")
	}
}
