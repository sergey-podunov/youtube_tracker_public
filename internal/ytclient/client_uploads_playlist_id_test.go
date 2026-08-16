package ytclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadsPlaylistID(t *testing.T) {
	tests := []struct {
		name      string
		channelID string
		expected  string
		expectErr bool
	}{
		{
			name:      "UC channel ID",
			channelID: "UCabc123",
			expected:  "UUabc123",
		},
		{
			name:      "real-looking channel ID",
			channelID: "UC16niRr50-MSBwiO3YDb3RA",
			expected:  "UU16niRr50-MSBwiO3YDb3RA",
		},
		{
			name:      "non-UC prefix returns error",
			channelID: "HCxyz",
			expectErr: true,
		},
		{
			name:      "empty string returns error",
			channelID: "",
			expectErr: true,
		},
		{
			name:      "bare UC prefix returns error",
			channelID: "UC",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UploadsPlaylistID(tt.channelID)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
