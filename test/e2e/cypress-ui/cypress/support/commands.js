
/** *****************************************************************************
 * Licensed Materials - Property of Red Hat, Inc.
 * Copyright (c) 2021 Red Hat, Inc.
 ****************************************************************************** */

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

const authUrl = Cypress.config().baseUrl.replace("multicloud-console", "oauth-openshift");

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
  cy.getCookie('acm-access-token-cookie').should('exist').then((token) => {
    oauthIssuer(token.value).then(() => {
      cy.get(acm23xheaderSelectors.headerTools.userDropdown, { timeout: 20000 })
        .click()
        .then(
          () => cy.contains(commonElementSelectors.elements.button, acm23xheaderSelectors.headerTools.text.logout)
          .click()
        );
    })
  })
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
