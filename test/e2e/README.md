## Run E2E tests locally in OCP cluster

#### Prerequisites:
- An OCP cluster with ACM or MCE installed (Hub cluster)
- A managed cluster imported successfully into the hub cluster
- The IDP management operator installed and running on the hub cluster

#### Steps to run tests:

1. Clone this repository and enter its root directory:
    ```
    git clone git@github.com:identitatem/idp-mgmt-operator.git && cd idp-mgmt-operator
    ```

2. Set configuration for the OCP Hub cluster:
   - export `KUBECONFIG` environment variable to the kubeconfig of the Hub OCP cluster:
     ```
     export KUBECONFIG=<kubeconfig-file-of-the-ocp-hub-cluster>
     ```
   - export `CLUSTER_SERVER_URL` environment variable to the cluster server URL of the Hub OCP cluster:
     ```
     export CLUSTER_SERVER_URL=<cluster-server-url-of-the-ocp-hub-cluster>
     ```

3. Set configuration for the managed cluster:
   - export `MANAGED_CLUSTER_KUBECONTEXT` environment variable to the kubecontext of the managed cluster:

        ```
        export MANAGED_CLUSTER_KUBECONTEXT=<kubecontext-of-the-managed-cluster>
        ```
    - optionally, you can copy the kubeconfig of the managed cluster into a new file and then set the `MANAGED_CLUSTER_KUBECONFIG` environment variable. (In this case, you don't need to set `MANAGED_CLUSTER_KUBECONTEXT`)
     Example:
      ```
      cp {IMPORT_CLUSTER_KUBE_CONFIG_PATH} ~/.kube/import-kubeconfig
      export MANAGED_CLUSTER_KUBECONFIG="~/.kube/import-kubeconfig"
      ```
   - export `MANAGED_CLUSTER_SERVER_URL` environment variable to the cluster server URL of the managed cluster:
     ```
     export MANAGED_CLUSTER_SERVER_URL=<cluster-server-url-of-the-managed-cluster>
     ```
   - export `MANAGED_CLUSTER_NAME` environment variable to the name of the managed cluster:
     ```
     export MANAGED_CLUSTER_NAME=<name-of-the-managed-cluster>
     ```

4. export `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET` environment variables to contain the client ID and secret of the GitHub OAuth application that will be used as the IDP for the e2e tests. (Note: Make sure that the `Homepage URL` and `Authorization callback URL` in the GitHub Oauth app correspond to the OCP hub cluster you are using)
   
5. Then execute the following command to run e2e testing:

    ```
    make e2e-ginkgo-test
    ```