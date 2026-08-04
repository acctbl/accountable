package clock

import (
	"context"
	"errors"
	"time"
)

const (
	kernelTimeError        = 5
	kernelStatusUnsync     = 0x40
	kernelStatusClockError = 0x1000
)

var ErrClockUnsynchronized = errors.New("system clock is not synchronized")

type Health interface {
	Check(context.Context) error
}

type SystemHealth struct{}

func (SystemHealth) Check(ctx context.Context) error { return ctx.Err() }

type kernelTimeState struct {
	state          int
	status         int32
	maxErrorMicros int64
}

type linuxHealth struct {
	maximumError time.Duration
	read         func() (kernelTimeState, error)
}

func (h linuxHealth) Check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	state, err := h.read()
	if err != nil || state.state == kernelTimeError ||
		state.status&(kernelStatusUnsync|kernelStatusClockError) != 0 ||
		state.maxErrorMicros < 0 || state.maxErrorMicros > h.maximumError.Microseconds() {
		return ErrClockUnsynchronized
	}
	return nil
}
