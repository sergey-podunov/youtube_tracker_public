package ytclient

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

type Client interface {
	GetChannelData(ctx context.Context, channelId string) (ChannelData, error)
	GetChannelByHandle(ctx context.Context, handle string) (ChannelData, error)
	GetChannelByUsername(ctx context.Context, username string) (ChannelData, error)
	GetChannelVideos(ctx context.Context, channelId string, maxResults int64) ([]VideoData, error)
	GetVideosData(ctx context.Context, videoIds []string) ([]VideoData, error)
}

type ChannelData struct {
	ChannelID        string `json:"channelId"`
	Title            string `json:"title"`
	CustomURL        string `json:"customUrl"`
	Description      string `json:"description"`
	SubscribersCount int64  `json:"subscribersCount"`
	ViewCount        int64  `json:"viewCount"`
	VideoCount       int64  `json:"videoCount"`
	PublishedAt      string `json:"publishedAt"`
	ThumbnailURL     string `json:"thumbnailUrl"`
}

type VideoData struct {
	VideoID      string `json:"videoId"`
	ChannelID    string `json:"channelId"`
	Title        string `json:"title"`
	PublishedAt  string `json:"publishedAt"`
	ViewCount    int64  `json:"viewCount"`
	LikeCount    int64  `json:"likeCount"`
	CommentCount int64  `json:"commentCount"`
}

// UploadsPlaylistID derives a channel's uploads playlist ID from its channel ID
// (the "UC" prefix replaced with "UU"). Returns an error for IDs that do not
// start with "UC".
func UploadsPlaylistID(channelID string) (string, error) {
	if !strings.HasPrefix(channelID, "UC") || len(channelID) == len("UC") {
		return "", fmt.Errorf("invalid channel ID %q: expected \"UC...\" format", channelID)
	}
	return "UU" + channelID[len("UC"):], nil
}

// ErrInvalidChannelURL is returned by ParseChannelURL for bad or unsupported URLs.
// Callers can check for this type with errors.As to return a 400 response.
type ErrInvalidChannelURL struct {
	Msg string
}

func (e ErrInvalidChannelURL) Error() string { return e.Msg }

// URLLookupType describes which YouTube channels.list parameter to use.
type URLLookupType int

const (
	LookupByID       URLLookupType = iota // channels.list?id=UCxxxxxx
	LookupByHandle                        // channels.list?forHandle=@name
	LookupByUsername                      // channels.list?forUsername=name (legacy /user/ URLs)
)

// ParsedChannelURL is the result of parsing a YouTube channel URL.
type ParsedChannelURL struct {
	Type  URLLookupType
	Value string
}

// ParseChannelURL parses a YouTube channel URL into a typed lookup descriptor.
//
// Supported formats:
//   - https://youtube.com/@handle       → LookupByHandle, "@handle"
//   - https://youtube.com/channel/UCxxx → LookupByID, "UCxxxxxx"
//   - https://youtube.com/user/name     → LookupByUsername, "name"
//
// Returns ErrInvalidChannelURL for /c/ paths (no YouTube API support) or unrecognised formats.
func ParseChannelURL(rawURL string) (ParsedChannelURL, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ParsedChannelURL{}, ErrInvalidChannelURL{fmt.Sprintf("invalid URL: %v", err)}
	}

	host := strings.TrimPrefix(parsedURL.Host, "www.")
	if host != "youtube.com" {
		return ParsedChannelURL{}, ErrInvalidChannelURL{
			fmt.Sprintf("unsupported host %q: must be youtube.com", parsedURL.Host),
		}
	}

	path := strings.TrimPrefix(parsedURL.Path, "/")
	segments := strings.SplitN(path, "/", 3)
	if len(segments) == 0 || segments[0] == "" {
		return ParsedChannelURL{}, ErrInvalidChannelURL{"missing channel path in URL"}
	}

	switch {
	case strings.HasPrefix(segments[0], "@"):
		if segments[0] == "@" {
			return ParsedChannelURL{}, ErrInvalidChannelURL{"empty handle in URL"}
		}
		return ParsedChannelURL{Type: LookupByHandle, Value: segments[0]}, nil

	case segments[0] == "channel" && len(segments) >= 2 && segments[1] != "":
		return ParsedChannelURL{Type: LookupByID, Value: segments[1]}, nil

	case segments[0] == "user" && len(segments) >= 2 && segments[1] != "":
		return ParsedChannelURL{Type: LookupByUsername, Value: segments[1]}, nil

	case segments[0] == "c":
		return ParsedChannelURL{}, ErrInvalidChannelURL{
			"the /c/ URL format cannot be resolved via the YouTube API; use the @handle URL instead",
		}

	default:
		return ParsedChannelURL{}, ErrInvalidChannelURL{
			fmt.Sprintf("unrecognised YouTube channel URL format: %s", rawURL),
		}
	}
}
