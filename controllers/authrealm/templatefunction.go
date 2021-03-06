// Copyright Red Hat

package authrealm

import (
	"text/template"

	"github.com/ghodss/yaml"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
	clusteradmasset "open-cluster-management.io/clusteradm/pkg/helpers/asset"
)

//ApplierFuncMap adds the function map
func FuncMap() template.FuncMap {
	return template.FuncMap(GenericFuncMap())
}

// GenericFuncMap returns a copy of the basic function map as a map[string]interface{}.
func GenericFuncMap() map[string]interface{} {
	gfm := make(map[string]interface{}, len(genericMap))
	for k, v := range genericMap {
		gfm[k] = v
	}
	return gfm
}

var genericMap = map[string]interface{}{
	"replaceObjectNameNamespace": replaceObjectNameNamespace,
}

func replaceObjectNameNamespace(reader *clusteradmasset.ScenarioResourcesReader, file string, newName, newNamespace string) (string, error) {
	b, err := reader.Asset(file)
	if err != nil {
		return "", err
	}
	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(b, obj); err != nil {
		return "", err
	}
	metadata := obj.Object["metadata"].(map[string]interface{})
	metadata["name"] = newName
	if len(newNamespace) != 0 {
		metadata["namespace"] = newNamespace
	}
	m, err := yaml.Marshal(obj)
	if err != nil {
		klog.Error(err)
		return "", err
	}
	return string(m), nil
}
