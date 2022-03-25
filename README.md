[comment]: # ( Copyright Red Hat )

# idp-mgmt-operator

This operator is in charge of configuring the idp client service. It will also install the dex operator.

Please fork this repo and clone from the fork.  All your work should be against the forked repo.

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
https://github.com/stolostron/sre-tools/wiki/ACM---Day-1#add-an-acme-certificate.

Here is a summary of the commands you need to run for an AWS based hub:

1. Clone the acmesh-official repo and setup AWS account environment variables

```bash
export AWS_ACCESS_KEY_ID={your AWS Access Key ID}
export AWS_SECRET_ACCESS_KEY={your AWS Secret Access Key}
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
./acme.sh --issue --dns dns_aws -d ${API} -d "*.${WILDCARD}"
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

If you are going to use GitHub as the OAuth provider.

1. On github.com, go to Settings > Developer Settings > OAuth Apps
   (The shortcut is https://github.com/settings/developers)

2. Add a `New OAuth App`. Copy the `Client ID` and `Client Secret` values. You will need them later on.

3. Fill in the `Application name` with something to help identify the owner and hub it will be used for.
   NOTE: If you have more than one hub, each one will need it's own entry.

4. Fill in `Homepage URL` and `Authorization callback URL` with the hub console URL.  
   (NOTE: A little bit later we will correct the `Authorization callback URL` value once we have the generated value.)

5. Click `Register Application`

NOTE: You will need to return to the GitHub OAuth a little bit later to correct the `Authorization callback URL`, once the value is generated for you.

## Setup OpenID Connect (OIDC) Keycloak OAuth

If you are going to use OpenID Connect as the OAuth provider and want to use Keycloak.  

1. Use Operator Hub to install the **Keycloak** community operator to the `keycloak` namespace.

1. Configure the following environment variables to define required keycloak Information
```
export KEYCLOAK_NAMESPACE=keycloak
export OPENID_ROUTE_SUBDOMAIN=openid-subdomain
```

1. Generate and apply Keycloak YAML to configure a realm, a client and a user on Keycloak. **Ensure you are `oc login` to the hub
cluster before you run the script!** Use the script in
the `hack` directory:
```
cd hack
./config-keycloak.sh
```
1. Save the output of the script.  It will provide you with information needed for the following environment variables used by the `generate-cr.sh` script to create an AuthRealm for OpenID and Keycloak:
  - OPENID_CLIENT_ID
  - OPENID_CLIENT_SECRET
  - OPENID_ISSUER     


## Install the operator using one of two methods

### Method 1: Install the operator from this repo

1. Fork and clone this repo

```bash
git clone https://github.com/<git username>/idp-mgmt-operator.git
cd idp-mgmt-operator
```

2. Login to Red Hat Advanced Cluster Management or Multi Cluster Engine hub
3. Verify you are logged into the hub cluster

```bash
oc cluster-info
```

4. From the cloned idp-mgmt-operator directory:

```bash
export IMG=quay.io/<your_user>/idp-mgmt-operator:<tag_you_want_to_use>
make docker-build docker-push deploy
```

5. Verify the installer is running

There is one pod that should be running:

- idp-mgmt-installer-controller-manager

Check using the following command:

```bash
oc get pods -n idp-mgmt-config
```

6. Create an idpconfig

```bash
echo '
apiVersion: identityconfig.identitatem.io/v1alpha1
kind: IDPConfig
metadata:
  name: idp-config
  namespace: idp-mgmt-config
spec:' | oc create -f -
```

7. Verify pods are running

There is now three pods that should be running

- idp-mgmt-installer-controller-manager
- idp-mgmt-operator-manager
- idp-mgmt-webhook-service

Check using the following command:

```bash
oc get pods -n idp-mgmt-config
```
### Method 2: Install the operator from a Catalog

**NOTE**: To install via the catalog, you must be on OpenShift 4.8.12 or newer due to [this OLM bug](https://bugzilla.redhat.com/show_bug.cgi?id=1969902) - a backport to OCP 4.7 is in progress.

If you built an operator bundle and catalog as [documented later](#build-a-bundle-and-catalog), or if you wish to use the latest published operator bundle and catalog to deploy:
1. Login to Red Hat Advanced Cluster Management or Multi Cluster Engine hub
2. Verify you are logged into the hub cluster

```bash
oc cluster-info
```

3. From the idp-mgmt-operator directory run `make deploy-catalog` - this will create a CatalogSource on your cluster.  
4. Create a subscription to the operator via the OperatorHub UI or by applying a subscription to the Operator - which looks something like:
```
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: idp-mgmt-operator
  namespace: idp-mgmt-config
