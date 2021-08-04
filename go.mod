module github.com/identitatem/idp-mgmt-operator

go 1.16

require (
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.4.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/openshift/api v0.0.0-20210127195806-54e5e88cf848 //Openshift 4.6
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
	k8s.io/code-generator v0.20.4
	k8s.io/klog/v2 v2.10.0
	open-cluster-management.io/api v0.0.0-20210804091127-340467ff6239
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/controller-tools v0.5.0
)

replace (
	k8s.io/api => k8s.io/api v0.20.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.4
	k8s.io/client-go => k8s.io/client-go v0.20.4

)
