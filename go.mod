module github.com/identitatem/idp-mgmt-operator

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	k8s.io/klog/v2 v2.10.0
	sigs.k8s.io/controller-runtime v0.5.0
)

replace github.com/go-logr/zapr => github.com/go-logr/zapr v0.2.0
