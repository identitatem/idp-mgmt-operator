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


export AUTHREALM_GITHUB_NAME=${AUTHREALM_GITHUB_NAME:-"authrealm-sample-github"}
export AUTHREALM_GITHUB_NS=${AUTHREALM_GITHUB_NS:-"authrealm-sample-github-ns"}
export APPS=$(oc get infrastructure cluster -ojsonpath='{.status.apiServerURL}' | cut -d':' -f2 | sed 's/\/\/api/apps/g')
export IDP_NAME=${IDP_NAME:-"github-sample-idp"}
export GITHUB_APP_CLIENT_ID=${GITHUB_APP_CLIENT_ID:-"githubappclientid"}
export GITHUB_APP_CLIENT_SECRET=${GITHUB_APP_CLIENT_SECRET:-"githubappclientsecret"}
export GITHUB_APP_CLIENT_ORG=${GITHUB_APP_CLIENT_ORG:-"githubappclientorg"}
export ROUTE_SUBDOMAIN=${ROUTE_SUBDOMAIN:-"testdomain"}


export GITHUB_FILENAME=/tmp/${AUTHREALM_GITHUB_NAME}".yaml"

export AUTHREALM_LDAP_NAME=${AUTHREALM_LDAP_NAME:-"authrealm-sample-ldap"}
export AUTHREALM_LDAP_NS=${AUTHREALM_LDAP_NS:-"authrealm-sample-ldap-ns"}
export LDAP_BINDPASSWORD=${LDAP_BINDPASSWORD:="ladp bind password"}
export LDAP_HOST=${LDAP_HOST:-"ldap host"}
export LDAP_BIND_DN=${DEXSERVER_LDAP_BIND_DN:-"cn=Manager,dc=example,dc=com"}
export LDAP_USERSEARCH_BASEDN=${DEXSERVER_LDAP_USERSEARCH_BASEDN:-"dc=example,dc=com"}
export LDAP_FILENAME=/tmp/"demo-ldap-authrealm.yaml"

export AUTHREALM_NAME=${AUTHREALM_NAME:-"authrealm-sample"}
export AUTHREALM_NS=${AUTHREALM_NS:-"authrealm-sample-ns"}
export FILENAME_BOTH=/tmp/"demo-both-authrealm.yaml"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
BASE64="base64 -w 0"
if [ "${OS}" == "darwin" ]; then
    BASE64="base64"
fi

GITHUB_APP_CLIENT_SECRET_B64=`echo -n "$GITHUB_APP_CLIENT_SECRET" | $BASE64`OS=$(uname -s | tr '[:upper:]' '[:lower:]')
BASE64="base64 -w 0"
if [ "${OS}" == "darwin" ]; then
    BASE64="base64"
fi

GITHUB_APP_CLIENT_SECRET_B64=`echo -n "$GITHUB_APP_CLIENT_SECRET" | $BASE64`

PS3="Please choose how you want to apply authrealm:"

options=("Github" "LDAP" "Both" "Quit")
select opt in "${options[@]}"
do
    case $opt in
        "Github")
            printf "\n${BLUE}Generating YAML for Github...${CLEAR}\n"

cat > ${GITHUB_FILENAME} <<EOF
---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    control-plane: controller-manager
  name: ${AUTHREALM_GITHUB_NS}
---
apiVersion: cluster.open-cluster-management.io/v1beta1
kind: ManagedClusterSet
metadata:
  name: ${AUTHREALM_GITHUB_NAME}-clusterset
  namespace: ${AUTHREALM_GITHUB_NS}
---
apiVersion: cluster.open-cluster-management.io/v1alpha1
kind: Placement
metadata:
  name: ${AUTHREALM_GITHUB_NAME}-placement
  namespace: ${AUTHREALM_GITHUB_NS}
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
  name: ${AUTHREALM_GITHUB_NAME}-clusterset
  namespace: ${AUTHREALM_GITHUB_NS}
spec:
  clusterSet: ${AUTHREALM_GITHUB_NAME}-clusterset
