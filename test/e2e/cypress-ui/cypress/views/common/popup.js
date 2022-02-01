/* Copyright Red Hat */

/// <reference types="cypress" />

export const popupModal = {
  whenAccept:() => {
    popupModal.shouldExist()
    cy.get('button.pf-m-danger', { timeout: 30000 }).click({ timeout: 15000 }).wait(500)
    cy.get('.pf-c-alert pf-m-inline pf-m-danger', { timeout: 10000 }).should('not.exist')
    popupModal.shouldNotExist()
  },
  shouldNotExist:() => cy.get('button.pf-m-danger', { timeout: 30000 }).should('not.exist'),
  shouldExist:() => cy.get('button.pf-m-danger', { timeout: 30000 }).should('exist')
}
