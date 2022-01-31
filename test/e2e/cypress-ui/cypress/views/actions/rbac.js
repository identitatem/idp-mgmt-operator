/* Copyright Red Hat */

/// <reference types="cypress" />

import * as rbac from '../../apis/rbac';

export const rbacActions = {
  shouldHaveClusterRolebindingForUser: (clusterRoleBinding, clusterRole, user) => {
    rbac.getClusterRolebinding(clusterRoleBinding).then((resp) => {
      if ( !resp.isOkStatusCode ) {
        let request_body=`
  {
  "kind": "ClusterRoleBinding",
  "apiVersion": "rbac.authorization.k8s.io/v1",
  "metadata": {
    "name": "${clusterRoleBinding}"
  },
  "subjects": [
    {
      "kind": "User",
      "apiGroup": "rbac.authorization.k8s.io",
      "name": "${user}"
    }
  ],
  "roleRef": {
      "apiGroup": "rbac.authorization.k8s.io",
      "kind": "ClusterRole",
      "name": "${clusterRole}"
    }
  }`
          rbac.createClusterRolebinding(request_body)
        }
    })
  },

  deleteClusterRolebinding: (clusterRoleBinding) => {
    rbac.deleteClusterRolebinding(clusterRoleBinding).then((resp) => expect(resp.isOkStatusCode))
  }
}