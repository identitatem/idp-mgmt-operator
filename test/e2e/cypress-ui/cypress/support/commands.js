/* Copyright Red Hat */

// ***********************************************
// This example commands.js shows you how to
// create various custom commands and overwrite
// existing commands.
//
// For more comprehensive examples of custom
// commands please read more here:
// https://on.cypress.io/custom-commands
// ***********************************************

import 'cypress-wait-until'
import 'cypress-fill-command'

import { acm23xheaderMethods, acm23xheaderSelectors } from '../views/header'

import { commonElementSelectors } from '../views/common/commonSelectors'
import { oauthIssuer } from '../views/common/welcome'

const authUrl = Cypress.config().baseUrl;

Cypress.Commands.add('login', (OPTIONS_HUB_USER, OPTIONS_HUB_PASSWORD, OC_IDP) => {
  var user = OPTIONS_HUB_USER || Cypress.env('OPTIONS_HUB_USER');
  var password = OPTIONS_HUB_PASSWORD || Cypress.env('OPTIONS_HUB_PASSWORD');
  var idp = OC_IDP || Cypress.env('OC_IDP');
  // Starting the tests from the overview page (Welcome page needs to be updated)
  cy.visit('/multicloud/welcome', { failOnStatusCode: false })
  cy.get('body').then(body => {
    // Check if logged in

    if (body.find(acm23xheaderSelectors.mainHeader).length === 0) {
      if (body.text().includes('Log in with OpenShift')) {
        cy.contains('Log in with OpenShift', { timeout: 15000 }).click()
      }
      // cy.contains('Log in with OpenShift',{timeout: 15000}).click()
      // Check if identity providers are configured
      if (body.find('form').length === 0)
        cy.contains(idp).click()
      // cy.contains(idp).click()
      cy.get('#inputUsername', { timeout: 20000 }).click().focused().type(user)
      cy.get('#inputPassword', { timeout: 20000 }).click().focused().type(password)
      cy.get('button[type="submit"]', { timeout: 20000 }).click()
      cy.get(acm23xheaderSelectors.mainHeader, { timeout: 30000 }).should('exist')
    }
  })
})

Cypress.Commands.add('checkCondition', (selector, condition, action) => {
  return cy.get('body').then($body => {
    var $elem = $body.find(selector)
    var result = condition($elem)
    if (result == true && action) {
      return action($elem)
    }

    return cy.wrap(result)
  })
})

Cypress.Commands.add('ifNotContains', (selector, text, action) => {
  return cy.checkCondition(selector, ($elem) => !$elem || !$elem.text().includes(text), action)
})

Cypress.Commands.add('logout', () => {
  //TODO Revisit this code for ACM 2.5 and see why it was not working.  Once fixed we can re-enable
  //cy.getCookie('acm-access-token-cookie').should('exist').then((token) => {
  //  oauthIssuer(token.value).then(() => {
  //    cy.get(acm23xheaderSelectors.headerTools.userDropdown, { timeout: 20000 })
  //      .click()
  //      .then(
  //        () => cy.contains(commonElementSelectors.elements.button, acm23xheaderSelectors.headerTools.text.logout)
  //        .click()
  //      );
  //  })
  //})
})

Cypress.Commands.add('paste', {
  prevSubject: true,
  element: true
}, ($element, text) => {

  const subString = text.substr(0, text.length - 1)
  const lastChar = text.slice(-1)

  $element.text(subString)
  $element.val(subString)
  cy.get($element).type(lastChar).then(() => {
    if ($element.val() !== text) // first usage only setStates the last character for some reason
      cy.get($element).clear().type(text)
  })
})

Cypress.Commands.add("acquireToken", () => {
  cy
    .request({
      method: "HEAD",
      url:
        authUrl +
        "/oauth/authorize?response_type=token&client_id=openshift-challenging-client",
      followRedirect: false,
      headers: {
        "X-CSRF-Token": 1
      },
      auth: {
        username: Cypress.env("OPTIONS_HUB_USER"),
        password: Cypress.env("OPTIONS_HUB_PASSWORD")
      }
    })
    .then(resp => {
      return cy.wrap(resp.headers.location.match(/access_token=([^&]+)/)[1]);
    });
}),

