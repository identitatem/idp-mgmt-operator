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


export THE_FILENAME=/tmp/${NAME}".yaml"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
BASE64="base64 -w 0"
if [ "${OS}" == "darwin" ]; then
    BASE64="base64"
fi

GITHUB_APP_CLIENT_SECRET_B64=`echo -n "$GITHUB_APP_CLIENT_SECRET" | $BASE64`

printf "\n${BLUE}Generating YAML for Github...${CLEAR}\n"

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
printf "    - ${GREEN}Authorization callback URL:${YELLOW} https://${NAME}-${ROUTE_SUBDOMAIN}.${APPS}/callback${CLEAR}\n"
printf "${BLUE}prior to running the ${GREEN}oc apply${BLUE} command shown below.\n\n${CLEAR}"

printf "${BLUE}2) Add the following labels to any managed cluster you want in the cluster set ${GREEN}${NAME}-clusterset${BLUE}:${CLEAR}\n"
printf "    ${GREEN}authdeployment=east${CLEAR}\n"
printf "    ${GREEN}cluster.open-cluster-management.io/clusterset=${NAME}-clusterset${CLEAR}\n"
printf "${BLUE}by using the command \"${GREEN}oc label managedclusters ${YELLOW}<managed cluster name> <label>${BLUE}\"${CLEAR}\n\n"

printf "${BLUE}YAML file ${GREEN}${THE_FILENAME}${BLUE} is generated.  Apply using \"${GREEN}oc apply -f ${THE_FILENAME}${BLUE}\"${CLEAR}\n\n"


export AUTHREALM_NAME=${AUTHREALM_NAME:-"authrealm-sample"}
export AUTHREALM_NS=${AUTHREALM_NS:-"authrealm-sample-ns"}
export LDAP_BINDPASSWORD=${LDAP_BINDPASSWORD:="ladp bind password"}
export LDAP_HOST=${LDAP_HOST:-"ldap host"}
export LDAP_BIND_DN=${DEXSERVER_LDAP_BIND_DN:-"cn=Manager,dc=example,dc=com"}
export LDAP_USERSEARCH_BASEDN=${DEXSERVER_LDAP_USERSEARCH_BASEDN:-"dc=example,dc=com"}
export LDAP_FILENAME=/tmp/"demo-ldap-authrealm.yaml"

printf "\n${BLUE}Generating YAML for LDAP...${CLEAR}\n"

cat > ${LDAP_FILENAME} <<EOF
---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    control-plane: controller-manager
  name: ${AUTHREALM_NS}
---
apiVersion: cluster.open-cluster-management.io/v1beta1
kind: ManagedClusterSet
metadata:
  name: ${AUTHREALM_NAME}-clusterset
  namespace: ${AUTHREALM_NS}
---
apiVersion: cluster.open-cluster-management.io/v1alpha1
kind: Placement
metadata:
  name: ${AUTHREALM_NAME}-placement
  namespace: ${AUTHREALM_NS}
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
  name: ${AUTHREALM_NAME}-clusterset
  namespace: ${AUTHREALM_NS}
spec:
  clusterSet: ${AUTHREALM_NAME}-clusterset
---
apiVersion: v1
kind: Secret
metadata:
  name: ${AUTHREALM_NAME}-ldap-secret
  namespace: ${AUTHREALM_NS}
type: Opaque
stringData:
  bindPW: ${LDAP_BINDPASSWORD}
---
apiVersion: identityconfig.identitatem.io/v1alpha1
kind: AuthRealm
metadata:
  name: ${AUTHREALM_NAME}
  namespace: ${AUTHREALM_NS}
spec:
  type: dex
  routeSubDomain: ${ROUTE_SUBDOMAIN}
  placementRef:
    name: ${AUTHREALM_NAME}-placement
  ldapExtraConfigs:
    openldap: 
      baseDN: ${LDAP_USERSEARCH_BASEDN}
      filter: "(objectClass=person)"
  identityProviders:
    - name: openldap
      type: LDAP
      mappingMethod: add
      ldap:
        url: ${LDAP_HOST}
        insecure: true
        bindDN: ${LDAP_BIND_DN}
        bindPassword:
          name: ${AUTHREALM_NAME}-ldap-secret
          namespace: ${AUTHREALM_NS}
        attributes:
          id: 
            - DN
          preferredUsername: 
            - mail
          name: 
            - cn
          email: 
            - mail  
EOF

printf "${BLUE} Add the following labels to any managed cluster you want in the cluster set ${GREEN}${NAME}-clusterset${BLUE}:${CLEAR}\n"
printf "    ${GREEN}authdeployment=east${CLEAR}\n"
printf "    ${GREEN}cluster.open-cluster-management.io/clusterset=${NAME}-clusterset${CLEAR}\n"
printf "${BLUE}by using the command \"${GREEN}oc label managedclusters ${YELLOW}<managed cluster name> <label>${BLUE}\"${CLEAR}\n\n"


printf "${BLUE}YAML file ${GREEN}${LDAP_FILENAME}${BLUE} is generated.  Apply using \"${GREEN}oc apply -f ${LDAP_FILENAME}${BLUE}\"${CLEAR}\n\n"
