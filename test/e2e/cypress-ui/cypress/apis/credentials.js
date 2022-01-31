/* Copyright Red Hat */

import * as constants from '../support/constants';

const headers = {
    "Content-Type": "application/json",
    Accept: "application/json"
    }

export const getCredential = (name, namespace) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.apiUrl +
            "/api/v1/namespaces/" +
            namespace +
            "/secrets/" +
            name,
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options).then(resp => {
        if (resp.status != 200)
            return cy.wrap(resp.status)
        return cy.wrap(resp.body)
    });
}

export const getCredentials = (name, type) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.apiUrl +
            "/api/v1/secrets"+
            '?labelSelector=cluster.open-cluster-management.io/credentials' ,
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options).then(resp => {
        let items = [];
        resp.body.items.forEach(item =>{
            if (item.metadata.name.includes(name) && 
                item.metadata.labels['cluster.open-cluster-management.io/type'] === type){
                items.push(`${item.metadata.namespace}/${item.metadata.name}`)
            }
        })
        return cy.wrap(items);
    })
}

export const deleteCredentials = (name, type) => {
    getCredentials(name,type).then( creds =>{
        creds.forEach( cred => {
            let options = {
                method: "DELETE",
                url:
                    constants.apiUrl +
                    `/api/v1/namespaces/${cred.split('/').shift()}/secrets/${cred.split('/').pop()}` ,
                headers: headers,
                failOnStatusCode: false
            };
            cy.request(options).then(resp => {
                cy.wrap(resp.status)
            })
        })
    })
}

