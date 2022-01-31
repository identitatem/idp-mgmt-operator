/* Copyright Red Hat */

/// <reference types="cypress" />

import { popupModal } from './popup'
import { getOpt } from '../../scripts/utils'

const SEARCH_MESSAGES_NO_RESULTS = 'No results found for the current search criteria.'


export const searchPage = {
  whenGoToSearchPage:() => cy.visit('/search'),
  whenGetResourceTableRow:(resource, name) => {
    return cy.get('tr').filter(`:contains("${name}")`)
  },
  whenDeleteResourceDetailItem:(resource, name) => {
    searchPage.whenGetResourceTableRow(resource, name).find('.pf-c-dropdown__toggle', {timeout: 2000}).click({ force: true })
    cy.get('button.pf-c-dropdown__menu-item', {timeout: 2000}).should('contain', `Delete ${resource}`).click({ timeout: 10000 }).wait(1000)
    popupModal.whenAccept()
  },
  whenDeleteNamespace: (namespace, options) => {
    var ignoreIfDoesNotExist = getOpt(options, 'ignoreIfDoesNotExist', true)
    var deleteFn = () => searchPage.whenDeleteResourceDetailItem('namespace', namespace)

    searchPage.whenGoToSearchPage()
    searchBar.whenFilterByKind('namespace')
    searchBar.whenFilterByName(namespace)
    searchPage.shouldLoadResults()
    if (ignoreIfDoesNotExist == true) {
      cy.ifNotContains('.pf-c-alert__title', SEARCH_MESSAGES_NO_RESULTS, deleteFn)
    } else {
      deleteFn()
    }
  },
  shouldFindNoResults: () => {
    cy.get('.pf-c-alert__title', { timeout: 30000 }).should('contain', SEARCH_MESSAGES_NO_RESULTS)
  },
  shouldLoadResults:() => cy.get('.pf-c-spinner', { timeout: 30000 }).should('not.exist'),
}

export const searchBar = {
  whenEnterTextInSearchBar:(property, value) => {
    cy.get('.react-tags__search-input', {timeout: 20000}).should('exist').focus().click().type(property).wait(200)
    cy.get('.react-tags', {timeout: 20000}).should('exist')
    cy.get('.react-tags__search-input', {timeout: 20000}).should('exist')
    cy.get('.react-tags__search-input', {timeout: 20000}).type(' ').wait(200)
    if (value && value !== null) {
      cy.get('.react-tags__search-input', {timeout: 20000}).type(value)
      cy.get('.react-tags__search-input', {timeout: 20000}).type(' ').wait(200)
    }
  },
  whenFilterByKind:(kind) => {
    searchBar.whenEnterTextInSearchBar('kind', kind)
  },
  whenFilterByName:(name) => {
    searchBar.whenEnterTextInSearchBar('name', name)
  },
}