Cypress.Commands.add("setAPIToken", () => {
  cy.acquireToken().then(token => {
    Cypress.env("token", token);
  });
})

Cypress.Commands.add("ocExtractClusterKubeconfig", (clusterMame) => {
  cy.exec(`oc extract -n ${clusterMame} secret/$(oc get cd -n ${clusterMame} ${clusterMame} ` +
    `-o jsonpath='{.spec.clusterMetadata.adminKubeconfigSecretRef.name}') --keys=kubeconfig --to=- > cypress/assets/${clusterMame}.kubeconfig`)
})

Cypress.Commands.add("ocLogin", (token, server) => {
  cy.exec('oc login --token ' + token + ' --server ' + server)
})

Cypress.Commands.add("clearOCMCookies", () => {
  cy.clearCookie("acm-access-token-cookie");
  cy.clearCookie("_oauth_proxy");
  cy.clearCookie("XSRF-TOKEN");
  cy.clearCookie("_csrf");
})

Cypress.Commands.add(
  'runCmd',
  (
    cmd,
    setAlias = false,
    failOnNonZeroExit = true,
    timeout = Cypress.config('defaultCommandTimeout'),
  ) => {
    cy.log(`Executing command: ${cmd}`);
    if (setAlias) {
      cy.exec(`${cmd}`, { timeout: timeout, failOnNonZeroExit: failOnNonZeroExit }).then((obj) => {
        cy.wrap(obj).as('runCmdAlias');
      });
    } else {
      cy.exec(`${cmd}`, { timeout: timeout, failOnNonZeroExit: failOnNonZeroExit });
    }
  }
)

// Cypress command to sign into GitHub OAuth Applications page
Cypress.Commands.add('loginToGitHubOAuthAppsPage', (OPTIONS_GH_OAUTH_APPS_URL, OPTIONS_GH_USER, OPTIONS_GH_PASSWORD) => {
  Cypress.Cookies.debug(true)
  
  var oauthAppsURL = OPTIONS_GH_OAUTH_APPS_URL || Cypress.env('OPTIONS_GH_OAUTH_APPS_URL');
  var username = OPTIONS_GH_USER || Cypress.env('OPTIONS_GH_USER');
  var password = OPTIONS_GH_PASSWORD || Cypress.env('OPTIONS_GH_PASSWORD');
  
  cy.visit(oauthAppsURL, { failOnStatusCode: false });

  cy.get('body').then(body => {
    if (body.find(`header > div > details > summary > img[alt="@${username}"]`).length == 0) {
      cy.log('Not signed in');

      // Sign in
      cy.get('input[name=login]').type(username);
      cy.get('input[name=password]').type(`${password}{enter}`, { log:false });

      // Should be signed in
      cy.get(`header > div > details > summary > img[alt="@${username}"]`, { timeout: 10000 }).should('exist');
    } else {
      cy.log('Signed in');
    }
  })
})

// Cypress command to logout of GitHub
Cypress.Commands.add('logoutOfGitHub', (OPTIONS_GH_OAUTH_APP_URL, OPTIONS_GH_USER, OPTIONS_GH_PASSWORD) => {
  cy.get('body').then(body => {
    if (body.find('header.header-logged-out').length > 0) {
      cy.log('Not signed in');
    } else {
      cy.log('Signed in');

      cy.wait(500);

      // Sign out
      cy.get('header > div > details > summary[aria-label="View profile and more"]').then((selectedElement) => {
        debugger
          selectedElement.trigger("click");
      })
      cy.contains("form.logout-form > button", "Sign out").click();

      cy.contains("header", "Sign in").should('exist');

      cy.log('Signed out');
    }
  })
})

