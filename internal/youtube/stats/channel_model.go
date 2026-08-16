package stats

import "time"

type YoutubeChannel struct {
	YoutubeChannelId int64      `json:"youtubeChannelId" gorm:"primary_key"`
	ExternalID       string     `json:"externalId"`
	Title            string     `json:"title"`
	PublishedAt      *time.Time `json:"publishedAt"`
	CreatedAt        time.Time  `json:"createdAt"`
	CheckedAt        *time.Time `json:"checkedAt"`
	Description      *string    `json:"description"`
	CustomURL        *string    `json:"customUrl"`
}

type YoutubeChannelStats struct {
	YoutubeChannelStatID int64     `json:"youtubeChannelStatId"`
	YoutubeChannelID     int64     `json:"youtubeChannelId"`
	SubscribersCount     int64     `json:"subscribersCount"`
	ViewCount            int64     `json:"viewCount"`
	VideoCount           int64     `json:"videoCount"`
	CreatedAt            time.Time `json:"createdAt"`
}
