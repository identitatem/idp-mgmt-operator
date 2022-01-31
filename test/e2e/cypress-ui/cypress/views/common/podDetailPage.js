/** *****************************************************************************
 * Licensed Materials - Property of Red Hat, Inc.
 * Copyright (c) 2021 Red Hat, Inc.
 ****************************************************************************** */

/// <reference types="cypress" />

export const podDetailPage = {
  whenClickOnLogsTab:() => {
    cy.get('.pf-c-nav__link').filter(':contains("Logs")').click({ timeout: 10000 })
  },
  shouldSeeLogs:(expected) => {
    cy.contains('.log-window-lines-container', expected, { timeout: 5000 })
  }
}
