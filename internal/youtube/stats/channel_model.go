package stats

import "time"

type YoutubeChannel struct {
	YoutubeChannelId int64     `json:"youtubeChannelId" gorm:"primary_key"`
	ExternalId       string    `json:"externalId"`
	Name             string    `json:"name"`
	CreatedAt        time.Time `json:"createdAt"`
}
