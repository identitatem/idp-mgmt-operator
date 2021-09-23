#!/bin/bash
# Copyright Red Hat

export NAME=${NAME:-"authrealm-sample"}
export NS=${NS:-"authrealm-sample-ns"}
export APPS=$(oc get infrastructure cluster -ojsonpath='{.status.apiServerURL}' | cut -d':' -f2 | sed 's/\/\/api/apps/g')
export IDP_NAME=${IDP_NAME:-"github-sample-idp"}
export GITHUB_APP_CLIENT_ID=${GITHUB_APP_CLIENT_ID:-"githubappclientid"}
export GITHUB_APP_CLIENT_SECRET=${GITHUB_APP_CLIENT_SECRET:-"githubappclientsecret"}
export GITHUB_APP_CLIENT_ORG=${GITHUB_APP_CLIENT_ORG:-"githubappclientorg"}
export ROUTE_SUBDOMAIN=${ROUTE_SUBDOMAIN:-"testdomain"}

export THE_FILENAME=/tmp/${NAME}".yaml"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
BASE64="base64 -w 0"
if [ "${OS}" == "darwin" ]; then
    BASE64="base64"
fi

GITHUB_APP_CLIENT_SECRET_B64=`echo $GITHUB_APP_CLIENT_SECRET | $BASE64`


cat > ${THE_FILENAME} <<EOF
---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    control-plane: controller-manager
  name: ${NS}
---
apiVersion: cluster.open-cluster-management.io/v1alpha1
kind: ManagedClusterSet
metadata:
  name: ${NAME}-clusterset
  namespace: ${NS}
---
apiVersion: cluster.open-cluster-management.io/v1alpha1
kind: Placement
metadata:
  name: ${NAME}-placement
  namespace: ${NS}
spec:
  predicates:
  - requiredClusterSelector:
      labelSelector:
        matchLabels:
          authdeployment: east
---
apiVersion: cluster.open-cluster-management.io/v1alpha1
kind: ManagedClusterSetBinding
metadata:
  name: ${NAME}-clusterset
  namespace: ${NS}
spec:
  clusterSet: ${NAME}-clusterset
---
apiVersion: v1
kind: Secret
metadata:
  name: ${NAME}-client-secret
  namespace: ${NS}
data:
  clientSecret: ${GITHUB_APP_CLIENT_SECRET_B64}
type: Opaque
---
apiVersion: identityconfig.identitatem.io/v1alpha1
kind: AuthRealm
metadata:
  name: ${NAME}
  namespace: ${NS}
spec:
  type: dex
  routeSubDomain: ${ROUTE_SUBDOMAIN}
  placementRef:
    name: ${NAME}-placement
  identityProviders:
    - name: ${IDP_NAME}
      mappingMethod: claim
      type: GitHub
      github:
        clientID: "${GITHUB_APP_CLIENT_ID}"
        clientSecret:
          name: ${NAME}-client-secret
        orgs:
        - ${GITHUB_APP_CLIENT_ORG}
EOF

echo "File ${THE_FILENAME} is generated and ready to \"oc apply -f ${THE_FILENAME}\""
echo ""
echo "Please ensure you have an entry in GitHub under Settings > Developer Settings > OAuth Apps "
echo "for Client ID ${GITHUB_APP_CLIENT_ID} contains:"
echo "    - Homepage URL: https://console-openshift-console.${APPS}"
echo "    - Authorization callback URL: https://${ROUTE_SUBDOMAIN}.${APPS}/callback"
echo "prior to running the \"oc\" command."
echo ""
echo "Add the following labels to any managed cluster you want in the cluster set ${NAME}-clusterset:"
echo "    authdeployment=east"
echo "    cluster.open-cluster-management.io/clusterset=${NAME}-clusterset"
echo ""
