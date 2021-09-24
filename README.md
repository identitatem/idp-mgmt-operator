[comment]: # ( Copyright Red Hat )

# idp-mgmt-operator

This operator is in charge to configure the idp client service. It will also install the
dex operator.

# Run test

`make test`

# Run functional test

`make functional-test-full`

# Installing

## Prereqs

You must meet the following requirements:

- OpenShift Container Platform (OCP) 4.6+
- `oc` (ver. 4.6+)
- `kustomize` (ver. 4.2.0+)

## Ensure you are logged in to the correct OCP hub

```bash
oc cluster-info
```

## Setup Hub to use a signed certificate

This is required for the Dex server we will be using under the covers to authenticate
the managed clusters OpenID OAuth requests. The quickest way to do this is using a tool from
https://github.com/open-cluster-management/sre-tools/wiki/ACM---Day-1#add-an-acme-certificate.

Here is a summary of the commands you need to run:

1. Clone the repo and setup AWS account environment variables

```bash
export AWS_ACCESS_KEY={your AWS Key}
export AWS_SECRET_ACCESS_KEY={your AWS Secret Key}
cd /tmp
git clone https://github.com/acmesh-official/acme.sh.git
cd acme.sh
```

2. Query OCP to setup some environment variables

```bash
export API=$(oc whoami --show-server | cut -f 2 -d ':' | cut -f 3 -d '/' | sed 's/-api././')
export WILDCARD=$(oc get ingresscontroller default -n openshift-ingress-operator -o jsonpath='{.status.domain}')
```

3. If this is the first time running acme.sh, register your email address

```bash
./acme.sh --register-account -m {your email address}
```

4. Generate the certificate

```bash
./acme.sh  --issue   --dns dns_aws -d ${API} -d "*.${WILDCARD}"
```

5. Apply the certificate to OCP

```bash
pushd ${PWD}
cd ${HOME}/.acme.sh/${API}
oc create secret tls router-certs --cert=fullchain.cer --key=${API}.key -n openshift-ingress
oc patch ingresscontroller default -n openshift-ingress-operator --type=merge --patch='{"spec": { "defaultCertificate": { "name": "router-certs" } } }'
popd
```

After running these commands, various pods on the OCP hub will restart in order to use the new certificate. Wait a while, sometimes 10-20 minutes for all the required pods to restart. The OCP UI Overview page can be used to check the overall health of OCP. The Status section will show which operators are updating due to the certificate update.

NOTE: To use the ACME certificate process, you must have Amazon AWS credentials to allow a Route53 domain to
be added for certificate verification during creation.

## Setup GitHub OAuth

Use GitHub as the OAuth provider.

1. On github.com, go to Settings > Developer Settings > OAuth Apps
   (The shortcut is https://github.com/settings/developers)

2. Add a `New OAuth App`. Copy the `Client ID` and `Client Secret` values. You will need them later on.

3. Fill in the `Application name` with something to help identify the owner and hub it will be used for.
   NOTE: If you have more than one hub, each one will need it's own entry.

4. Fill in `Homepage URL` and `Authorization callback URL` with the hub console URL.  
   (NOTE: A little bit later we will correct the `Authorization callback URL` value once we have the generated value.)

5. Click `Register Application`

NOTE: You will need to return to the GitHub OAuth a little bit later to correct the `Authorization callback URL`, once the value is generated for you.

## Install the operator

1. Clone this repo

```bash
git clone https://github.com/identitatem/idp-mgmt-operator.git
cd idp-mgmt-operator
```

2. Login to Red Hat Advanced Cluster Management or Multi Cluster Engine hub
3. Verify you are logged into the hub cluster

```bash
oc cluster-info
```

4. From the cloned idp-mgmt-operator directory:

```bash
make deploy
```

5. Verify the pods are running

There are two pods that should be running:

- idp-mgmt-operator-controller-manager
- idp-mgmt-operator-webhook-service

Check using the following command:

```bash
oc get pods -n idp-mgmt-config
```

## Create an AuthRealm, ManagedClusterSet, etc

We need to create a few custom resources to get idp-mgmt-operator to trigger. Use the `generate-cr.sh` script in
the `hack` directory to generate a yaml file with all the CRs needed.

1. Setup the required environment variables:

```bash
GITHUB_APP_CLIENT_ID={your GitHub App Client ID}
GITHUB_APP_CLIENT_SECRET={your GitHub App Client Secret}
IDP_NAME=sample-idp
NAME=authrealm-sample
NS=authrealm-sample-ns
```

2. Verify you are logged into the hub cluster

```bash
oc cluster-info
```

3. run the script

```bash
cd hack
./generate-cr.sh
```

The `generates-cr.sh` script will create the yaml and output the name of the yaml file. In addition, it will
provide values that need to be copied back into the GitHub Auth App.

A list of labels will also be displayed which will need to be added to the managed cluster to add it to the cluster set.

## Correct the GitHub OAuth App

1. Open https://github.com/settings/developers

2. Select the OAuth App you had created above

3. Correct the `Authorization callback URL` with the value displayed in the `Authorization callback URL` output produced from running the `generates-cr.sh` script.

4. Click `Update application`

## Apply the Custom Resources

Apply the custom resources using the yaml generated by the `generates-cr.sh` script

1. Verify you are logged into the hub cluster

```bash
oc cluster-info
```

2. Apply the YAML

```bash
oc apply -f {your YAML file}
```

## Create or Import a managed cluster

A managed cluster is needed so that the OAuth policy can be configured by idp-mgmt-operator. Make sure
you have a managed cluster defined in the hub.

## Add labels to managed cluster to join the ClusterSet

Use the labels displayed by the `generate-cr.sh` script and add them to the managed cluster.

```bash
oc label managedclusters {managed cluster name} {key=value}
```

## Wait for managed cluster to have new OAuth method configured

It will take a little while for the Placement to cause a ManifestWork to be generated and applied
to the managed cluster. You can monitor the managed cluster's OAuth CR to see if the new OAuth entry appears.
Once the entry appears, you can logout of the managed cluster and when you attempt to login, the new OAuth option will appear on the login screen.

# Remove labels to remove managed cluster from the ClusterSet

If you want to undo the OAuth on the managed cluster, remove the labels you added and the managed cluster will be removed from the cluster set.

```bash
oc label managedclusters {managed cluster name} {key}-
```

# Uninstall the Custom Resources

To delete the custom resources using the yaml generated by the `generates-cr.sh` script

```bash
oc delete -f {your YAML file}
```

# Uninstall the operator

```bash
make undeploy
```
