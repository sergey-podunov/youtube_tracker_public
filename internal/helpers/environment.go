package helpers

import (
	"os"
)

func GetBuildTag() string {
	return GetEnvWithFallback("BUILD_TAG", "undefined")
}

func GetEnv(key string) string {
	return GetEnvWithFallback(key, "")
}

func GetEnvWithFallback(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}