/** *****************************************************************************
 * Licensed Materials - Property of Red Hat, Inc.
 * Copyright (c) 2021 Red Hat, Inc.
 ****************************************************************************** */

/// <reference types="cypress" />
describe('Sample Test Suite', {
    tags: ['@IDP', 'tag-test']
}, function () {
    before(function () {
        cy.clearOCMCookies()
    })

    after(function () {
        cy.logout();
    })

    it('Login to ACM', { tags: ['login'] }, function () {
        cy.login()
    });

    it('Empty Test', { tags: ['empty'] }, function () {
        cy.log('Nothing Done Here')
    });
})