// Cypress command to create a GitHub OAuth app
Cypress.Commands.add('createGitHubOAuthApp', () => {
    // Login to GitHub
    cy.loginToGitHubOAuthAppsPage()

    // The OAuth apps page is currently open
    cy.contains('h2', 'OAuth Apps').should('exist');

    // Known exception returned on the New OAuth app form, bypass it
    cy.on('uncaught:exception', (err, runnable) => {
        // we expect an error with message 'n.subscriptions is not iterable'
        // and don't want to fail the test so we return false
        if (err.message.includes('n.subscriptions is not iterable') || err.message.includes('o.subscriptions is not iterable')) {
          return false
        }
        // we still want to ensure there are no other unexpected
        // errors, so we let them fail the test
    });

    // Create an OAuth app
    cy.contains('a.btn', 'New OAuth App').click();
    cy.get('h2[id=oauth_application_form_label]').should('exist');

    var oAuthNameSuffix = Math.floor(1000 + Math.random() * 9000);
    var oAuthAppName = "test-e2e-" + oAuthNameSuffix;
    cy.get('input[name="oauth_application[name]"]').type(oAuthAppName);

    // Set OAuth app name (to be used later when the OAuth app is destroyed)
    Cypress.env('OPTIONS_GH_OAUTH_APP_NAME', oAuthAppName);

    // Enter OAuth Callback URL
    var oauthCallbackURL = Cypress.env('OPTIONS_GH_OAUTH_CALLBACK_URL');
    cy.get('input[name="oauth_application[callback_url]"]').type(oauthCallbackURL);

    // Enter OAuth Homepage URL
    var oauthHomepageURL = Cypress.env('OPTIONS_GH_OAUTH_HOMEPAGE_URL');
    cy.get('input[name="oauth_application[url]"]').type(oauthHomepageURL);     

    // Submit
    cy.get('form.new_oauth_application > p > button[type=submit]').click();
    
    cy.contains('h2', oAuthAppName).should('exist');

    // Create a Client Secret
    cy.contains('form > input.btn[type=submit]', 'Generate a new client secret').click();

    // Copy client secret and client ID into environment variables to be used in Cypress tests for GitHub IDP
    cy.get('code[id=new-oauth-token]')
      .invoke('text')
      .then((clientSecret) => {
        Cypress.env('OPTIONS_GH_OAUTH_CLIENT_SECRET', clientSecret);
      });
    cy.get('code.f4')
      .invoke('text')
      .then((clientID) => {
        Cypress.env('OPTIONS_GH_OAUTH_CLIENT_ID', clientID);
      });
    
    // Logout of GitHub
    cy.logoutOfGitHub();
})

// Cypress command to delete a GitHub OAuth app after use
Cypress.Commands.add('deleteGitHubOAuthApp', () => {
    // Login to GitHub
    cy.loginToGitHubOAuthAppsPage()

    // The OAuth apps page is currently open
    cy.contains('h2', 'OAuth Apps').should('exist');

    // Open the UI for the specific OAuth app
    var oAuthAppName = Cypress.env('OPTIONS_GH_OAUTH_APP_NAME');  // Get the OAuth app previously created
    cy.contains('a.text-bold', oAuthAppName).click();
    cy.contains('h2', oAuthAppName).should('exist');

    // Delete application
    cy.contains('a.js-selected-navigation-item', 'Advanced').click();
    cy.contains('summary.btn-danger', 'Delete application').should('be.visible');
    cy.contains('summary.btn-danger', 'Delete application').click();
    cy.contains('details-dialog > div > form > button[type=submit]', 'Delete OAuth application').should('be.visible');
    cy.contains('details-dialog > div > form > button[type=submit]', 'Delete OAuth application').click();

    // The OAuth apps page is open
    cy.contains('h2', 'OAuth Apps').should('exist');
    
    // Logout of GitHub
    cy.logoutOfGitHub();
})