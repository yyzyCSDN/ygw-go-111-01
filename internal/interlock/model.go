package interlock

import "time"

type Result struct {
	Allowed bool
	Reason  string
	Budget  time.Duration
}
