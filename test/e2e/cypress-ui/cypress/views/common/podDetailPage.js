/* Copyright Red Hat */

/// <reference types="cypress" />

export const podDetailPage = {
  whenClickOnLogsTab:() => {
    cy.get('.pf-c-nav__link').filter(':contains("Logs")').click({ timeout: 10000 })
  },
  shouldSeeLogs:(expected) => {
    cy.contains('.log-window-lines-container', expected, { timeout: 5000 })
  }
}
