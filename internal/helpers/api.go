package helpers

import "youtube_tracker/internal/api"

func OptString(s *string) api.OptString {
	if s != nil {
		return api.NewOptString(*s)
	}
	return api.OptString{}
}

func OptInt64(i *int64) api.OptInt64 {
	if i != nil {
		return api.NewOptInt64(*i)
	}
	return api.OptInt64{}
}
