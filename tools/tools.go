//go:build tools
// +build tools

package tools

import (
	_ "github.com/onsi/ginkgo/v2" // comment to make golint happy
)

// This file imports packages that are used when running go generate, or used
// during the development process but not otherwise depended on by built code.
