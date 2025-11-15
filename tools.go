//go:build tools

package example

// Import to keep it in go.mod.
import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/ogen-go/ogen/cmd/ogen"
)
