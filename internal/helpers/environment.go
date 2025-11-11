package helpers

import "os"

func GetEnv(key string) string {
	return GetEnvWithFallback(key, "")
}

func GetEnvWithFallback(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}