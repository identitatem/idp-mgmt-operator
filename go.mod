module github.com/identitatem/idp-mgmt-operator

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	golang.org/x/net v0.0.0-20200707034311-ab3426394381 // indirect
	golang.org/x/text v0.3.3 // indirect
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	k8s.io/klog/v2 v2.10.0
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6 // indirect
	sigs.k8s.io/controller-runtime v0.5.0
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace github.com/go-logr/zapr => github.com/go-logr/zapr v0.2.0
