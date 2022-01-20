#!/bin/bash

# Copyright Red Hat

echo "Install identity configuration management service for Kubernetes ..."

idp_dir=$(mktemp -d -t idp-XXXXX)
cd "$idp_dir" || exit 1
export HOME="$idp_dir"

# Set up repo URLs.
echo "Cloning branch idp-mgmt-operator ${PULL_BASE_REF}"
idp_mgmt_operator_url="https://${IDP_MGMT_OPERATOR_REPO}.git"
idp_mgmt_operator_dir="${idp_dir}/idp-mgmt-operator"
git clone -b "${PULL_BASE_REF}" "$idp_mgmt_operator_url" "$idp_mgmt_operator_dir" || {
    log "ERROR Could not clone branch ${PULL_BAWE_REF} from idp-mgmt-operator repo $idp_mgmt_operator_url"
    exit 1
}


cd ${idp_mgmt_operator_dir}

export IMG="quay.io/identitatem/idp-mgmt-operator:0.1-PR${PULL_NUMBER}-${PULL_PULL_SHA}"
echo "Quay image is ${IMG}"

echo "Start deploy"
make deploy

sleep 20

echo "Check that the pod is running"
oc get pods -n idp-mgmt-config

echo "Create IDPConfig"
echo '
apiVersion: identityconfig.identitatem.io/v1alpha1
kind: IDPConfig
metadata:
  name: idp-config
  namespace: idp-mgmt-config
spec:' | oc create -f -


sleep 20

echo "Check for more pods running"
oc get pods -n idp-mgmt-config

echo "Done installing identity configuration management service for Kubernetes"
