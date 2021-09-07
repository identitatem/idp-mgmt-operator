module github.com/identitatem/idp-mgmt-operator

go 1.16

require (
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/logr v0.4.0
	github.com/identitatem/dex-operator v0.0.4-0.20210903200056-66e1bc1c8aa1
	github.com/identitatem/idp-client-api v0.0.0-20210827164743-3225e7e96ed6
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.14.0
	github.com/openshift/api v3.9.0+incompatible
	k8s.io/api v0.22.0
	k8s.io/apiextensions-apiserver v0.22.0
	k8s.io/apimachinery v0.22.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog/v2 v2.10.0
	k8s.io/kubectl v0.21.0
	open-cluster-management.io/api v0.0.0-20210902053421-22c45bd6c7a1
	open-cluster-management.io/clusteradm v0.1.0-alpha.5
	sigs.k8s.io/controller-runtime v0.9.6
)

replace (
	// github.com/identitatem/dex-operator => /Users/dvernier/go/src/github.com/identitatem/dex-operator
	github.com/openshift/api => github.com/openshift/api v0.0.0-20210817132244-67c28690af52
	k8s.io/api => k8s.io/api v0.22.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.22.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.0
	k8s.io/client-go => k8s.io/client-go v0.22.0
	k8s.io/code-generator => k8s.io/code-generator v0.22.0
)
