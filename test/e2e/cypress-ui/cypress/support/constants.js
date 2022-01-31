/// <reference types="cypress" />

// API Endpoint

export const ocm_api_v1_path = "/apis/cluster.open-cluster-management.io/v1";
export const ocm_api_v1alpha_path = "/apis/addon.open-cluster-management.io/v1alpha1";
export const ocm_api_v1beta_path = "/apis/cluster.open-cluster-management.io/v1beta1";
export const hive_namespaced_api_path = "/apis/hive.openshift.io/v1/namespaces/";
export const hive_api_path = "/apis/hive.openshift.io/v1";
export const rbac_api_path = "/apis/rbac.authorization.k8s.io/v1";
export const user_api_path = "/apis/user.openshift.io/v1";

export const managedclustersets_path = "/managedclustersets"
export const clusterpools_path = "/clusterpools"
export const clusterclaims_path = "/clusterclaims"

// API URL
export const apiUrl = Cypress.config().baseUrl.replace("multicloud-console.apps", "api") + ":6443";
export const ocpUrl = Cypress.config().baseUrl.replace("multicloud-console.apps", "console-openshift-console.apps");
export const prometheusUrl = Cypress.config().baseUrl.replace("multicloud-console.apps", "prometheus-k8s-openshift-monitoring.apps");

export const supportedOCPReleasesRegex = "4.6|4.8|4.9|4.10"