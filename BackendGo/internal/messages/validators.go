package messages

import "time"

const MaxTTL = 30 * 24 * time.Hour

func ClampTTL(d time.Duration) time.Duration {
	if d <= 0 { return 0 }
	if d > MaxTTL { return MaxTTL }
	return d
}