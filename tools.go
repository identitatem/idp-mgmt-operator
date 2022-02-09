// Copyright Red Hat

// Used ot keep `go mod tidy -compat=1.17` from removing ginkgo modules from go.mod and go.sum
// Based on https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

// +build tools

// Place any runtime dependencies as imports in this file.
// Go modules will be forced to download and install them.
package tools

import (
  _ "github.com/onsi/ginkgo/v2/ginkgo/generators"
  _ "github.com/onsi/ginkgo/v2/ginkgo/internal"
  _ "github.com/onsi/ginkgo/v2/ginkgo/labels"
)
