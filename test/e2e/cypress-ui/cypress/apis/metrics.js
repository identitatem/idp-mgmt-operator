/** *****************************************************************************
 * Licensed Materials - Property of Red Hat, Inc.
 * Copyright (c) 2021 Red Hat, Inc.
 ****************************************************************************** */

/// <reference types="cypress" />

import * as constants from "../support/constants";

var headers = {
    "Content-Type": "application/json",
    Accept: "application/json"
}

export const getClusterMetrics = (clusterID) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.prometheusUrl +
            `/api/v1/query?query=acm_managed_cluster_info{managed_cluster_id='${clusterID}'}`,
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
            if (resp.status != 200)
                return cy.wrap(resp.status)
            return cy.wrap(resp.body);
    })
}

export const getMetrics = () => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "GET",
        url:
            constants.prometheusUrl +
            `/api/v1/query?query=acm_managed_cluster_info`,
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
        return cy.wrap(resp.body);
    })
}