---
apiVersion: v1
kind: Secret
metadata:
  name: ${AUTHREALM_GITHUB_NAME}-client-secret
  namespace: ${AUTHREALM_GITHUB_NS}
data:
  clientSecret: ${GITHUB_APP_CLIENT_SECRET_B64}
type: Opaque
---
apiVersion: identityconfig.identitatem.io/v1alpha1
kind: AuthRealm
metadata:
  name: ${AUTHREALM_GITHUB_NAME}
  namespace: ${AUTHREALM_GITHUB_NS}
spec:
  type: dex
  routeSubDomain: ${ROUTE_SUBDOMAIN}
  placementRef:
    name: ${AUTHREALM_GITHUB_NAME}-placement
  identityProviders:
    - name: "${IDP_NAME}"
      mappingMethod: claim
      type: GitHub
      github:
        clientID: "${GITHUB_APP_CLIENT_ID}"
        clientSecret:
          name: ${AUTHREALM_GITHUB_NAME}-client-secret
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

printf "${BLUE}YAML file ${GREEN}${GITHUB_FILENAME}${BLUE} is generated.  Apply using \"${GREEN}oc apply -f ${GITHUB_FILENAME}${BLUE}\"${CLEAR}\n\n"
          break
            ;;
        "LDAP")
            printf "\n${BLUE}Generating YAML for LDAP...${CLEAR}\n"

cat > ${LDAP_FILENAME} <<EOF
---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    control-plane: controller-manager
  name: ${AUTHREALM_LDAP_NS}
---
apiVersion: cluster.open-cluster-management.io/v1beta1
kind: ManagedClusterSet
metadata:
  name: ${AUTHREALM_LDAP_NAME}-clusterset
  namespace: ${AUTHREALM_LDAP_NS}
---
apiVersion: cluster.open-cluster-management.io/v1alpha1
kind: Placement
metadata:
  name: ${AUTHREALM_LDAP_NAME}-placement
  namespace: ${AUTHREALM_LDAP_NS}
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
  name: ${AUTHREALM_LDAP_NAME}-clusterset
  namespace: ${AUTHREALM_LDAP_NS}
spec:
  clusterSet: ${AUTHREALM_LDAP_NAME}-clusterset
---
apiVersion: v1
kind: Secret
metadata:
  name: ${AUTHREALM_LDAP_NAME}-ldap-secret
  namespace: ${AUTHREALM_LDAP_NS}
type: Opaque
stringData:
  bindPW: ${LDAP_BINDPASSWORD}
---
apiVersion: identityconfig.identitatem.io/v1alpha1
kind: AuthRealm
metadata:
  name: ${AUTHREALM_LDAP_NAME}
  namespace: ${AUTHREALM_LDAP_NS}
spec:
  type: dex
  routeSubDomain: ${ROUTE_SUBDOMAIN}
  placementRef:
    name: ${AUTHREALM_LDAP_NAME}-placement
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
          name: ${AUTHREALM_LDAP_NAME}-ldap-secret
          namespace: ${AUTHREALM_LDAP_NS}
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
            break
            ;;
        "Both")
            printf "\n${BLUE}Generating YAML for both...${CLEAR}\n"

cat > ${FILENAME_BOTH} <<EOF
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
  name: ${AUTHREALM_NAME}-client-secret
  namespace: ${AUTHREALM_NS}
data:
  clientSecret: ${GITHUB_APP_CLIENT_SECRET_B64}
type: Opaque
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
    - name: "${IDP_NAME}"
      mappingMethod: claim
      type: GitHub
      github:
        clientID: "${GITHUB_APP_CLIENT_ID}"
        clientSecret:
          name: ${AUTHREALM_NAME}-client-secret
        organizations:
        - ${GITHUB_APP_CLIENT_ORG}
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

printf "${BLUE}YAML file ${GREEN}${FILENAME_BOTH}${BLUE} is generated.  Apply using \"${GREEN}oc apply -f ${FILENAME_BOTH}${BLUE}\"${CLEAR}\n\n"
            break
            ;;
        "Quit")
            break
            ;;
        *) echo "invalid option";;
    esac
done