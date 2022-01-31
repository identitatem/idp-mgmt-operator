/* Copyright Red Hat */
/// <reference types="cypress"/>

export const commonElementSelectors = {
  elements: {
    button: 'button',
    a: 'a',
    h1: 'h1',
    tr: 'tr',
    input: 'input',
    class: 'class',
    dialog: 'div[role="dialog"]',
    delete: '#delete',
    actionsButton: '#toggle-id',
    checkAll: '[name="check-all"]',
    selectMenuItem: ".pf-c-select__menu-item",
    pageClassKey: ".pf-c-page",
    dialogPageCover: ".pf-l-bullseye",
    buttonClassKey: ".pf-c-empty-state__primary > .pf-c-button",
    disabledButton: "pf-m-aria-disabled",
    pageSearch: 'input[aria-label="Search input"]',
    resetSearch: 'button[aria-label="Reset"]',
    dropDownToggleButton: 'button[aria-label="Options menu"]',
    xButton: 'button[aria-label="Close"]',
    lable: 'div[class="pf-c-form__group-label"]',
    currentTabClass: 'a[class="pf-c-nav__link pf-m-current"]',
    tabClass: 'a[class="pf-c-nav__link"]',
    radioInput: 'div[class="pf-c-radio"]',
    table: 'table',
    firstRowIndex: '1'
  },
  elementsText: {
    next: 'Next',
    add: 'Add',
    create: 'Create',
    cancel: 'Cancel',
    import: 'Import',
    save: 'Save',
    delete: 'Delete'
  },
  alerts: {
    dangerAlert: '[aria-label="Danger Alert"]'
  }


}

export const commonPageMethods = {
  resourceTable: {
    shouldExist: () => cy.get(commonElementSelectors.elements.table, { timeout: 20000 }).should('exist'),
    shouldNotExist: () => cy.get(commonElementSelectors.elements.table, { timeout: 20000 }).should('not.exist'),
    rowShouldExist: function (name) {
      this.searchTable(name)
      cy.get(`tr > td a`, { timeout: 30000 }).contains(name, { timeout: 30000 }).should('exist')
    },
    rowShouldNotExist: function (name, timeout, disableSearch) {
      !disableSearch && this.searchTable(name)
      cy.get(`tr[data-ouia-component-id="${name}"]`, { timeout: timeout || 30000 }).should('not.exist')
    },

    getRowByIndex: function (rowIndex) {
      return cy.get(commonElementSelectors.elements.table, { timeout: 1000 })
      .find(commonElementSelectors.elements.tr).eq(rowIndex)
    },

    checkIfRowExistsByName: function (name) {
      this.searchTable(name)
      return cy.wait(500).get(commonElementSelectors.elements.table).then($table => {
        return cy.wrap($table.text().includes(name))
      })
    },

    // openRowMenu: name => { cy.get(`tr[data-ouia-component-id="${name}"]`).children('td[class="pf-m-fit-content"]').children().find('button').click(), cy.get('div[triggerclassname="overflow-permission-tooltip"]', { timeout: 2000 }).should('not.exist') },
    openRowMenu: (name) => { cy.log(name),cy.get(`#${name}-actions`).click(), cy.get('div[triggerclassname="overflow-permission-tooltip"]', { timeout: 2000 }).should('not.exist') },
    clickActionButton: () => { cy.get(commonElementSelectors.elements.actionsButton).click() },
    menuClickEdit: () => cy.get('button[data-table-action="table.actions.connection.edit"]').click(),
    menuClickEditLabels: () => cy.get('button[data-table-action="table.actions.cluster.edit.labels"]').click(),
    menuClickDelete: () => cy.get('button[data-table-action="table.actions.connection.delete"]').click(),
    menuClickDeleteType: (type) => cy.get(`button[data-table-action="table.actions.${type}.delete"]`).click(),
    menuClickDestroy: () => cy.get('button[data-table-action="table.actions.cluster.destroy"]').click(),
    menuClickDetach: () => cy.get('button[data-table-action="table.actions.cluster.detach"]').click(),
    clearSearch: () => cy.get('.pf-c-search-input__text-input').clear(),
    searchTable: (name) => {
      commonPageMethods.resourceTable.clearSearch();
      cy.get('.pf-c-search-input__text-input').type(name)
      cy.wait(5 * 1000)
    },
    rowShouldToggle: function (name) {
      cy.get('#pf-c-page').then(page => {
        if (page.find('.pf-c-search-input__text-input', { timeout: 15000 }).length > 0) {
          cy.get('.pf-c-search-input__text-input').clear({ force: true })
          cy.get('.pf-c-search-input__text-input').type(name)
        }
      })
      cy.get(`tr[data-row-name="${name}"]`).get('input[name="tableSelectRow"]').click({ force: true })
    },
    rowCount: () =>
      cy.get("table", { timeout: 30 * 1000 }).then($table => {     
        return $table.find("tbody").find("tr").length;
    }),
  },

  modal : {
    getDialogElement: () => { 
      return cy.get(commonElementSelectors.elements.dialog, { withinSubject:null }) },
    shouldBeOpen: () => cy.get('div[role="dialog"]', {withinSubject:null}).should('exist'),
    shouldBeClosed: () => cy.get('.pf-c-modal-box pf-m-warning pf-m-md', {withinSubject:null}).should('not.exist'),
    clickDanger: (text) => cy.get('div[role="dialog"]',{withinSubject:null}).contains('button',text).click(),
    clickPrimary: () => cy.get('div[role="dialog"]',{withinSubject:null}).contains('button','Cancel') .click({ timeout: 20000 }),
    clickSecondaryClose: () => cy.get('button[aria-label="Close"]',{withinSubject:null}).click(),
    confirmAction: text => cy.get('#confirm',{withinSubject:null}).type(text)
  },

  notification: {
    shouldExist: type =>
      cy.get(`.pf-c-alert.pf-m-${type}`, { timeout: 20 * 1000 })
        .should("exist"),
    shouldSuccess: () =>
      cy.contains(".pf-c-alert__title","Success").should("be.visible")
    }
  }

