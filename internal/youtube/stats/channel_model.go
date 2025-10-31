package stats

import "time"

type YoutubeChannel struct {
	YoutubeChannelId int64      `json:"youtubeChannelId" gorm:"primary_key"`
	ExternalID       string     `json:"externalId"`
	Name             string     `json:"name"`
	CreatedAt        time.Time  `json:"createdAt"`
	CheckedAt        *time.Time `json:"checkedAt"`
}

type YoutubeChannelStats struct {
	YoutubeChannelStatID int64     `json:"youtubeChannelStatId"`
	YoutubeChannelID     int64     `json:"youtubeChannelId"`
	SubscribersCount     int64     `json:"subscribersCount"`
	CreatedAt            time.Time `json:"createdAt"`
}
