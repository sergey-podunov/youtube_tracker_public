package ytclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseChannelURL(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedType   URLLookupType
		expectedValue  string
		expectedErrMsg string // non-empty = expect error containing this substring
	}{
		{
			name:          "handle URL",
			url:           "https://youtube.com/@BBCNews",
			expectedType:  LookupByHandle,
			expectedValue: "@BBCNews",
		},
		{
			name:          "handle URL with www",
			url:           "https://www.youtube.com/@BBCNews",
			expectedType:  LookupByHandle,
			expectedValue: "@BBCNews",
		},
		{
			name:          "channel ID URL",
			url:           "https://youtube.com/channel/UC16niRr50-MSBwiO3YDb3RA",
			expectedType:  LookupByID,
			expectedValue: "UC16niRr50-MSBwiO3YDb3RA",
		},
		{
			name:          "legacy user URL",
			url:           "https://youtube.com/user/BBCNews",
			expectedType:  LookupByUsername,
			expectedValue: "BBCNews",
		},
		{
			name:           "/c/ URL returns error",
			url:            "https://youtube.com/c/BBCNews",
			expectedErrMsg: "/c/ URL format",
		},
		{
			name:           "wrong host returns error",
			url:            "https://example.com/@BBCNews",
			expectedErrMsg: "must be youtube.com",
		},
		{
			name:           "empty handle returns error",
			url:            "https://youtube.com/@",
			expectedErrMsg: "empty handle",
		},
		{
			name:           "unrecognised path returns error",
			url:            "https://youtube.com/watch?v=abc",
			expectedErrMsg: "unrecognised",
		},
		{
			name:           "missing path returns error",
			url:            "https://youtube.com/",
			expectedErrMsg: "missing channel path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseChannelURL(tt.url)
			if tt.expectedErrMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				var urlErr ErrInvalidChannelURL
				assert.ErrorAs(t, err, &urlErr, "error should be ErrInvalidChannelURL")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedType, got.Type)
			assert.Equal(t, tt.expectedValue, got.Value)
		})
	}
}
