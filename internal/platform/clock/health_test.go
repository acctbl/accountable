package clock

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLinuxHealthRefusesUnsynchronizedOrUncertainClock(t *testing.T) {
	t.Parallel()

	for _, state := range []kernelTimeState{
		{state: kernelTimeError},
		{status: kernelStatusUnsync},
		{status: kernelStatusClockError},
		{maxErrorMicros: 1_000_001},
	} {
		health := linuxHealth{maximumError: time.Second, read: func() (kernelTimeState, error) { return state, nil }}
		if err := health.Check(context.Background()); !errors.Is(err, ErrClockUnsynchronized) {
			t.Fatalf("Check(%+v) = %v", state, err)
		}
	}
}

func TestLinuxHealthAcceptsSynchronizedClockInsideBound(t *testing.T) {
	t.Parallel()

	health := linuxHealth{maximumError: time.Second, read: func() (kernelTimeState, error) {
		return kernelTimeState{maxErrorMicros: 999_999}, nil
	}}
	if err := health.Check(context.Background()); err != nil {
		t.Fatalf("Check: %v", err)
	}
}
