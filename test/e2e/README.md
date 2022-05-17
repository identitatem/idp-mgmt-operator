[comment]: # ( Copyright Red Hat )
## Run E2E tests (Ginkgo) locally in OCP cluster

#### Prerequisites:
- An OCP cluster with ACM or MCE installed (Hub cluster)
- A managed cluster imported successfully into the hub cluster
- The IDP management operator installed and running on the hub cluster
- For OpenID tests, Keycloak operator must be installed in keycloak namespace

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

4. For the GitHub IDP export the following environment variables:
    ```bash
     # Client ID of the GitHub OAuth App
     export GITHUB_APP_CLIENT_ID=...
     # Client Secret of the GitHub OAuth App
     export GITHUB_APP_CLIENT_SECRET=...   
     ```
    Note: The `Homepage URL` and `Authorization callback URL` in the GitHub OAuth app should correspond to the OCP hub cluster you are using.

5. For the LDAP (Azure Active Directory) IDP export the following environment variables:
    ```bash
     # Azure Active Directory Domain Services Secure LDAP Host
     export LDAP_AZURE_HOST=...
     # Bind DN
     export LDAP_AZURE_BIND_DN=...
     # Bind password (The bindDN and bindPW are used as credentials to search for users and passwords)
     export LDAP_AZURE_BIND_PASSWORD=...
     # Base DN to start the search from
     export LDAP_AZURE_BASE_DN=...
     # Signed certificate for secure LDAP
     export LDAP_AZURE_SERVER_CERT=...     
     ```
6. For the OpenID (Keycloak) IDP run the following script to setup keycloak and export a few environment variables:
   ```bash
    # Setup OpenID client secret
    export OPENID_CLIENT_SECRET=$(head /dev/urandom | LC_CTYPE=C tr -dc a-z0-9 | head -c 16 ; echo '')
    # Setup a keycloak instance and set OPENID_ISSUER (via saving to /tmp/openid/openid_issuer)
    test/e2e/install-keycloak.sh    
    ```

7. Then execute the following command to run e2e testing:

    ```
    make e2e-ginkgo-test
    ```
    NOTE: Or you may want to use
    ```
    ginkgo -tags e2e -v test/e2e --  --ginkgo.vv --ginkgo.trace
    ```
