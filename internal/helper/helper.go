package helper

import (
	"math"
	"time"
)

func DurationToUint32(value time.Duration) uint32 {
	val := int64(value / time.Second)
	var valueInt32 uint32
	switch {
	case val < 0:
		valueInt32 = 0
	case val > math.MaxUint32:
		valueInt32 = math.MaxUint32
	default:
		valueInt32 = uint32(val)
	}
	return valueInt32
}

func IntToUint32(timeout int) uint32 {
	var timeoutInt32 uint32
	switch {
	case timeout < 0:
		timeoutInt32 = 0
	case timeout > math.MaxUint32:
		timeoutInt32 = math.MaxUint32
	default:
		timeoutInt32 = uint32(timeout)
	}
	return timeoutInt32
}
