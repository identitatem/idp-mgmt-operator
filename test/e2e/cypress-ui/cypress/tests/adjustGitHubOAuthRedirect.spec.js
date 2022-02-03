/* Copyright Red Hat */

/// <reference types="cypress" />
describe('Adjust GitHub OAuth Redirect', {
    tags: ['@IDP', 'tag-test']
}, function () {
    it('Update homepage and callback URLs for the GitHub OAuth app', { tags: [] }, function () {         // These steps need to be in a single spec since Cypress clears out cookies between specs
        // Login to GitHub
        cy.loginToGitHubOAuthApp()

        // The OAuth apps page is currently open
        cy.contains('h2', 'OAuth Apps').should('exist');

        // Open the UI for the specific OAuth app
        var oauthAppName = Cypress.env('OPTIONS_GH_OAUTH_APPNAME')
        cy.contains('a.text-bold', oauthAppName).click();
        cy.contains('h2', oauthAppName).should('exist');

        // Change OAuth Callback URL
        var oauthCallbackURL = Cypress.env('OPTIONS_GH_OAUTH_CALLBACK_URL');
        cy.get('input[name="oauth_application[callback_url]"]').clear();
        cy.get('input[name="oauth_application[callback_url]"]').type(oauthCallbackURL);

        // Change OAuth Homepage URL
        var oauthHomepageURL = Cypress.env('OPTIONS_GH_OAUTH_HOMEPAGE_URL');
        cy.get('input[name="oauth_application[url]"]').clear();
        cy.get('input[name="oauth_application[url]"]').type(oauthHomepageURL);     

        // Update
        cy.get('form.edit_oauth_application > button[type=submit]').click();
        cy.wait(1000)      
        
        // Logout of GitHub
        cy.logoutOfGitHub();
    });
})