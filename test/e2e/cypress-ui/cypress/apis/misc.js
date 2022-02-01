/* Copyright Red Hat */

import * as constants from '../support/constants';

const headers = {
    "Content-Type": "application/json",
    Accept: "application/json"
    }

export const getLatestImageSet = () => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.apiUrl +
            "/apis/hive.openshift.io/v1/clusterimagesets",
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options).its('body').its('items').then( items =>{
        return cy.wrap(items[items.length - 1].spec.releaseImage)
    })
}

export const enableSNO = () => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.apiUrl +
            "/api/v1/namespaces/" +
            Cypress.env("ACM_NAMESPACE") +
            "/configmaps/console-config",
        headers: headers
    };
    return cy.request(options).then( resp => {
        if (resp.body.data.singleNodeOpenshift !== 'enabled') {
            options.method = 'PUT';
            resp.body.data.singleNodeOpenshift = 'enabled'
            options.body = resp.body
            return cy.request(options)
        }
    })
}