/* Copyright Red Hat */

/// <reference types="cypress" />

export const pageLoader = {
  shouldExist: () => cy.get('.content-spinner', { timeout: 20000 }).should('exist')  ,
  shouldNotExist: () => cy.get('.content-spinner', { timeout: 20000 }).should('not.exist')
}

export const resourceTable = {
  shouldExist: () => cy.get('pf-l-stack__item', { timeout: 20000 }).should('exist'),
  shouldNotExist: () => cy.get('pf-l-stack__item', { timeout: 20000 }).should('not.exist'),
  rowShouldExist: function (name) {
    this.searchTable(name)
    cy.get(`tr[data-row-name="${name}"]`, { timeout: 30000 }).should('exist')
  },
  rowShouldNotExist: function (name, timeout, disableSearch) {
    !disableSearch && this.searchTable(name)
    cy.get(`tr[data-row-name="${name}"]`, { timeout: timeout || 30000 }).should('not.exist')
  },
  openRowMenu: name => {cy.get(`tr[data-row-name="${name}"] .bx--overflow-menu`).click(), cy.get('div[triggerclassname="overflow-permission-tooltip"]', { timeout: 2000 }).should('not.exist')},
  menuClickEdit: () => cy.get('button[data-table-action="table.actions.connection.edit"]').click(),
  menuClickEditLabels: () => cy.get('button[data-table-action="table.actions.cluster.edit.labels"]').click(),
  menuClickDelete: () => cy.get('button[data-table-action="table.actions.connection.delete"]').click(),
  menuClickDeleteType: (type) => cy.get(`button[data-table-action="table.actions.${type}.delete"]`).click(),
  menuClickDestroy: () => cy.get('button[data-table-action="table.actions.cluster.destroy"]').click(),
  menuClickDetach: () => cy.get('button[data-table-action="table.actions.cluster.detach"]').click(),
  clearSearch: () => cy.get('.pf-c-search-input__text-input').clear(),
  searchTable: function (name) {
    cy.get('#pf-c-page').then(page => {
      if (page.find('.pf-c-search-input__text-input', { timeout: 15000 }).length > 0) {
        this.clearSearch()
        cy.get('.pf-c-search-input__text-input').paste(name)
      }
    })
  },
  rowShouldToggle: function(name) {
    cy.get('#pf-c-page').then(page => {
      if (page.find('.pf-c-search-input__text-input', { timeout: 15000 }).length > 0) {
        cy.get('.pf-c-search-input__text-input').clear({force: true})
        cy.get('.pf-c-search-input__text-input').type(name)
      }
    })
    cy.get(`tr[data-row-name="${name}"]`).get('input[name="tableSelectRow"]').click({force: true})
  },
}

export const secondaryHeader = {
  clickPrimary: () => cy.get('.secondary-header-actions-primary').click(),
  clickSecondary: () => cy.get('.secondary-header-actions-secondary').click()
}

export const noResource = {
  shouldExist: () => cy.get('.pf-c-empty-state__content').should('exist'),
  shouldNotExist: () => cy.get('.pf-c-empty-state__content').should('not.exist')
}

export const modal = {
  shouldBeOpen: () => cy.get('.bx--modal', { timeout: 10000 }).should('exist'),
  shouldBeClosed: () => cy.get('.bx--modal', { timeout: 10000 }).should('not.exist'),
  clickDanger: () => cy.get('.bx--modal .bx--btn--danger--primary').click(),
  clickPrimary: () => cy.get('.bx--modal .bx--btn--primary').click({ timeout: 20000 }),
  clickSecondary: () => cy.get('.bx--modal-footer .bx--btn--secondary').click(),
  confirmAction: text => cy.get('#confirm-action').type(text)
}

export const notification = {
  shouldExist: (type, timeout) => cy.get(`.bx--inline-notification[kind="${type}"]`, { timeout: timeout || 10000 }).should('exist'),
  shouldNotExist: (type, timeout) => cy.get(`.bx--inline-notification[kind="${type}"]`, { timeout: timeout || 10000 }).should('not.exist'),
  messageExist: (message) => cy.get('.bx--inline-notification__subtitle').contains(message),
  scrollIntoView: () => cy.get('.bx--inline-notification', { timeout: 10000 }).scrollIntoView()
}

export const resourcePage = {
  shouldLoad: () => {
    cy.get('.pf-c-page').then(page => {
      if (page.find('.resource-table').length) {
        resourceTable.shouldExist()
      } else {
        noResource.shouldExist()
      }
    })
  }
}

export const codeSnippet = {
  shouldExist: () => cy.get('.bx--snippet', { timeout: 10000 }).should('exist'),
  shouldNotExist: () => cy.get('.bx--snippet', { timeout: 10000 }).should('not.exist'),
  clickCopy: () => cy.get('.bx--snippet-button', { timeout: 10000 }).click()
}

export const doTestIfExists = (name, test) => {
  pageLoader.shouldNotExist()
  resourceTable.searchTable(name)
  cy.get('#page').then(page => {
    if (page.find(`tr[data-row-name="${name}"]`).length > 0) {
      test()
    } else {
      cy.log(`${name} not found, skipping...`)
    }
  })
}

export const releaseImageSet = {
  selectRandomReleaseImage: function() {
    cy.get('.bx--list-box__menu-item').invoke('text').then((items) => {
      const res = items.split('quay.io/').filter(n => n)
      const release43 = res.filter(element => element.includes('4.3')).sort().reverse().shift()
      const release44 = res.filter(element => element.includes('4.4')).sort().reverse().shift()
      const release45 = res.filter(element => element.includes('4.5')).sort().reverse().shift()
      const release46 = res.filter(element => element.includes('4.6')).sort().reverse().shift()
      const release = [ release43, release44, release45, release46 ]
      const releaseImages = release.filter(n => n)
      const item = releaseImages[Math.floor(Math.random() * releaseImages.length)]
      cy.log('release', item)
      cy.contains(item, { timeout: 30 * 1000 }).click()
    })
  }
}

export const recurse = (commandFn, checkFn, limit = 10, interval) => {
  if (limit < 0) {
      throw new Error('Recursion limit reached')
  }
  cy.log(`**${commandFn}** remaining attempts **${limit}**`)
  return commandFn().then( (x) => {
    // cy.log(`**${checkFn(x)}**`)
      if (checkFn(x)) {
        cy.log( `**${commandFn}** completed` )
        return
      }
      cy.wait(interval);
      recurse(commandFn, checkFn, limit - 1, interval)
  })
}