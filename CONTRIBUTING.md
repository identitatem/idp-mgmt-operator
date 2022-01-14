[comment]: # ( Copyright Red Hat )

**Table of Contents**

- [Contributing guidelines](#contributing-guidelines)
    - [Contributions](#contributions)
    - [Certificate of Origin](#certificate-of-origin)
    - [Contributing A Patch](#contributing-a-patch)
    - [Issue and Pull Request Management](#issue-and-pull-request-management)
    - [Pre-check before submitting a PR](#pre-check-before-submitting-a-pr)
    - [Build images](#build-images)

# Contributing guidelines

## Terms

All contributions to the repository must be submitted under the terms of the [Apache Public License 2.0](https://www.apache.org/licenses/LICENSE-2.0).

## Certificate of Origin

By contributing to this project, you agree to the Developer Certificate of Origin (DCO). This document was created by the Linux Kernel community and is a simple statement that you, as a contributor, have the legal right to make the contribution. See the [DCO](DCO) file for details.

## Contributing a patch

1. Submit an issue describing your proposed change to the repository in question. The repository owners will respond to your issue promptly.
2. Fork the desired repository, then develop and test your code changes.
3. Submit a pull request.

## Issue and pull request management

Anyone can comment on issues and submit reviews for pull requests. In order to be assigned an issue or pull request, you can leave a `/assign <your Github ID>` comment on the issue or pull request.

## Pre-check before submitting a PR

Before you commit, please run following commands to check your code and then if it passes commit and create a PR.

```shell
make check
make test
# NOTE: If `make test` returns the error:
#  fork/exec /usr/local/kubebuilder/bin/etcd: no such file or directory
# Please follow comments in the Makefile for `make envtest-tools`
make functional-test-full
```

If the functional test fails, you can connect to the kind cluster by exporting the kubeconfig:

```shell
export KUBECONFIG=kind_kubeconfig.yaml
```

and check the operator logs with:

```shell
oc get pods -n idp-mgmt-config
```
2 pods must be running the installer and the operator one

```shell
oc logs -n idp-mgmt-config <pod_name>
```


## Build images

Make sure your code build passed.

```shell
make docker-build
```

Now, you can follow the [getting started guide](./README.md#getting-started) to work with this repository.

## Generate Deepcopy

```shell
make generate
```

## Geneerate CRD manifests (yaml)

```shell
make manifests
```

## Generate Clients

Important: the project must be in the GOPATH

```shell
make generate-clients
```

## How the installer operator works

- The installer controller monitors the `idpconfigs.identityconfig.identitatem.io` CR and reconcile it.
- When an idpconfig CR is created, the installer deploys the idp-mgmt-operator and the dex-operator.
- The installer controller and other idp-mgmt-operator controllers are baked in the same image and same executable. 
- The installer is launched `idp-mgmt installer` and the idp-mgmt-operator controllers are launched using `idp-mgmt manager`.
- The idp-mgmt-operator is deployed using this [deployment](https://github.com/identitatem/idp-mgmt-operator/blob/58ce9ac08caddbd0ceb54f9799d6c98d9a818a38/deploy/idp-mgmt-operator/manager.yaml) and the image is set as the same of the installer.
- The dex-operator and dex-server images are set using respectively the `RELATED_IMAGE_DEX_OPERATOR` and `RELATED_IMAGE_DEX_SERVER` enviroment variables in the installer deployment CR. The dex-operator install the dex-server and so the `RELATED_IMAGE_DEX_SERVER` environment variable is set as `RELATED_IMAGE_DEX` environment variable in the dex-operator deployment CR in this [deployment](https://github.com/identitatem/idp-mgmt-operator/blob/3d9a0d03159a52d449e004215997f65d41a330dd/deploy/dex-operator/manager.yaml).