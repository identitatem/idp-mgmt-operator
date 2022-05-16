#!/bin/bash

# Copyright Red Hat

set -e


echo "--- Install keycloak operator ..."


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

export APPS=$(oc get infrastructure cluster -ojsonpath='{.status.apiServerURL}' | cut -d':' -f2 | sed 's/\/\/api/apps/g')
export KEYCLOAK_REDIRECT_URI="https://openid-subdomain.${APPS}/callback"


echo "--- Check namespace - before"
oc get namespaces

echo "--- Create keycloak namespace"
oc create namespace keycloak

echo "--- Install keycloak from OperatorHub"

cat <<EOF | kubectl create -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: keycloak-op-group
  namespace: keycloak
spec:
  targetNamespaces:
    - "keycloak"
EOF

cat <<EOF | kubectl create -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: keycloak-operator
  namespace: keycloak
spec:
  channel: alpha
  installPlanApproval: Automatic
  name: keycloak-operator
  source: community-operators
  sourceNamespace: openshift-marketplace
  startingCSV: keycloak-operator.v18.0.0
EOF

echo "--- Sleep a bit for operator to install..."
sleep 30

echo "--- Check namespace - after"
oc get namespaces

echo "--- Show keycloak deployment"
oc get deployment -n keycloak keycloak-operator -o yaml

echo "--- Check that the pod is running"
oc wait --for=condition=ready pods --all --timeout=5m -n keycloak || {
  echo "ERROR - No keycloak pods running!"
  oc get pods -n keycloak
  oc get deployment -n keycloak keycloak-operator -o yaml
  exit 1
}

echo "--- Create keycloak instance"

cat <<EOF | kubectl create -f -
apiVersion: keycloak.org/v1alpha1
kind: Keycloak
metadata:
  name: mykeycloak
  namespace: keycloak
  labels:
    app: mykeycloak
spec:
  instances: 1
  externalAccess:
    enabled: True
EOF

CMD="get keycloak/mykeycloak -o jsonpath={.status.ready}"
CMDOPTS="-n keycloak"
wait_for_kubectl_true "${CMD}" "${CMDOPTS}"


echo "Create keycloak realm"

cat <<EOF | kubectl create -f -
apiVersion: keycloak.org/v1alpha1
kind: KeycloakRealm
metadata:
  name: myrealm
  namespace: keycloak
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

CMD="get keycloakrealm/myrealm -o jsonpath={.status.ready}"
CMDOPTS="-n keycloak"
wait_for_kubectl_true "${CMD}" "${CMDOPTS}"

KEYCLOAK_HOST=`oc get route keycloak -n keycloak --template='{{ .spec.host }}'`
export KEYCLOAK_URL=https://${KEYCLOAK_HOST}/auth
export OPENID_ISSUER="${KEYCLOAK_URL}/realms/myrealm"

echo "Keycloak host is ${KEYCLOAK_HOST}"

echo "Create a Keycloak user"

cat <<EOF | kubectl create -f -
apiVersion: keycloak.org/v1alpha1
kind: KeycloakUser
metadata:
  name: myuser
  namespace: keycloak
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
        value: ${OPENID_CLIENT_SECRET}
  realmSelector:
    matchLabels:
      realm: myrealm
EOF

CMD="get keycloakuser/myuser -o jsonpath={.spec.user.enabled}"
CMDOPTS="-n keycloak"
wait_for_kubectl_true "${CMD}" "${CMDOPTS}"

#wait for user secret to be created
sleep 5

echo "Create Keycloak client for OpenID connect"

cat <<EOF | kubectl create -f -
apiVersion: keycloak.org/v1alpha1
kind: KeycloakClient
metadata:
  name: myclient
  namespace: keycloak
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


CMD="get keycloakclient/myclient -o jsonpath={.status.ready}"
CMDOPTS="-n keycloak"
wait_for_kubectl_true "${CMD}" "${CMDOPTS}"

#wait for client secret to be created
sleep 10


echo "--- Done installing keycloak and setting up client instance"
