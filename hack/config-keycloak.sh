#!/bin/bash
# Copyright Red Hat
set -e

# After using OpenShift OperatorHub to install KeyCloak community operator, this
# script can be run to help config the KeyCloak instance that can then be used
# for testing IDP OIDC auth against

#

KEYCLOAK_NAMESPACE=${KEYCLOAK_NAMESPACE:-'mykeycloak'}
echo "Using namespace ${KEYCLOAK_NAMESPACE}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
BASE64="base64 -w 0"
if [ "${OS}" == "darwin" ]; then
    BASE64="base64"
fi

wait_for_command() {
    CMD=$1
    CMDOPTS=$2
    TIMEOUT=600
    until kubectl ${CMD} ${CMDOPTS} || test $TIMEOUT -le 0; do
        let TIMEOUT=TIMEOUT-10 && sleep 10
    done
    if [ $TIMEOUT -le 0 ]; then
        echo "ERROR: ${CMD}"
        exit 1
    fi
}



# Ensure keycloak operator is already installed and ready to use
echo "Check to be sure keycloak operator is installed and ready to use..."
# CMD="rollout status -w deploy/keycloak-operator -n ${KEYCLOAK_NAMESPACE}"
# CMDOPTS=""
# wait_for_command "${CMD}" "${CMDOPTS}"

kubectl wait --for=condition=available deploy/keycloak-operator --timeout 600s -n ${KEYCLOAK_NAMESPACE}


echo "Create keycloak instance and wait for it to be ready (this will take several minutes)..."
cat << EOF > keycloak.yaml
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

kubectl create -f keycloak.yaml

CMD="get keycloak/mykeycloak -o jsonpath='{.status.ready}' -n ${KEYCLOAK_NAMESPACE}"
CMDOPTS=""
wait_for_command "${CMD}" "${CMDOPTS}"


echo "Create keycloak realm"

cat << EOF > keycloak-realm.yaml
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

kubectl create -f keycloak-realm.yaml

CMD="get keycloakrealm/myrealm -o jsonpath='{.status.ready}' -n ${KEYCLOAK_NAMESPACE}"
CMDOPTS=""
wait_for_command "${CMD}" "${CMDOPTS}"

SECRET_NAME=`kubectl get keycloak mykeycloak --output="jsonpath={.status.credentialSecret}" -n ${KEYCLOAK_NAMESPACE}`
echo "Keycloak secret name is ${SECRET_NAME}"

KEYCLOAK_CREDS=`kubectl get secret ${SECRET_NAME} -n ${KEYCLOAK_NAMESPACE} -o go-template='{{range $k,$v := .data}}{{printf "%s: " $k}}{{if not $v}}{{$v}}{{else}}{{$v | base64decode}}{{end}}{{"\n"}}{{end}}'`
echo "Keycloak credentials are:"
echo ${KEYCLOAK_CREDS}

KEYCLOAK_HOST=`oc get route keycloak -n ${KEYCLOAK_NAMESPACE} --template='{{ .spec.host }}'`
KEYCLOAK_URL=https://${KEYCLOAK_HOST}/auth
echo ""
echo "Keycloak:                 $KEYCLOAK_URL"
echo "Keycloak Admin Console:   $KEYCLOAK_URL/admin"
echo "Keycloak Account Console: $KEYCLOAK_URL/realms/myrealm/account"
echo ""


echo "Create a Keycloak user"

cat << EOF > keycloak-user.yaml
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

kubectl create -f keycloak-user.yaml

CMD="get keycloakuser/myuser -o jsonpath='{.status.ready}' -n ${KEYCLOAK_NAMESPACE}"
CMDOPTS=""
wait_for_command "${CMD}" "${CMDOPTS}"


USER_CREDS=`kubectl get secret credential-myrealm-myuser-${KEYCLOAK_NAMESPACE} -n ${KEYCLOAK_NAMESPACE} -o go-template='{{range $k,$v := .data}}{{printf "%s: " $k}}{{if not $v}}{{$v}}{{else}}{{$v | base64decode}}{{end}}{{"\n"}}{{end}}'`
echo "User info: "
echo $USER_CREDS


echo "Create Keycloak client for OpenID connect"


cat << EOF > keycloak-client.yaml
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
    protocol: openid-connect
    rootUrl: https://www.keycloak.org/app/

EOF

kubectl create -f keycloak-client.yaml

CMD="get keycloakclient/myclient -o jsonpath='{.status.ready}' -n ${KEYCLOAK_NAMESPACE}"
CMDOPTS=""
wait_for_command "${CMD}" "${CMDOPTS}"




echo "Done"
