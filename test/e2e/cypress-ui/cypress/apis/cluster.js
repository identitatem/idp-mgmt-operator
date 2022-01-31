/** *****************************************************************************
 * Licensed Materials - Property of Red Hat, Inc.
 * Copyright (c) 2021 Red Hat, Inc.
 ****************************************************************************** */

/// <reference types="cypress" />
import * as constants from "../support/constants";
import { genericFunctions } from '../support/genericFunctions'

var headers = {
    "Content-Type": "application/json",
    Accept: "application/json"
}

export const getClusterSet = (clusterSet) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.apiUrl +
            constants.ocm_api_v1beta_path + 
            constants.managedclustersets_path + 
            '/' + clusterSet,
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
            return resp
        });
}

export const getClusterSets = () => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.apiUrl +
            constants.ocm_api_v1beta_path + 
            constants.managedclustersets_path,
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
            return resp
        });
}

export const getClusterPool = (clusterPoolName, namespace) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.apiUrl +
            constants.hive_api_path +
            '/namespaces/' +
            namespace +
            constants.clusterpools_path +
            '/' + clusterPoolName,
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
            return resp
        });
}

export const getClusterPools = () => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.apiUrl +
            constants.hive_api_path + 
            constants.clusterpools_path,
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
            return resp
        });
}

export const getClusterClaim = (clusterClaimName, namespace) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.apiUrl +
            constants.hive_api_path +
            '/namespaces/' +
            namespace +
            constants.clusterclaims_path +
            '/' + clusterClaimName,
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
            return resp
        });
}

export const getClusterClaims = () => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.apiUrl +
            constants.hive_api_path + 
            constants.clusterclaims_path,
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
            return resp
        });
}

export const getManagedCluster = (clusterName) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.apiUrl +
            constants.ocm_api_v1_path +
            "/managedclusters/" +
            clusterName, 
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
            return resp
        });
}

export const getManagedClusters = (labels) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let url = constants.apiUrl + constants.ocm_api_v1_path + "/managedclusters"
    if (labels) url = url + `?labelSelector=${labels}`
    let options = {
        method: "GET",
        url: url,
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
            return resp
        });
}

export const getManagedClusterDetails = (clusterName) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.apiUrl +
            constants.ocm_api_v1_path +
            "/managedclusters/" +
            clusterName, 
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
            if (resp.status != 200)
                return cy.wrap(resp.status)
            return cy.wrap(resp.body)
        });
}

export const getManagedClusterInfo = (clusterName) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.apiUrl +
            '/apis/internal.open-cluster-management.io/v1beta1/namespaces/' + clusterName +
            "/managedclusterinfos/" + clusterName,
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
            if (resp.status != 200)
                return cy.wrap(resp.status)
            return cy.wrap(resp.body)
        });
}


export const createManagedCluster = (body) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "POST",
        url:
            constants.apiUrl +
            constants.ocm_api_v1_path +
            "/managedclusters",
        headers: headers,
        body: body
    };
    return cy.request(options)
        .then(resp => {
            return resp
        });
}

export const deleteManagedCluster = (clusterName) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "DELETE",
        url:
            constants.apiUrl +
            constants.ocm_api_v1_path +
            "/managedclusters/" +
            clusterName,
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
            return resp
        });
}

export const getClusterDeployment = (clusterName) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.apiUrl +
            constants.hive_namespaced_api_path +
            clusterName +
            "/clusterdeployments/" +
            clusterName,
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
            if (resp.status != 200)
                return cy.wrap(resp.status)
            return cy.wrap(resp.body)
        });
}

export const getManagedClusterAddons = (clusterName) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.apiUrl +
            constants.ocm_api_v1alpha_path +
            "/namespaces/" +
            clusterName +
            "/managedclusteraddons",
        headers: headers
    };
    return cy.request(options)
        .then(resp => {
            return resp
        });
}

export const getAllSNOClusters = () => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let clusters = [];
    let options = {
        method: "GET",
        url:
            constants.apiUrl +
            constants.ocm_api_v1_path +
            "/managedclusters", 
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options).then(resp => {
        if (resp.body.items.length < 1) {
            cy.log("No clusters found")
            return null
        }
        else {
            resp.body.items.forEach( item => {
                if (item.metadata.name.includes('-sno-')) {
                    clusters.push(item.metadata.name)
                }
                return cy.wrap(clusters)
            });
        }
    })
}

export const checkClusterDeployment = (clusterName) => {
    // timeout = 90 mins, polling every 5 minutes
    let interval = 300 * 1000; //5 mins
    function getProvisionStatus (name) {
        return getClusterDeployment(name).then( cd => {
            let status = cd.spec.installed
            cd.status.conditions.forEach( condition => {
                if (condition.type === 'ProvisionFailed' ){
                    status = condition.status === 'True' ? condition.status : status
                }               
            })
            return status
        })
    }
    genericFunctions.recurse(
        () => getProvisionStatus(clusterName),
        (status) => Boolean(status),
        16,
        interval)
}

export const checkManagedClusterInfoStatus = (clusterName) => {
    // timeout = 15 mins, polling every 30s
    let interval = 30 * 1000;
    function getClusterInfoStatus (name) {
        return getManagedClusterInfo(name).then( clusterInfo => {
            let status = 'false'
            clusterInfo.status.conditions.forEach(condition => {
                if (condition.type == 'ManagedClusterInfoSynced') {
                    status = condition.status === 'True' ? condition.status : status
                }
            })
            return status
        })
    }
    genericFunctions.recurse(
        () => getClusterInfoStatus(clusterName),
        (status) => Boolean(status),
        30,
        interval) 
}

export const checkClusterDeploymentDeleted = (clusterName) => {
    // timeout = 30 mins, polling every 60s
    let interval = 60 * 1000; 
    genericFunctions.recurse(
        () => getClusterDeployment(clusterName),
        (status) => status == 404,
        30,
        interval) 
}

export const checkManagedClusterInfoDeleted = (clusterName) => {
    // timeout = 15 mins, polling every 30s
    let interval = 30 * 1000; 
    genericFunctions.recurse(
        () => getManagedClusterInfo(clusterName),
        (status) => status == 404,
        30,
        interval) 
}
