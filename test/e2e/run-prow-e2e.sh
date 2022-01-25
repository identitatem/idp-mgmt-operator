#!/bin/bash
# Copyright Red Hat

set -e

###############################################################################
# Test Setup
###############################################################################

echo $SHARED_DIR


BROWSER=chrome
BUILD_WEB_URL=https://prow.ci.openshift.org/view/gs/origin-ci-test/${JOB_NAME}/${BUILD_ID}
GIT_PULL_NUMBER=$PULL_NUMBER
GIT_REPO_SLUG=${REPO_OWNER}/${REPO_NAME}
HUB_CREDS=$(cat "${SHARED_DIR}/hub-1.json")
OCM_NAMESPACE=open-cluster-management
OCM_ROUTE=multicloud-console
# Hub cluster
export KUBECONFIG="${SHARED_DIR}/hub-1.kc"

OCM_ADDRESS=https://`oc -n $OCM_NAMESPACE get route $OCM_ROUTE -o json | jq -r '.spec.host'`
export CYPRESS_BASE_URL=$OCM_ADDRESS
export CYPRESS_OC_CLUSTER_URL=$(echo $HUB_CREDS | jq -r '.api_url')
export CYPRESS_OC_CLUSTER_USER=$(echo $HUB_CREDS | jq -r '.username')
export CYPRESS_OC_CLUSTER_PASS=$(echo $HUB_CREDS | jq -r '.password')

# Cypress env variables
#export ANSIBLE_URL=$(cat "/etc/e2e-secrets/ansible-url")
#export ANSIBLE_TOKEN=$(cat "/etc/e2e-secrets/ansible-token")
export BROWSER=$BROWSER
export BUILD_WEB_URL=$BUILD_WEB_URL
export CYPRESS_JOB_ID=$PROW_JOB_ID
#export CYPRESS_RBAC_TEST=$(cat "/etc/e2e-secrets/cypress-rbac-test")
export CYPRESS_TEST_MODE=BVT
export GITHUB_USER=$(cat "/etc/ocm-mgdsvcs-e2e-test/github-user")
export GITHUB_TOKEN=$(cat "/etc/ocm-mgdsvcs-e2e-test/github-token")

export ACME_REPO="github.com/acmesh-official/acme.sh"
export IDP_MGMT_OPERATOR_REPO="github.com/identitatem/idp-mgmt-operator"

#export GITHUB_PRIVATE_URL=$(cat "/etc/e2e-secrets/github-private-url")
#export GITHUB_USER=$(cat "/etc/e2e-secrets/github-user")
#export GITHUB_TOKEN=$(cat "/etc/e2e-secrets/github-token")
export GIT_PULL_NUMBER=$PULL_NUMBER
export GIT_REPO_SLUG=$GIT_REPO_SLUG
#export HELM_PRIVATE_URL=$(cat "/etc/e2e-secrets/helm-private-url")
#export HELM_USERNAME=$(cat "/etc/e2e-secrets/github-user")
#export HELM_PASSWORD=$(cat "/etc/e2e-secrets/github-token")
#export HELM_CHART_NAME=$(cat "/etc/e2e-secrets/helm-chart-name")
#export OBJECTSTORE_PRIVATE_URL=$(cat "/etc/e2e-secrets/objectstore-private-url")
#export OBJECTSTORE_ACCESS_KEY=$(cat "/etc/e2e-secrets/objectstore-access-key")
#export OBJECTSTORE_SECRET_KEY=$(cat "/etc/e2e-secrets/objectstore-secret-key")
#export SLACK_TOKEN=$(cat "/etc/e2e-secrets/slack-token")

#In order to verify the signed certifiate, we need to use AWS for route53 domain stuff
export AWS_ACCESS_KEY_ID=$(cat "/etc/ocm-mgdsvcs-e2e-test/aws-access-key")
export AWS_SECRET_ACCESS_KEY=$(cat "/etc/ocm-mgdsvcs-e2e-test/aws-secret-access-key")



# Workaround for "error: x509: certificate signed by unknown authority" problem with oc login
mkdir -p ${HOME}/certificates
OAUTH_POD=$(oc -n openshift-authentication get pods -o jsonpath='{.items[0].metadata.name}')
export CYPRESS_OC_CLUSTER_INGRESS_CA=/certificates/ingress-ca.crt
oc rsh -n openshift-authentication $OAUTH_POD cat /run/secrets/kubernetes.io/serviceaccount/ca.crt > ${HOME}${CYPRESS_OC_CLUSTER_INGRESS_CA}

# managed cluster
MANAGED_CREDS=$(cat "${SHARED_DIR}/managed-1.json")
export CYPRESS_MANAGED_OCP_URL=$(echo $MANAGED_CREDS | jq -r '.api_url')
export CYPRESS_MANAGED_OCP_USER=$(echo $MANAGED_CREDS | jq -r '.username')
export CYPRESS_MANAGED_OCP_PASS=$(echo $MANAGED_CREDS | jq -r '.password')
export CYPRESS_PROW="true"

# Set up git credentials.
echo "--- Setting up git credentials."
{
  echo "https://${GITHUB_USER}:${GITHUB_TOKEN}@${ACME_REPO}.git"
  echo "https://${GITHUB_USER}:${GITHUB_TOKEN}@${IDP_MGMT_OPERATOR_REPO}.git"
} >> ghcreds
git config --global credential.helper 'store --file=ghcreds'



# Set up Quay credentials.
echo "--- Setting up Quay credentials."
export QUAY_TOKEN=$(cat "/etc/acm-cicd-quay-pull")


echo "--- Check current hub cluster info"
oc cluster-info

echo "--- Show managed cluster"
oc get managedclusters

echo "--- Configure OpenShift to use a signed certificate..."
./install-signed-cert.sh

echo "--- Install identity configuration management service for Kubernetes ..."
./install-IDP.sh

echo "--- Running ginkgo E2E tests"
./run-ginkgo-e2e-tests.sh



#echo "--- Running ${CYPRESS_TEST_MODE} tests"
#./start-cypress-tests.sh
