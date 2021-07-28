// Copyright Contributors to the Open Cluster Management project

// +build functional

package functional

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = BeforeSuite(func() {
	By("Setup Hub client")
})

func TestRcmController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "idp-mgmt-operator Suite")
}
