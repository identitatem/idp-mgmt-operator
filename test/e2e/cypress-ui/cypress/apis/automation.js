/* Copyright Red Hat */

import * as constants from '../support/constants';
import { genericFunctions } from '../support/genericFunctions'

const headers = {
    "Content-Type": "application/json",
    Accept: "application/json"
    }

export const getClusterCurator = (name, namespace) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.apiUrl +
            `/apis/cluster.open-cluster-management.io/v1beta1/namespaces/${namespace}/clustercurators/${name}`,
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options).then(resp => {
        return cy.wrap(resp.status);
    })
}

export const deleteClusterCurators = (namespace) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let url = constants.apiUrl +
            `/apis/cluster.open-cluster-management.io/v1beta1/namespaces/${namespace}/clustercurators`;
    let options = {
        method: "GET",
        url: url,
        headers: headers,
        failOnStatusCode: false
    };
    cy.request(options).then(resp => {
        resp.body.items.forEach( item => {
            options.method = "DELETE";
            options.url = `${url}/${item.metadata.name}`
            cy.request(options).then( resp => {
                return cy.wrap(resp.status);
            })
        })
    })
}

export const getAnsibleJobResult = (namespace, stage) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let url = constants.apiUrl +
            `/apis/tower.ansible.com/v1alpha1/namespaces/${namespace}/ansiblejobs`;
    let options = {
        method: "GET",
        url: url,
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options).its('body').then( body => {
        let status = 'Pending'
        if (body.items.length < 1 ) {
            return status
        }
        else {
            body.items.forEach( item => {
                if (item.metadata.name.includes(stage)) {
                    cy.wrap(item.metadata.name)
                    status = item.status.ansibleJobResult != null ? item.status.ansibleJobResult.status : 'in progress'
                }
            })
            return cy.wrap(status) 
        }
    })
}

export const checkAnsibleJobResult = (namespace, stage) => {
    genericFunctions.recurse(
        () => getAnsibleJobResult(namespace, stage),
        (status) => ['failed','successful'].includes(status),
    ) 
}