/** *****************************************************************************
 * Licensed Materials - Property of Red Hat, Inc.
 * Copyright (c) 2021 Red Hat, Inc.
 ****************************************************************************** */

/// <reference types="cypress" />

import { welcomePage } from './welcome'

const typeDelay = 1

export const resourcePage = {
  whenGoToResourcePage: () =>  {
    welcomePage.whenGoToWelcomePage()
    cy.get('#acm-create-resource', { timeout: 20000 }).click()
  },
  whenSelectTargetCluster: (clusterName) => {
    cy.get('#create-resource-select', { timeout: 20000 }).click()
    cy.get('.bx--checkbox-wrapper input[name="' + clusterName + '"]').parent().click()
  },
  whenCreateNamespace: (namespace) => {
    // WORKAROUND: delays are needed because this cypress issue https://github.com/cypress-io/cypress/issues/5480
    cy.get('.react-monaco-editor-container').click().focused().type(Cypress.platform !== 'darwin' ? '{ctrl}a' : '{meta}a')
      .type('{enter}apiVersion: v1{enter}', { delay: typeDelay })
      .type('kind: Namespace{enter}', { delay: typeDelay })
      .type('metadata:{enter}', { delay: typeDelay })
      .type('  name: ' + namespace + '{enter}', { delay: typeDelay });
    resourcePage.shouldCreateResource();
  },
  whenCreateDeployment: (namespace, name, image) => {
    // WORKAROUND: delays are needed because this cypress issue https://github.com/cypress-io/cypress/issues/5480
    cy.get('.react-monaco-editor-container').click().focused().type(Cypress.platform !== 'darwin' ? '{ctrl}a' : '{meta}a')
      .type('{enter}apiVersion: apps/v1{enter}', { delay: typeDelay })
      .type('kind: Deployment{enter}', { delay: typeDelay })
      .type('metadata:{enter}', { delay: typeDelay })
      .type('  name: ' + name + '{enter}{backspace}', { delay: typeDelay })
      .type('  namespace: ' + namespace + '{enter}{backspace}', { delay: typeDelay })
      .type('spec:{enter}', { delay: typeDelay })
      .type('  replicas: 1{enter}{backspace}', { delay: typeDelay })
      .type('  selector:{enter}{backspace}', { delay: typeDelay })
      .type('    matchLabels:{enter}{backspace}{backspace}', { delay: typeDelay })
      .type('      app: ' + name + '{enter}{backspace}{backspace}{backspace}', { delay: typeDelay })
      .type('  template:{enter}{backspace}', { delay: typeDelay })
      .type('    metadata:{enter}{backspace}{backspace}', { delay: typeDelay })
      .type('      labels:{enter}{backspace}{backspace}{backspace}', { delay: typeDelay })
      .type('        app: ' + name + '{enter}{backspace}{backspace}{backspace}{backspace}', { delay: typeDelay })
      .type('    spec:{enter}{backspace}{backspace}', { delay: typeDelay })
      .type('      containers:{enter}{backspace}{backspace}{backspace}', { delay: typeDelay })
      .type('        - name: ' + name + '{enter}{backspace}{backspace}{backspace}{backspace}', { delay: typeDelay })
      .type('          image: ' + image + '{enter}', { delay: typeDelay });
    resourcePage.shouldCreateResource();
  },
  shouldCreateResource: () => {
    cy.get('.bx--btn--primary').click();
    cy.get('.bx--inline-notification', { timeout: 30000 }).should('not.exist');
    cy.get('.react-monaco-editor-container', { timeout: 30000 }).should('not.exist');
  }
}
