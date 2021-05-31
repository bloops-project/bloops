// +build tools

// Package tools includes the list of tools used in the project.
package tools

// $ go generate -tags tools tools/tools.go
// make bootstrap
import (
	//go:generate go install golang.org/x/tools/cmd/goimports
	_ "golang.org/x/tools/cmd/goimports"

	//go:generate go install github.com/client9/misspell/cmd/misspell
	_ "github.com/client9/misspell/cmd/misspell"
)
