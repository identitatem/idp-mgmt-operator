#!/bin/bash

# Copyright Red Hat

set -e

echo "Running bundle check..."

export IMG=quay.io/identitatem/idp-mgmt-operator:latest
export PREV_BUNDLE_INDEX_IMG=$(cat ./PREV_BUNDLE_INDEX_IMG )
make bundle

if [[ `git diff bundle config` ]]; then
    echo "FAILED: Please git add modified files "
    exit 1
fi
echo "##### bundle check #### Success"
exit 0
