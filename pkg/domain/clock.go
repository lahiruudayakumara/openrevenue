package domain

import "time"

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }

type FixedClock struct{ now time.Time }

func NewFixedClock(now time.Time) (FixedClock, error) {
	if now.IsZero() {
		return FixedClock{}, invalid("clock time", "must not be zero")
	}
	return FixedClock{now: now.UTC()}, nil
}

func (c FixedClock) Now() time.Time { return c.now }
