package clock

import (
	"time"
	_ "time/tzdata"
)

type Clock interface {
	Now() time.Time
}

type System struct{}

func (System) Now() time.Time {
	return toContract(time.Now())
}

type Fixed struct {
	Instant time.Time
}

func (f Fixed) Now() time.Time {
	return toContract(f.Instant)
}

func toContract(t time.Time) time.Time {
	return t.UTC().Truncate(time.Microsecond)
}
