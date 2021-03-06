
#!/bin/bash
# Copyright Red Hat

export NAME=${NAME:-"authrealm-sample"}

if [ -z "$1" ]; then
   echo "cluster name is missing";
   exit 1
fi

oc label managedclusters $1 authdeployment=east $2
oc label managedcluster $1 cluster.open-cluster-management.io/clusterset=${NAME}-clusterset $2