spec:
  channel: alpha
  installPlanApproval: Automatic
  name: idp-mgmt-operator
  source: idp-mgmt-config-catalog
  sourceNamespace: idp-mgmt-config
  startingCSV: idp-mgmt-operator.v0.3.0
```
5. Create an idpconfig

```bash
echo '
apiVersion: identityconfig.identitatem.io/v1alpha1
kind: IDPConfig
metadata:
  name: idp-config
  namespace: idp-mgmt-config
spec:' | oc create -f -
```

6. Wait for the operator to install - then move on to the next step!

## Create an AuthRealm, ManagedClusterSet, etc

We need to create a few custom resources to get idp-mgmt-operator to trigger. Use the `generate-cr.sh` script in
the `hack` directory to generate a yaml file with all the CRs needed.

1. Setup the required environment variables:

```bash
export GITHUB_APP_CLIENT_ID={your GitHub App Client ID}
export GITHUB_APP_CLIENT_SECRET={your GitHub App Client Secret}
export GITHUB_APP_CLIENT_ORG={your GitHub organization}
export IDP_NAME=sample-idp
export AUTHREALM_GITHUB_NAME=authrealm-sample-github
export AUTHREALM_GITHUB_NS=authrealm-sample-github-ns
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

---
---
---

# Build a Bundle and Catalog

You can build and push an operator bundle and installable catalog for this operator, which will allow you to apply a CatalogSource to your cluster and install via the OperatorHub UI or via an Operator Subscription object.  

To build and publish a bundle and catalog:
1. [IMPORTANT] Use your own registry!  Do the following to enable you to use your own registry and create repos in your quay.io org (your user org) named `idp-mgmt-operator-bundle` and `idp-mgmt-operator-catalog` and, if so,
```bash
export IMAGE_TAG_BASE=quay.io/\<your-user\>/idp-mgmt-operator
```
  **If you don’t do this, you need push permissions to identitatem - and you’ll push straight to the source identitatem org. - Which only the build system should do!**

  **You might need to make these personal repos public to allow the images to be downloaded by OLM**
2. Set the **VERSION** as a timestamp.  For example:
```bash
export VERSION=`date -u "+0.0.0-%Y%m%d-%H-%M-%S"`
```
3. `export` DOCKER_USER and DOCKER_PASS equal to a docker user and password that will allow you to push to the quay repositories outlined in step 1.
4. If you want to use a specific idp-mgmt-operator image, set the **IMG** environment variable to that image. For example, to point to a PR built image:
```bash
export IMG=quay.io/identitatem/idp-mgmt-operator@sha256:f1303674fc463cbc3834d3dd6c9d023cc991144a3e170c496dac5d2a44459d5c
```
Otherwise use:
```bash
export IMG=quay.io/identitatem/idp-mgmt-operator:latest
```
You can also use an image located in your own repo. To create such image run:
```bash
export IMG=quay.io/<your_user>/idp-mgmt-operator:<tag_you_want_to_use>
make docker-build docker-push
```
5. If you upgrade from a prior version set the PREV_BUNDLE_INDEX_IMG to the previous catalog image
```bash
export PREV_BUNDLE_INDEX_IMG=quay.io/<repo_name>/idp-mgmt-operator-catalog:<previous_catalog_version>
```
6. run `make bundle` to generate bundle manifests
7. run `make publish` - this should acquire any dependencies and push to quay!
8. To test, run `make deploy-catalog`.  This will create a catalogsource on your hub cluster and you can install the test catalog from OperatorHub.
9. PS: Before commit, double check the file [idp-mgmt-operator.clusterserviceversion.yaml](bundle/manifests/idp-mgmt-operator.clusterserviceversion.yaml) to make sure it doesn't contain your own name, image and version as this file is used as-is by downstream.

# ClusterServiceVersion updates
After making changes to `config/manifests/bases/idp-mgmt-operator.clusterserviceversion.yaml` you will need to run `make bundle IMG=quay.io/identitatem/idp-mgmt-operator:latest` to get the changes into `bundle/manifests/idp-mgmt-operator.clusterserviceversion.yaml`.  You should not see any image or version changes in `bundle/manifests/idp-mgmt-operator.clusterserviceversion.yaml`. If you do, revert those changes before checking in the file.

# Tagging and Generating a Release

