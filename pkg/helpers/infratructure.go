// Copyright Red Hat

package helpers

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	ocinfrav1 "github.com/openshift/api/config/v1"
	giterrors "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	infrastructureConfigName = "cluster"
	apiserverConfigName      = "cluster"
	openshiftConfigNamespace = "openshift-config"
)

func infrastructureConfigNameNsN() types.NamespacedName {
	return types.NamespacedName{
		Name: infrastructureConfigName,
	}
}

func GetKubeAPIServerAddress(client client.Client) (string, error) {
	infraConfig := &ocinfrav1.Infrastructure{}

	if err := client.Get(context.TODO(), infrastructureConfigNameNsN(), infraConfig); err != nil {
		return "", giterrors.WithStack(err)
	}

	return infraConfig.Status.APIServerURL, nil
}

func GetAppsURL(c client.Client, withPort bool) (string, string, error) {
	apiServerURL, err := GetKubeAPIServerAddress(c)
	if err != nil {
		return "", "", err
	}
	u, err := url.Parse(apiServerURL)
	if err != nil {
		return "", "", giterrors.WithStack(err)
	}

	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		return u.Scheme, "", giterrors.WithStack(err)
	}

	host = strings.Replace(host, "api", "apps", 1)
	if withPort && len(port) != 0 {
		host = fmt.Sprintf("%s:%s", host, port)
	}
	return u.Scheme, host, nil
}
