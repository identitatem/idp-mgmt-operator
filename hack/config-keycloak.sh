#!/bin/bash
# Copyright Red Hat
set -e

# After using OpenShift OperatorHub to install KeyCloak community operator, this
# script can be run to help config the KeyCloak instance that can then be used
# for testing IDP OIDC auth against.
#
# This script also assumed that Keycloak and IDP will be installed on the same
# OpenShift hub cluster.

if [ -z $KEYCLOAK_NAMESPACE ]; then
  echo "KEYCLOAK_NAMESPACE for the Keycloak client was not specified"
  exit
fi

echo "Keycloak namespace is: ${KEYCLOAK_NAMESPACE}"

# Need to build the Redirect URI needed by Keycloak client
export OPENID_ROUTE_SUBDOMAIN=${OPENID_ROUTE_SUBDOMAIN:-"openid-subdomain"}

echo "IDP OpenID route subdomain is: ${OPENID_ROUTE_SUBDOMAIN}"

export APPS=$(oc get infrastructure cluster -ojsonpath='{.status.apiServerURL}' | cut -d':' -f2 | sed 's/\/\/api/apps/g')

echo ""
echo "NOTE: This script assumes you are installing Keycloak on the hub cluster where IDP is also installed."
echo "      If this is not the case, you will have to manually patch the Keycloak client redirect URI with the correct value."
echo ""

KEYCLOAK_REDIRECT_URI="https://${OPENID_ROUTE_SUBDOMAIN}.${APPS}/callback"

NS=$(kubectl get namespace $KEYCLOAK_NAMESPACE --ignore-not-found);
if [[ "$NS" ]]; then
  echo "Namespace $KEYCLOAK_NAMESPACE already exists";
else
  echo "Creating namespace $KEYCLOAK_NAMESPACE";
  kubectl create namespace $KEYCLOAK_NAMESPACE;
fi;

if [ -z $KEYCLOAK_REDIRECT_URI ]; then
  echo "KEYCLOAK_REDIRECT_URI for the Keycloak client was not specified"
  exit
fi

echo "Keycloak client redirect URI is: ${KEYCLOAK_REDIRECT_URI}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
BASE64="base64 -w 0"
if [ "${OS}" == "darwin" ]; then
    BASE64="base64"
fi

KEYCLOAK_DIR="/tmp/keycloak"
if [ ! -d "${KEYCLOAK_DIR}" ]; then
  mkdir "${KEYCLOAK_DIR}"
fi
echo "Writing YAML files to directory ${KEYCLOAK_DIR}"

# This only works for kubectl output that returns true or false
wait_for_kubectl_true() {
    CMD=$1
    CMDOPTS=$2
    TIMEOUT=600
    until test $TIMEOUT -le 0; do
      #NOTE:
      #  When this method is called the CR might not yet be created or
      #  right after creation there may be no status at all so
      #  this must be accounted for
      set +e
      CMD_OUTPUT=`kubectl ${CMD} ${CMDOPTS}`
      CMD_RC=$?
      set -e
      if [ $CMD_RC -eq 0 ]; then
        if [ "$CMD_OUTPUT" == "true" ]; then
          break
        fi
      else
        # Don't fail on first check, the CR might not be created yet
        if [ $TIMEOUT -lt 600 ]; then
          echo "Command failed! $CMD $CMDOPTS"
          echo "Return code was $CMD_RC"
          exit 1
        fi
      fi

      #until kubectl ${CMD} ${CMDOPTS} || test $TIMEOUT -le 0; do
      let TIMEOUT=TIMEOUT-10 && sleep 10
      echo -n "."
    done
    if [ $TIMEOUT -le 0 ]; then
        echo "ERROR: ${CMD}"
        exit 1
    fi
    echo ""
}



# Ensure keycloak community operator OR Red Hat SSO keybloak is already installed and ready to use
KEYCLOAK_DEPLOY=$(kubectl get deploy -n $KEYCLOAK_NAMESPACE --no-headers);
if [[ "$KEYCLOAK_DEPLOY" == *"keycloak-operator"* ]]; then
  echo "Check to be sure keycloak operator is installed and ready to use..."
  kubectl wait --for=condition=available deploy/keycloak-operator --timeout 600s -n ${KEYCLOAK_NAMESPACE}
elif [[ "$KEYCLOAK_DEPLOY" == *"rhsso-operator"* ]]; then
  echo "Check to be sure RH SSO operator is installed and ready to use..."
  kubectl wait --for=condition=available deploy/rhsso-operator --timeout 600s -n ${KEYCLOAK_NAMESPACE}
else
  echo "No Keycloak or Red Hat SSO found!"
  exit 1
fi

echo "Create keycloak instance and wait for it to be ready (this will take several minutes)..."

cat << EOF > ${KEYCLOAK_DIR}/keycloak.yaml
apiVersion: keycloak.org/v1alpha1
kind: Keycloak
metadata:
  name: mykeycloak
  namespace: ${KEYCLOAK_NAMESPACE}
  labels:
    app: mykeycloak
spec:
  instances: 1
  externalAccess:
    enabled: True

EOF

kubectl create -f ${KEYCLOAK_DIR}/keycloak.yaml

CMD="get keycloak/mykeycloak -o jsonpath={.status.ready}"
CMDOPTS="-n ${KEYCLOAK_NAMESPACE}"
wait_for_kubectl_true "${CMD}" "${CMDOPTS}"


echo "Create keycloak realm"

cat << EOF > ${KEYCLOAK_DIR}/keycloak-realm.yaml
apiVersion: keycloak.org/v1alpha1
kind: KeycloakRealm
metadata:
  name: myrealm
  namespace: ${KEYCLOAK_NAMESPACE}
  labels:
    realm: myrealm
