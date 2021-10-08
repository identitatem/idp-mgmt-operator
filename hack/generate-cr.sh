#!/bin/bash
# Copyright Red Hat

# Color codes for bash output
BLUE='\e[36m'
GREEN='\e[32m'
RED='\e[31m'
YELLOW='\e[33m'
CLEAR='\e[39m'
if [[ "$COLOR" == "False" || "$COLOR" == "false" ]]; then
    BLUE='\e[39m'
    GREEN='\e[39m'
    RED='\e[39m'
    YELLOW='\e[39m'
fi

printf "\n${BLUE}Gathering OpenShift cluster URL...${CLEAR}\n"

export NAME=${NAME:-"authrealm-sample"}
export NS=${NS:-"authrealm-sample-ns"}
export APPS=$(oc get infrastructure cluster -ojsonpath='{.status.apiServerURL}' | cut -d':' -f2 | sed 's/\/\/api/apps/g')
export IDP_NAME=${IDP_NAME:-"github-sample-idp"}
export GITHUB_APP_CLIENT_ID=${GITHUB_APP_CLIENT_ID:-"githubappclientid"}
export GITHUB_APP_CLIENT_SECRET=${GITHUB_APP_CLIENT_SECRET:-"githubappclientsecret"}
export GITHUB_APP_CLIENT_ORG=${GITHUB_APP_CLIENT_ORG:-"githubappclientorg"}
export ROUTE_SUBDOMAIN=${ROUTE_SUBDOMAIN:-"testdomain"}


export LDAP_BIND_PASSWORD=${LDAP_BIND_PASSWORD:-"admin"}

export LDAP_HOST=${LDAP_HOST:-"adf558f301d884463a9d44329fbafc4c-145647244.us-east-1.elb.amazonaws.com:636"}
export LDAP_BIND_DN=${DEXSERVER_LDAP_BIND_DN:-"cn=Manager,dc=example,dc=com"}
export LDAP_USERSEARCH_BASEDN=${DEXSERVER_LDAP_USERSEARCH_BASEDN:-"dc=example,dc=com"}

export THE_FILENAME=/tmp/${NAME}".yaml"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
BASE64="base64 -w 0"
if [ "${OS}" == "darwin" ]; then
    BASE64="base64"
fi

GITHUB_APP_CLIENT_SECRET_B64=`echo -n "$GITHUB_APP_CLIENT_SECRET" | $BASE64`

printf "\n${BLUE}Generating YAML...${CLEAR}\n"

cat > ${THE_FILENAME} <<EOF
---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    control-plane: controller-manager
  name: ${NS}
---
apiVersion: cluster.open-cluster-management.io/v1beta1
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
apiVersion: cluster.open-cluster-management.io/v1beta1
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
apiVersion: v1
kind: Secret
metadata:
  name: ${NAME}-ldap-secret
  namespace: ${NS}
type: Opaque
stringData:
  bindPW: ${LDAP_BIND_PASSWORD}
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
    - name: "${IDP_NAME}"
      mappingMethod: claim
      type: GitHub
      github:
        clientID: "${GITHUB_APP_CLIENT_ID}"
        clientSecret:
          name: ${NAME}-client-secret
        organizations:
        - ${GITHUB_APP_CLIENT_ORG}
    
EOF

printf "\n${BLUE}Before using the generated YAML:${CLEAR}\n\n"

printf "${BLUE}1) Ensure there is an entry in GitHub under Settings > Developer Settings > OAuth Apps${CLEAR}\n"
printf "${BLUE}for Client ID ${GREEN}${GITHUB_APP_CLIENT_ID}${BLUE} which contains:${CLEAR}\n"
printf "    - ${GREEN}Homepage URL:${YELLOW} https://console-openshift-console.${APPS}${CLEAR}\n"
printf "    - ${GREEN}Authorization callback URL:${YELLOW} https://${ROUTE_SUBDOMAIN}.${APPS}/callback${CLEAR}\n"
printf "${BLUE}prior to running the ${GREEN}oc apply${BLUE} command shown below.\n\n${CLEAR}"

printf "${BLUE}2) Add the following labels to any managed cluster you want in the cluster set ${GREEN}${NAME}-clusterset${BLUE}:${CLEAR}\n"
printf "    ${GREEN}authdeployment=east${CLEAR}\n"
printf "    ${GREEN}cluster.open-cluster-management.io/clusterset=${NAME}-clusterset${CLEAR}\n"
printf "${BLUE}by using the command \"${GREEN}oc label managedclusters ${YELLOW}<managed cluster name> <label>${BLUE}\"${CLEAR}\n\n"

printf "${BLUE}YAML file ${GREEN}${THE_FILENAME}${BLUE} is generated.  Apply using \"${GREEN}oc apply -f ${THE_FILENAME}${BLUE}\"${CLEAR}\n\n"
