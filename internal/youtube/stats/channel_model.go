package stats

import "time"

type YoutubeChannel struct {
	youtubeChannelId int       `json:"youtubeChannelId" gorm:"primary_key"`
	externalId       string    `json:"externalId"`
	name             string    `json:"name"`
	createdAt        time.Time `json:"createdAt"`
}