spec:
  realm:
    realm: "myrealm"
    enabled: True
    displayName: "My Realm"
  instanceSelector:
    matchLabels:
      app: mykeycloak

EOF

kubectl create -f ${KEYCLOAK_DIR}/keycloak-realm.yaml

CMD="get keycloakrealm/myrealm -o jsonpath={.status.ready}"
CMDOPTS="-n ${KEYCLOAK_NAMESPACE}"
wait_for_kubectl_true "${CMD}" "${CMDOPTS}"

SECRET_NAME=`kubectl get keycloak mykeycloak --output="jsonpath={.status.credentialSecret}" -n ${KEYCLOAK_NAMESPACE}`
echo "Keycloak admin secret name is ${SECRET_NAME}"

KEYCLOAK_CREDS=`kubectl get secret ${SECRET_NAME} -n ${KEYCLOAK_NAMESPACE} -o go-template='{{range $k,$v := .data}}{{printf "%s: " $k}}{{if not $v}}{{$v}}{{else}}{{$v | base64decode}}{{end}}{{"\n"}}{{end}}'`
echo "Keycloak admin credentials are:"
echo ${KEYCLOAK_CREDS}

KEYCLOAK_HOST=`oc get route keycloak -n ${KEYCLOAK_NAMESPACE} --template='{{ .spec.host }}'`
KEYCLOAK_URL=https://${KEYCLOAK_HOST}/auth
echo ""
echo "Keycloak URLs:"
echo "Keycloak:                 $KEYCLOAK_URL"
echo "Keycloak Admin Console:   $KEYCLOAK_URL/admin"
echo "Keycloak Account Console: $KEYCLOAK_URL/realms/myrealm/account"
echo "Keycloak OpenID Configuration: $KEYCLOAK_URL/realms/myrealm/.well-known/openid-configuration"
echo ""


echo "Create a Keycloak user"

cat << EOF > ${KEYCLOAK_DIR}/keycloak-user.yaml
apiVersion: keycloak.org/v1alpha1
kind: KeycloakUser
metadata:
  name: myuser
  namespace: ${KEYCLOAK_NAMESPACE}
  labels:
    app: mykeycloak
spec:
  user:
    username: "myuser"
    firstName: "John"
    lastName: "Doe"
    email: "myuser@keycloak.org"
    enabled: True
    credentials:
      - type: "password"
        value: "For-IDP-2-Test"
  realmSelector:
    matchLabels:
      realm: myrealm

EOF

kubectl create -f ${KEYCLOAK_DIR}/keycloak-user.yaml

CMD="get keycloakuser/myuser -o jsonpath={.spec.user.enabled}"
CMDOPTS="-n ${KEYCLOAK_NAMESPACE}"
wait_for_kubectl_true "${CMD}" "${CMDOPTS}"

#wait for user secret to be created
sleep 5

USER_CREDS=`kubectl get secret credential-myrealm-myuser-${KEYCLOAK_NAMESPACE} -n ${KEYCLOAK_NAMESPACE} -o go-template='{{range $k,$v := .data}}{{printf "%s: " $k}}{{if not $v}}{{$v}}{{else}}{{$v | base64decode}}{{end}}{{"\n"}}{{end}}'`
echo "Keycloak User Info: "
echo $USER_CREDS


echo "Create Keycloak client for OpenID connect"

cat << EOF > ${KEYCLOAK_DIR}/keycloak-client.yaml
apiVersion: keycloak.org/v1alpha1
kind: KeycloakClient
metadata:
  name: myclient
  namespace: ${KEYCLOAK_NAMESPACE}
  labels:
    client: myclient
spec:
  realmSelector:
    matchLabels:
      realm: myrealm
  client:
    clientId: myclient
    optionalClientScopes:
    - email
    - profile
    protocol: openid-connect
    redirectUris:
    - ${KEYCLOAK_REDIRECT_URI}
    standardFlowEnabled: true

EOF

kubectl create -f ${KEYCLOAK_DIR}/keycloak-client.yaml

CMD="get keycloakclient/myclient -o jsonpath={.status.ready}"
CMDOPTS="-n ${KEYCLOAK_NAMESPACE}"
wait_for_kubectl_true "${CMD}" "${CMDOPTS}"

#wait for client secret to be created
sleep 10

KEYCLOAK_CLIENT_SECRET_NAME=`oc get keycloakclient -n ${KEYCLOAK_NAMESPACE} -o jsonpath='{.items[0].status.secondaryResources.Secret[0]}'`

KEYCLOAK_CLIENT_CREDS=`kubectl get secret ${KEYCLOAK_CLIENT_SECRET_NAME} -n ${KEYCLOAK_NAMESPACE} -o go-template='{{range $k,$v := .data}}{{printf "%s: " $k}}{{if not $v}}{{$v}}{{else}}{{$v | base64decode}}{{end}}{{"\n"}}{{end}}'`
echo "Keycloak client info: "
echo $KEYCLOAK_CLIENT_CREDS
echo "NOTE: Before running generate-cr.sh, export environment variables"
echo "OPENID_CLIENT_ID and OPENID_CLIENT_SECRET using the values shown above."
echo ""
echo "NOTE: Also export environment variable OPENID_ISSUER using the value:"
echo "$KEYCLOAK_URL/realms/myrealm"

echo ""
echo "Done"
exit 0

# To remove this keycloak configuration you can run:
# oc delete -f ${KEYCLOAK_DIR}/keycloak-user.yaml
# oc delete -f ${KEYCLOAK_DIR}/keycloak-client.yaml
# oc delete -f ${KEYCLOAK_DIR}/keycloak-realm.yaml
# oc delete -f ${KEYCLOAK_DIR}/keycloak.yaml
