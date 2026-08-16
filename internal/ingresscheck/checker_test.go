package ingresscheck

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUncoveredPaths(t *testing.T) {
	type testCase struct {
		name       string
		apiPaths   []string
		rules      []Rule
		exceptions []string
		want       []string
	}

	tests := []testCase{
		{
			name:     "exact rule covers exact path",
			apiPaths: []string{"/status"},
			rules:    []Rule{{Path: "/status", PathType: "Exact"}},
			want:     nil,
		},
		{
			name:     "exact rule does not cover templated subpath",
			apiPaths: []string{"/youtube/channels", "/youtube/channels/{id}"},
			rules:    []Rule{{Path: "/youtube/channels", PathType: "Exact"}},
			want:     []string{"/youtube/channels/{id}"},
		},
		{
			name: "regex rule covers templated paths",
			apiPaths: []string{
				"/youtube/video/{id}",
				"/youtube/video/{id}/statistics",
			},
			rules: []Rule{{Path: "/youtube/video(/.*)?", PathType: "ImplementationSpecific"}},
			want:  nil,
		},
		{
			name:     "regex rule is start-anchored",
			apiPaths: []string{"/api/youtube/channel"},
			rules:    []Rule{{Path: "/youtube/channel(/.*)?", PathType: "ImplementationSpecific"}},
			want:     []string{"/api/youtube/channel"},
		},
		{
			name:     "prefix rule covers itself and subpaths",
			apiPaths: []string{"/brand", "/brand/{id}/details"},
			rules:    []Rule{{Path: "/brand", PathType: "Prefix"}},
			want:     nil,
		},
		{
			name:     "prefix rule matches on segment boundary only",
			apiPaths: []string{"/brandnew"},
			rules:    []Rule{{Path: "/brand", PathType: "Prefix"}},
			want:     []string{"/brandnew"},
		},
		{
			name:     "uncovered path is reported",
			apiPaths: []string{"/status", "/brandnew/endpoint"},
			rules:    []Rule{{Path: "/status", PathType: "Exact"}},
			want:     []string{"/brandnew/endpoint"},
		},
		{
			name: "exceptions skip the prefix itself and nested paths",
			apiPaths: []string{
				"/schedule",
				"/schedule/video/job/{id}",
				"/status",
			},
			rules:      []Rule{{Path: "/status", PathType: "Exact"}},
			exceptions: []string{"/schedule"},
			want:       nil,
		},
		{
			name:       "exception matches on segment boundary only",
			apiPaths:   []string{"/schedulefoo"},
			rules:      []Rule{{Path: "/status", PathType: "Exact"}},
			exceptions: []string{"/schedule"},
			want:       []string{"/schedulefoo"},
		},
		{
			name:     "no rules leaves every non-excepted path uncovered",
			apiPaths: []string{"/status", "/youtube/channels"},
			rules:    nil,
			want:     []string{"/status", "/youtube/channels"},
		},
		{
			name: "result is sorted",
			apiPaths: []string{
				"/zebra",
				"/alpha",
				"/middle",
			},
			rules: nil,
			want:  []string{"/alpha", "/middle", "/zebra"},
		},
		{
			name:     "no api paths yields no uncovered",
			apiPaths: nil,
			rules:    []Rule{{Path: "/status", PathType: "Exact"}},
			want:     nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := UncoveredPaths(tc.apiPaths, tc.rules, tc.exceptions)

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestUncoveredPaths_InvalidRegex(t *testing.T) {
	_, err := UncoveredPaths(
		[]string{"/status"},
		[]Rule{{Path: "/status(", PathType: "ImplementationSpecific"}},
		nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "/status(")
}
