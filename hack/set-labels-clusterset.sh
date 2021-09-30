
#!/bin/bash
# Copyright Red Hat

oc label managedclusters idp-mc  authdeployment=east
oc label managedcluster idp-mc cluster.open-cluster-management.io/clusterset=authrealm-sample-clusterset