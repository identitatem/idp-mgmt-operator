// Copyright Red Hat
package deploy

import (
	"embed"

	"open-cluster-management.io/clusteradm/pkg/helpers/asset"
)

//go:embed dex-operator
var files embed.FS

func GetScenarioResourcesReader() *asset.ScenarioResourcesReader {
	return asset.NewScenarioResourcesReader(&files)
}
