package stats

import "time"

type YoutubeChannel struct {
	YoutubeChannelId int64     `json:"youtubeChannelId" gorm:"primary_key"`
	ExternalId       string    `json:"externalId"`
	Name             string    `json:"name"`
	CreatedAt        time.Time `json:"createdAt"`
}

type YoutubeChannelStats struct {
	YoutubeChannelStatId int64     `json:"youtube_channel_stat_id"`
	YoutubeChannelId     int64     `json:"youtubeChannelId"`
	SubscribersCount     int64     `json:"subscribersCount"`
	CreatedAt            time.Time `json:"createdAt"`
}
