package helpers

import "time"

func ParseTime(rfc3339Val string) time.Time {
	t, err := time.Parse(time.RFC3339, rfc3339Val)
	if err != nil {
		panic(err)
	}

	return t
}
