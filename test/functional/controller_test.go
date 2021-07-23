// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

// +build functional

package functional

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/klog/v2"
)

func init() {
	klog.SetOutput(GinkgoWriter)
	klog.InitFlags(nil)

}

var _ = Describe("AuthRealm", func() {
	BeforeEach(func() {
		SetDefaultEventuallyTimeout(20 * time.Second)
		SetDefaultEventuallyPollingInterval(1 * time.Second)
	})

	AfterEach(func() {
	})

})
