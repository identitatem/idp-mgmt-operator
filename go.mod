module github.com/identitatem/idp-mgmt-operator

go 1.16

require (
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/logr v0.4.0
	github.com/identitatem/dex-operator v0.0.4-0.20210923130926-5be7f97451cb
	github.com/identitatem/idp-client-api v0.0.0-20210920132446-528523b992c0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.14.0
	github.com/open-cluster-management/multicloud-operators-foundation v1.0.0-2021-09-22-22-06-10.0.20210923102123-b296cd01e3f7
	github.com/openshift/api v3.9.1-0.20191111211345-a27ff30ebf09+incompatible
	github.com/openshift/generic-admission-server v1.14.1-0.20210422140326-da96454c926d
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.22.0
	k8s.io/apiextensions-apiserver v0.22.0
	k8s.io/apimachinery v0.22.0
	k8s.io/apiserver v0.22.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/component-base v0.22.0
	k8s.io/klog/v2 v2.10.0
	k8s.io/kubectl v0.21.2
	open-cluster-management.io/api v0.0.0-20210916013819-2e58cdb938f9
	open-cluster-management.io/clusteradm v0.1.0-alpha.5
	sigs.k8s.io/controller-runtime v0.9.6
)

replace (
	github.com/openshift/api => github.com/openshift/api v0.0.0-20210817132244-67c28690af52
	k8s.io/api => k8s.io/api v0.22.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.22.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.0
	k8s.io/apiserver => k8s.io/apiserver v0.22.0
	k8s.io/client-go => k8s.io/client-go v0.22.0
	k8s.io/code-generator => k8s.io/code-generator v0.22.0
	k8s.io/kubectl => k8s.io/kubectl v0.21.0
)

// Determined by go.mod in github.com/openshift/hive
// from installer
replace (
	github.com/Azure/go-autorest/autorest/adal => github.com/Azure/go-autorest/autorest/adal v0.9.15
	github.com/Azure/go-autorest/autorest/azure/cli => github.com/Azure/go-autorest/autorest/azure/cli v0.3.1
	github.com/Azure/go-autorest/autorest/date => github.com/Azure/go-autorest/autorest/date v0.2.0
	github.com/Azure/go-autorest/autorest/validation => github.com/Azure/go-autorest/autorest/validation v0.2.1-0.20191028180845-3492b2aff503
	github.com/apparentlymart/go-cidr => github.com/apparentlymart/go-cidr v1.0.1
	github.com/go-openapi/spec => github.com/go-openapi/spec v0.19.0
	github.com/hashicorp/go-plugin => github.com/hashicorp/go-plugin v1.2.2
	github.com/kubevirt/terraform-provider-kubevirt => github.com/nirarg/terraform-provider-kubevirt v0.0.0-20201222125919-101cee051ed3
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20200715132148-0f91f62a41fe
	github.com/metal3-io/cluster-api-provider-baremetal => github.com/openshift/cluster-api-provider-baremetal v0.0.0-20190821174549-a2a477909c1d
	github.com/openshift/hive/apis => github.com/openshift/hive/apis v0.0.0-20210812165704-3dc3c3c7ca4b
	github.com/terraform-providers/terraform-provider-aws => github.com/openshift/terraform-provider-aws v1.60.1-0.20200630224953-76d1fb4e5699
	github.com/terraform-providers/terraform-provider-azurerm => github.com/openshift/terraform-provider-azurerm v1.40.1-0.20200707062554-97ea089cc12a
	github.com/terraform-providers/terraform-provider-ignition/v2 => github.com/community-terraform-providers/terraform-provider-ignition/v2 v2.1.0
	github.com/ulikunitz/xz => github.com/ulikunitz/xz v0.5.7
	google.golang.org/api => google.golang.org/api v0.25.0
	google.golang.org/genproto => google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
	kubevirt.io/client-go => kubevirt.io/client-go v0.29.0
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20201022175424-d30c7a274820
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20201016155852-4090a6970205
	sigs.k8s.io/cluster-api-provider-openstack => github.com/openshift/cluster-api-provider-openstack v0.0.0-20201116051540-155384b859c5
)
