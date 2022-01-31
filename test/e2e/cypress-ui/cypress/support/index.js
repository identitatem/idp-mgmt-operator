/** *****************************************************************************
 * Licensed Materials - Property of Red Hat, Inc.
 * Copyright (c) 2021 Red Hat, Inc.
 ****************************************************************************** */

// ***********************************************************
// This example support/index.js is processed and
// loaded automatically before your test files.
//
// This is a great place to put global configuration and
// behavior that modifies Cypress.
//
// You can change the location of this file or turn off
// automatically serving support files with the
// 'supportFile' configuration option.
//
// You can read more here:
// https://on.cypress.io/configuration
// ***********************************************************

import './commands'
import './filterTests'
require('cypress-terminal-report/src/installLogsCollector')()
require('cypress-grep')()
process.env.NODE_TLS_REJECT_UNAUTHORIZED='0'
var timeoutID

Cypress.Cookies.defaults({
  preserve: ["acm-access-token-cookie", "_oauth_proxy", "XSRF-TOKEN", "_csrf"]
});

before(() => {
  // cy.clearCookies() 
})

beforeEach(function(){ 
  Cypress.Cookies.preserveOnce("acm-access-token-cookie", "_oauth_proxy", "XSRF-TOKEN", "_csrf");
}) 

afterEach(() => {
  clearTimeout(timeoutID)
})
