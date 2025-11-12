package helpers

import (
	"strconv"
	"time"
)

func ToInt32(value string, fallback int32) int32 {
	if len(value) == 0 {
		return fallback
	}

	if intValue, err := strconv.ParseInt(value, 10, 32); err == nil {
		return int32(intValue)
	}

	return fallback
}

func ToInt64(value string, fallback int64) int64 {
	if len(value) == 0 {
		return fallback
	}

	if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
		return intValue
	}

	return fallback
}

func ToDuration(value string, fallback time.Duration) time.Duration {
	if len(value) == 0 {
		return fallback
	}

	if duration, err := time.ParseDuration(value); err == nil {
		return duration
	}

	return fallback
}
