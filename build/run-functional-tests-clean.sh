#!/bin/bash
###############################################################################
# Copyright (c) Red Hat, Inc.
# Copyright Red Hat
###############################################################################

set -e
# set -x

CLUSTER_NAME=$PROJECT_NAME-functional-test

echo "delete clusters"
kind delete cluster --name ${CLUSTER_NAME}
# kind delete cluster --name ${CLUSTER_NAME}-managed
