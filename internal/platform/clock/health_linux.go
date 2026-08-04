//go:build linux

package clock

import (
	"time"

	"golang.org/x/sys/unix"
)

func NewLinuxHealth(maximumError time.Duration) Health {
	return linuxHealth{maximumError: maximumError, read: readLinuxTimeState}
}

func readLinuxTimeState() (kernelTimeState, error) {
	var value unix.Timex
	state, err := unix.Adjtimex(&value)
	return kernelTimeState{state: state, status: value.Status, maxErrorMicros: value.Maxerror}, err
}
