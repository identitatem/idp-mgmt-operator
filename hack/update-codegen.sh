#!/usr/bin/env bash
# Copyright Red Hat

set -o errexit
set -o nounset
set -o pipefail
# set -x

SCRIPT_ROOT=$(dirname "${BASH_SOURCE}")/..

# go get -u k8s.io/code-generator
go install k8s.io/code-generator/cmd/{client-gen,lister-gen,informer-gen,deepcopy-gen,register-gen}

# Go installs the above commands to get installed in $GOBIN if defined, and $GOPATH/bin otherwise:
GOBIN="$(go env GOBIN)"
gobin="${GOBIN:-$(go env GOPATH)/bin}"

if [[ "${VERIFY_CODEGEN:-}" == "true" ]]; then
  echo "Running in verification mode"
  VERIFY_FLAG="--verify-only"
fi
COMMON_FLAGS="${VERIFY_FLAG:-} --go-header-file ${SCRIPT_ROOT}/hack/boilerplate.go.txt"

OUTPUT_PKG=github.com/identitatem/${PROJECT_NAME}/api/client
APIS_PKG=github.com/identitatem/${PROJECT_NAME}/api
CLIENTSET_NAME=versioned
CLIENTSET_PKG_NAME=clientset
FQ_APIS=github.com/identitatem/${PROJECT_NAME}/api/identitatem/v1alpha1
echo "Generating deepcopy funcs"
"${gobin}/deepcopy-gen" --input-dirs "${FQ_APIS}" -O zz_generated.deepcopy --bounding-dirs "${APIS_PKG}" ${COMMON_FLAGS}

echo "Generating register at ${FQ_APIS}"
"${gobin}/register-gen" --output-package "${FQ_APIS}" --input-dirs ${FQ_APIS} ${COMMON_FLAGS}

echo "Generating clientset at ${OUTPUT_PKG}/${CLIENTSET_PKG_NAME}"
"${gobin}/client-gen" --clientset-name "${CLIENTSET_NAME}" --input-base "" --input "${FQ_APIS}" --output-package "${OUTPUT_PKG}/${CLIENTSET_PKG_NAME}" ${COMMON_FLAGS}

echo "Generating listers at ${OUTPUT_PKG}/listers"
"${gobin}/lister-gen" --input-dirs "${FQ_APIS}" --output-package "${OUTPUT_PKG}/listers" ${COMMON_FLAGS}

echo "Generating informers at ${OUTPUT_PKG}/informers"
"${gobin}/informer-gen" \
        --input-dirs "${FQ_APIS}" \
        --versioned-clientset-package "${OUTPUT_PKG}/${CLIENTSET_PKG_NAME}/${CLIENTSET_NAME}" \
        --listers-package "${OUTPUT_PKG}/listers" \
        --output-package "${OUTPUT_PKG}/informers" \
        ${COMMON_FLAGS}
