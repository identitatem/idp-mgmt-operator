module github.com/identitatem/idp-mgmt-operator

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	gopkg.in/yaml.v1 v1.0.0-20140924161607-9f9df34309c0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/api v0.20.4 // indirect
	k8s.io/apimachinery v0.20.4
	k8s.io/client-go v0.20.4
	k8s.io/code-generator v0.20.4
	k8s.io/klog/v2 v2.10.0
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/controller-tools v0.5.0
)
