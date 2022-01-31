/* Copyright Red Hat */

/// <reference types="cypress" />

import { popupModal } from './popup'

export const deploymentDetailPage = {
  whenScaleReplicasTo:(replicas) => {
    cy.get('button.pf-m-primary').click({ timeout: 20000 }).wait(1000)
    cy.get('.react-monaco-editor-container').click().type(Cypress.platform !== 'darwin' ? '{ctrl}f' : '{meta}f')
      .get('.find-widget .monaco-inputbox textarea:first').focus().click().type('replicas: 1')
    cy.get('.react-monaco-editor-container .view-line > span')
      .filter(':contains("replicas:")').contains('1').parent()
      .find('span:last').click().focused().type('{del}' + replicas)
    cy.get('button.pf-m-primary').filter(':contains("Save")').click({ timeout: 10000, force: true })
    // popupModal.whenAccept() // FIXME: Code bug? We don't have a confirmation modal.
  }
}
