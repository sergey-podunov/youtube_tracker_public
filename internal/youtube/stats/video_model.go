package stats

import "time"

type YoutubeVideo struct {
	YoutubeVideoId   int64      `json:"youtubeVideoId" gorm:"primary_key"`
	YoutubeChannelId int64      `json:"youtubeChannelId"`
	ExternalID       string     `json:"externalId"`
	Title            string     `json:"title"`
	PublishedAt      *time.Time `json:"publishedAt"`
	CreatedAt        time.Time  `json:"createdAt"`
	CheckedAt        *time.Time `json:"checkedAt"`
}

type YoutubeVideoStats struct {
	YoutubeVideoStatID int64     `json:"youtubeVideoStatId"`
	YoutubeVideoID     int64     `json:"youtubeVideoId"`
	ViewCount          int64     `json:"viewCount"`
	LikeCount          int64     `json:"likeCount"`
	CommentCount       int64     `json:"commentCount"`
	CreatedAt          time.Time `json:"createdAt"`
}
