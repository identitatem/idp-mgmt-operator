#!/bin/bash
# Copyright Red Hat

set -e

###############################################################################
# Test Setup
###############################################################################
cd ${IDP_MGMT_OPERATOR_DIR}


export KUBECONFIG="${SHARED_DIR}/hub-1.kc"


# Specify kubeconfig files (the default values are the ones generated in Prow)
HUB_NAME=${HUB_NAME:-"hub-1"}
MANAGED_NAME=${MANAGED_NAME:-"managed-1"}
HUB_KUBE=${HUB_KUBE:-"${SHARED_DIR}/${HUB_NAME}.kc"}
HUB_CREDS=$(cat "${SHARED_DIR}/${HUB_NAME}.json")
MANAGED_KUBE=${MANAGED_KUBE:-"${SHARED_DIR}/${MANAGED_NAME}.kc"}
MANAGED_CREDS=$(cat "${SHARED_DIR}/${MANAGED_NAME}.json")

# Hub cluster
export KUBECONFIG=${HUB_KUBE}

export CLUSTER_SERVER_URL=$(echo $HUB_CREDS | jq -r '.api_url')

#Managed cluster
export MANAGED_CLUSTER_KUBECONFIG="${SHARED_DIR}/managed-1.kc"
# managed cluster
MANAGED_CREDS=$(cat "${SHARED_DIR}/managed-1.json")
export MANAGED_CLUSTER_SERVER_URL=$(echo $MANAGED_CREDS | jq -r '.api_url')

# TODO until the prow job is changed to not specify the name "managed", we will have to hard code
#export MANAGED_CLUSTER_NAME=${MANAGED_NAME}
export MANAGED_CLUSTER_NAME="managed"

echo "--- Show cluster info ..."
oc cluster-info

echo "--- Show managed cluster"
oc get managedclusters

echo "--- Running ginkgo E2E tests"
go install github.com/onsi/ginkgo/v2/ginkgo


make e2e-ginkgo-test


echo "--- Done with ginkgo E2E tests"
