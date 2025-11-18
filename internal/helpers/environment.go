package helpers

import (
	"fmt"
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

func GetRequiredEnv(key string) (string, error) {
	if value, ok := os.LookupEnv(key); ok {
		return value, nil
	}
	
	return "", fmt.Errorf("%s environment variable is required", key)
}