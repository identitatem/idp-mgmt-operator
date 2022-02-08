/* Copyright Red Hat */

/// <reference types="cypress" />
describe('Sample Test Suite to test creation and deletion of a GitHub OAuth app', {
    tags: ['@IDP', 'tag-github-oauth-test']
}, function () {
    before(function () {
        cy.createGitHubOAuthApp()
    })

    after(function () {
        cy.deleteGitHubOAuthApp();
    })

    it('Empty Test', { tags: ['empty'] }, function () {
        cy.log('OAuth app name: ', Cypress.env('OPTIONS_GH_OAUTH_APP_NAME'));
        cy.log('OAuth app client ID', Cypress.env('OPTIONS_GH_OAUTH_CLIENT_ID'));
        cy.log('OAuth app client Secret', Cypress.env('OPTIONS_GH_OAUTH_CLIENT_SECRET'));
        cy.log('Nothing Done Here');
    });
})
