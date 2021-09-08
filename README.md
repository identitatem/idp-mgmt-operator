
[comment]: # ( Copyright Red Hat )
# idp-mgmt-operator
This operator is in charge to configure the idp client service.

# Run test

`make test`

# Run functional test

`make functional-test-full`

# Deploy the operator

1. Login on your hub
2. From the idp-mgmt-operator: `make deploy`

# Example

An example of the Authrealm CR is available [here](test/config/example/github-authrealm.yaml)

You can do a `kubectl apply -f test/config/example/github-authrealm.yaml` to set that example
and then delete the example with `kubectl delete -f test/config/example/github-authrealm.yaml`

This example creates a placement with a matchlabel `authdeployment: east` and so you have to add that label in the managedcluster you want to be managed by the authrealm AND add a label `cluster.open-cluster-management.io/clusterset: cluster-sample` in it.

```yaml
    authdeployment: east
    cluster.open-cluster-management.io/clusterset: cluster-sample
```

# Undeploy the operator

`make undeploy`