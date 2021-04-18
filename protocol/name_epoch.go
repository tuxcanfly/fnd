package protocol

import (
	"time"

	"fnd.localhost/handshake/primitives"
)

func NameEpoch(name string) uint16 {
	hash := primitives.HashName(name)
	mod := modBuffer(hash, hoursPerWeek)
	offset := mod * secondsPerHour
	startDate := epochDate.Add(time.Duration(offset) * time.Second)
	return uint16(int(now().Sub(startDate).Seconds()) / int(monthDuration.Seconds()))
}
