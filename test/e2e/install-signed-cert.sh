#!/bin/bash

# Copyright Red Hat

echo "Creating a signed certificate ..."

# Based on info at https://github.com/open-cluster-management/sre-tools/wiki/ACM---Day-1#add-an-acme-certificate

#In order to verify the signed certifiate, we need to use AWS for route53 domain stuff
export AWS_ACCESS_KEY_ID=$(cat "/etc/e2e-secrets/aws-access-key")
export AWS_SECRET_ACCESS_KEY=$(cat "/etc/e2e-secrets/aws-secret-access-key")

#export GITHUB_PRIVATE_URL=$(cat "/etc/e2e-secrets/github-private-url")
export GITHUB_USER=$(cat "/etc/e2e-secrets/github-user")
export GITHUB_TOKEN=$(cat "/etc/e2e-secrets/github-token")
#export GIT_PULL_NUMBER=$PULL_NUMBER
#export GIT_REPO_SLUG=$GIT_REPO_SLUG

acme_dir=$(mktemp -d -t acme-XXXXX)
cd "$acme_dir" || exit 1
export HOME="$acme_dir"

# Set up git credentials.
echo "Setting up git credentials."
ACME_REPO=github.com/acmesh-official/acme.sh
{
    echo "https://${GITHUB_USER}:${GITHUB_TOKEN}@${ACME_REPO}.git"
} >> ghcreds
git config --global credential.helper 'store --file=ghcreds'

# Set up repo URLs.
acme_url="https://${ACME_REPO}.git"
acme_git_dir="${acme_dir}/acme.sh"
echo "Cloning repo..."
git clone "$acme_url" "$acme_git_dir" || {
    echo "ERROR Could not clone release repo $acme_url"
    exit 1
}

cd ${acme_git_dir}

echo "Check current cluster info"
oc cluster-info

export API=$(oc whoami --show-server | cut -f 2 -d ':' | cut -f 3 -d '/' | sed 's/-api././')
export WILDCARD=$(oc get ingresscontroller default -n openshift-ingress-operator -o jsonpath='{.status.domain}')

./acme.sh --register-account -m cahl@redhat.com || {
    echo "ERROR Could not register email address"
    exit 1
}

echo "Generate the signed certificate..."

#./acme.sh  --issue   --dns dns_aws -d ${API} -d "*.${WILDCARD}"
# The above sometimes returns a 503 error, so use a different server
./acme.sh  --issue   --dns dns_aws -d ${API} -d "*.${WILDCARD}" --server letsencrypt || {
    echo "ERROR Could not create signed certificate"
    exit 1
}


echo "Install the signed certificate ..."

pushd ${PWD}
#TODO - check to see if this will HOME dir will work, otherwise set $LE_WORKING_DIR before generating signed certificate
cd ${HOME}/.acme.sh/${API}
oc create secret tls router-certs --cert=fullchain.cer --key=${API}.key -n openshift-ingress
oc patch ingresscontroller default -n openshift-ingress-operator --type=merge --patch='{"spec": { "defaultCertificate": { "name": "router-certs" } } }'
popd


echo "OpenShift nodes need about 10 minutes to restart and use new signed certificate ..."
# TODO - Add a wait loop checking various pods that get restarted
sleep 600

echo "Done setting up signed certificate"
