//go:build !linux

package clock

import (
	"errors"
	"time"
)

func NewLinuxHealth(maximumError time.Duration) Health {
	return linuxHealth{maximumError: maximumError, read: func() (kernelTimeState, error) {
		return kernelTimeState{}, errors.New("linux kernel clock status is unavailable")
	}}
}
