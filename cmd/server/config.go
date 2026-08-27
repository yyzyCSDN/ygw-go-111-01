package main

import (
	"flag"
	"time"

	"platformscreendoor/internal/door"
)

type config struct {
	addr            string
	dataDir         string
	doors           int
	openTimeout     time.Duration
	closeTimeout    time.Duration
	trapInterval    time.Duration
	heartbeatTTL    time.Duration
	heartbeatJitter time.Duration
}

func parseConfig() config {
	cfg := config{
		addr:            "127.0.0.1:8090",
		dataDir:         "data",
		doors:           12,
		openTimeout:     door.DefaultOpenTimeout,
		closeTimeout:    door.DefaultCloseTimeout,
		trapInterval:    door.DefaultTrapInterval,
		heartbeatTTL:    door.DefaultHeartbeatTTL,
		heartbeatJitter: door.DefaultHeartbeatJitter,
	}
	flag.StringVar(&cfg.addr, "addr", cfg.addr, "listen address")
	flag.StringVar(&cfg.dataDir, "dir", cfg.dataDir, "data directory")
	flag.IntVar(&cfg.doors, "doors", cfg.doors, "number of platform doors")
	flag.DurationVar(&cfg.openTimeout, "open-timeout", cfg.openTimeout, "door open confirmation timeout")
	flag.DurationVar(&cfg.closeTimeout, "close-timeout", cfg.closeTimeout, "door close confirmation timeout")
	flag.DurationVar(&cfg.trapInterval, "trap-interval", cfg.trapInterval, "anti trap sampling interval")
	flag.DurationVar(&cfg.heartbeatTTL, "heartbeat-ttl", cfg.heartbeatTTL, "heartbeat offline threshold")
	flag.DurationVar(&cfg.heartbeatJitter, "heartbeat-jitter", cfg.heartbeatJitter, "heartbeat grace jitter")
	flag.Parse()
	return cfg
}
