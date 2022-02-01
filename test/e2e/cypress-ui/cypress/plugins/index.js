/* Copyright Red Hat */
// ***********************************************************
// This example plugins/index.js can be used to load plugins
//
// You can change the location of this file or turn off loading
// the plugins file with the 'pluginsFile' configuration option.
//
// You can read more here:
// https://on.cypress.io/plugins-guide
// ***********************************************************

/// <reference types="cypress" />

const fs = require('fs')

/**
 * @type {Cypress.PluginConfig}
 */

const { cleanReports } = require('../scripts/helpers')
// const extraVars = require('../config').getExtraVars()

module.exports = (on, config) => {

  cleanReports()

  // `on` is used to hook into various events Cypress emits
  // `config` is the resolved Cypress config

  require('cypress-terminal-report/src/installLogsPrinter')(on)
  // config.env.TEST_CONFIG = testConfig

  require('cypress-grep/src/plugin')(config)

  return config
}
