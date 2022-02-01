/* Copyright Red Hat */

/// <reference types="cypress" />
import * as constants from "../support/constants";

var headers = {
    "Content-Type": "application/json",
    Accept: "application/json"
}

export const getUser = (user) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method:"GET",
        url:
            constants.apiUrl +
            constants.user_api_path +
            "/users/" +
            user, 
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
            return resp
        });
}

export const getClusterRole = (clusterRole) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method:"GET",
        url:
            constants.apiUrl +
            constants.rbac_api_path +
            "/clusterroles/" +
            clusterRole, 
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
            return resp
        });
}

export const getClusterRolebinding = (clusterRoleBinding) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method:"GET",
        url:
            constants.apiUrl +
            constants.rbac_api_path +
            "/clusterrolebindings/" +
            clusterRoleBinding, 
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
            return resp
        });
}

export const createClusterRolebinding = (body) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "POST",
        url:
            constants.apiUrl +
            constants.rbac_api_path +
            "/clusterrolebindings",
        headers: headers,
        body: body
    };
    return cy.request(options)
        .then(resp => {
            return resp
        });
}

export const deleteClusterRolebinding = (clusterRoleBinding) => {
    headers.Authorization = `Bearer ${Cypress.env("token")}`;
    let options = {
        method: "DELETE",
        url:
            constants.apiUrl +
            constants.rbac_api_path +
            "/clusterrolebindings/" +
            clusterRoleBinding,
        headers: headers,
        failOnStatusCode: false
    };
    return cy.request(options)
        .then(resp => {
            return resp
        });
}