We have a GitHub action defined to generate a tagged bundle and catalog image when a SemVer GitHub tag is created on this repo.  To create a new release that will generate a versioned Bundle/Catalog there are two methods:

**NOTE: Filter more generic as of Feb 2022 ~~The Major.Minor SemVer is watched by the mid/downstream processes.  If you must change the version from 0.2.*, you MUST notify CICD so they can change the tag_filter they use to watch for changes!!!~~**


## Method One - Use GitHub UI

1. Run the following command to generate the value we will use as part of the release and tag for non-shipped releases
```bash
date -u "+0.3.1-%Y%m%d-%H-%M-%S"
```
  If you are building a release candidate, the format should be **0.3.1-rc#**.  (Where **#** is the release candidate number. Do not use UPPERCASE characters!)  If you are building the final release, the format should be **0.3.1**.

1. Go to dex-operator github page and select **Releases** (https://github.com/identitatem/dex-operator/releases)
1. Select **Draft a new release**
1. For **Release title**, enter **v** and then paste the value from the date command above
1. Select **Choose a tag**
1. In the **Find or create a new tag field**, paste the value from the date command above
1. Select **Create new tab on publish**
1. Select **Publish release**.
   This will cause a github action to start.  Once the github action is complete, the new dex-operator quay image will be available at https://quay.io/repository/identitatem/dex-operator?tab=tags.  Now we need to pull this new dex-operator image
   into the idp-mgmt-operator.
1. In your fork of the https://github.com/identitatem/idp-mgmt-operator repo, create a new branch
1. Update `go.mod` entry `github.com/identitatem/dex-operator` to reference the new tag, for example `github.com/identitatem/dex-operator 0.3.1-20220317-12-30-05`  
1. Run `go mod tidy -compat=1.17`.  The `go.mod` entry will be updated to the correct value and the end of the new value will have the dex-operator git tag commit id.  
1. Update https://github.com/identitatem/idp-mgmt-operator/blob/main/config/installer/installer.yaml
so RELATED_IMAGE_DEX_OPERATOR points to the new dex-operator image in quay.
1. Export IMG variable to point to latest
```bash
export IMG=quay.io/identitatem/idp-mgmt-operator:latest
```
1. Run `make bundle` to update https://github.com/identitatem/idp-mgmt-operator/blob/main/bundle/manifests/idp-mgmt-operator.clusterserviceversion.yaml.  **You should not see any changes to `image:`, `version:` or `replaces:` in `bundle/manifests/idp-mgmt-operator.clusterserviceversion.yaml`. If you do, revert those changes before checking in the file.  Also MAKE SURE `replaces` IS CORRECT AND POINTS TO THE PREVIOUS VERSION so upgrade works properly!**
1. Test the changes using `make test` then `make deploy`.  Apply an IDPConfig and apply an AuthRealm to a managed cluster and be sure it works.
1. Commit the PR changes and get them reviewed and merged.
1. Run the following command to generate the value we will use as part of the release and tag (OR possibly use the same tag dex-operator used)
```bash
date -u "+0.3.1-%Y%m%d-%H-%M-%S"
```
 If you are building a release candidate, the format should be **0.3.1-rc#**.  (Where **#** is the release candidate number. Do not use UPPERCASE characters!)  If you are building the final release, the format should be **0.3.1**.

1. Go to idp-mgmt-operator github page and select **Releases** (https://github.com/identitatem/idp-mgmt-operator/releases)
1. Select **Draft a new release**
1. For **Release title**, enter v and then paste the value from the date command above
1. Select Choose a tag
1. In the **Find or create a new tag** field, paste the value from the date command above
1. Select **Create new tab on publish**
1. Select **Publish release**.
   This will cause a github action to start.  Once the github action is complete a
   new Operator Hub catalog will be available at
   https://quay.io/repository/identitatem/idp-mgmt-operator-catalog?tab=tags which you can reference in
   your catalog source.



## Method Two - Create a tag in non-forked repo (NOT USED)
1. `git tag <semver-tag>`
2. `git push --tags`
3. Wait for the [GitHub Action to complete](https://github.com/identitatem/idp-mgmt-operator/actions).
4. The action will create and push an [idp-mgmt-operator-bundle](https://quay.io/repository/identitatem/idp-mgmt-operator-bundle) and [idp-mgmt-operator-catalog](https://quay.io/repository/identitatem/idp-mgmt-operator-catalog) image and push to quay, tagged with the same SemVer tag you applied to the GitHub repo to generate that release.  

In order to deploy this version - follow the Catalog deploy method but set:
```
VERSION=<semver-tag>
```
