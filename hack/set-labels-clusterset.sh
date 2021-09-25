
#!/bin/bash
# Copyright Red Hat

oc label managedclusters itdove-mc  authdeployment=east
oc label managedcluster itdove-mc cluster.open-cluster-management.io/clusterset=authrealm-sample-clusterset