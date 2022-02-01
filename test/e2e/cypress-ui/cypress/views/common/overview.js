/* Copyright Red Hat */

/// <reference types="cypress" />

export const overviewPage = {
    whenGoToOverviewPage:() => cy.visit('/overview'),
    shouldLoad: () => {
        cy.get('.pf-c-page').should('contain', 'Overview')
        cy.get('.pf-c-spinner', { timeout: 20000 }).should('not.exist')
    },
    shouldLoadCloudConnectionPage: () => {
        cy.get('.pf-c-page').should('contain', 'Create cluster')
    },
    shouldHaveLinkToSearchPage: () => {
        cy.visit('/overview')
        cy.get('#clusters-summary a').contains(/[0-9]+/).then((c) => {
            let cluster = c.text()
            cy.wrap(c).click().then((p) => {
                cy.wrap(p).get('.react-tags__selected').should('have.length', 1).invoke('text').should('eq', 'kind:cluster')
                cy.wrap(p).get('.pf-c-expandable-section__toggle-text').invoke('text').should('equal', 'Cluster ('+cluster+')')
            })
        })
        cy.visit('/overview')
        cy.get('#pods-summary a').then((a) => {
            let pod = a.text()
            cy.wrap(a).click().then((p) => {
                cy.wrap(p).get('.react-tags__selected').should('have.length', 1).invoke('text').should('eq', 'kind:pod')
                cy.wrap(p).get('.pf-c-expandable-section__toggle-text').invoke('text').should('equal', 'Pod ('+pod+')')
            })
        })
    },
}
