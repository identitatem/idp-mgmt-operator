/* Copyright Red Hat */

/// <reference types="cypress" />

import * as constants from '../../support/constants'

const consolePublic = (token) => {
    return cy.request({
        url: constants.apiUrl + "/api/v1/namespaces/openshift-config-managed/configmaps/console-public",
        headers: {
            Authorization: "Bearer " + token,
            "Content-Type": "application/json",
            Accept: "application/json"
        }
    }).then(resp => {
        return resp.body['data']['consoleURL']
    })
}

const acmVersion = (token) => {
    return cy.request({
        url: Cypress.config().baseUrl + "/multicloud/header/version",
        headers: {
            Authorization: "Bearer " + token,
            "Content-Type": "application/json",
            Accept: "application/json"
        }
    }).then(resp => {
        return resp.body['status']['currentVersion']
    })
}

const oauthTokenEndpoint = (token) => {
    return cy.request({
        url: constants.apiUrl + "/.well-known/oauth-authorization-server",
        headers: {
            Authorization: "Bearer " + token,
            "Content-Type": "application/json",
            Accept: "application/json"
        }
    }).then(resp => {
        return resp.body['token_endpoint']
    })
}

export const oauthIssuer = (token) => {
    return cy.request({
        url: constants.apiUrl + "/.well-known/oauth-authorization-server",
        headers: {
            Authorization: "Bearer " + token,
            "Content-Type": "application/json",
            Accept: "application/json"
        }
    }).then(resp => {
        return resp.body['issuer']
    })
}

export const welcomePage = {
    whenGoToWelcomePage:() => cy.visit('/multicloud/welcome'),
    shouldExist: () => {
        cy.get('.welcome--introduction').should('contain', 'Welcome! Let’s get started.')
        cy.get('.welcome--svcs').should('contain', 'Go to Overview').and('contain', 'Go to Clusters').and('contain', 'Go to Applications').and('contain', 'Go to Governance and risk')
    },
    validateSvcs: () => {
        cy.contains('Go to Overview').click()
        overviewPage.shouldExist()
        cy.visit('/multicloud/welcome')
        cy.contains('Go to Clusters').click()
        cy.get('.pf-c-title').should('contain', 'Cluster management')
        cy.visit('/multicloud/welcome')
        cy.contains('Go to Applications').click()
        cy.get('.secondary-header-wrapper').find('h1').should('contain', 'Applications')
        cy.visit('/multicloud/welcome')
        cy.contains('Go to Governance and risk').click()
        cy.get('.secondary-header-wrapper').find('h1').should('contain', 'Governance and risk')
        cy.visit('/multicloud/welcome')
    },
    validateConnect: () => {
        cy.get('[target="dev-community"]').should('have.prop', 'href', 'https://www.redhat.com/en/blog/products')
        cy.get('[target="support"]').should('have.prop', 'href', 'https://access.redhat.com/support')
    }
}

export const overviewPage = {
    shouldExist: () => {
        cy.get('.pf-c-page').should('contain', 'Overview')
    }
}

export const searchPage = {
    shouldExist: () => {
        cy.get('.pf-c-page').should('contain', 'Search')
    }
}

export const bmAssetPage = {
    shouldExist: () => {
        cy.get('.pf-c-page').should('contain', 'Bare metal assets')
    }
}

export const applicationPage = {
    shouldExist: () => {
        cy.get('.secondary-header', {timeout: 10000 }).should('contain', 'Applications')
    }
}

export const grcPage = {
    shouldExist: () => {
        cy.get('.secondary-header', {timeout: 10000 }).should('contain', 'Governance and risk')
    }
}

export const resourcePage = {
    shouldExist: () => {
        cy.get('.bx--modal-header__heading').should('contain', 'Create resource')
    }
}



export const userMenu = {
    openApps: () => {
        cy.get('.navigation-container #acm-apps-dropdown', {timeout: 20000}).click()
        cy.get('.dropdown-content-header', {timeout: 20000})
        cy.get('#applications a.dropwdown-content-items', {timeout: 20000}).should('have.attr', 'href').and('contain', 'console-openshift-console')
    },
    openSearch: () => {
        cy.get('.navigation-container #acm-search').click()
        searchPage.shouldExist()
    },
    openResources: () => {
        cy.get('.navigation-container #acm-create-resource').click()
        resourcePage.shouldExist()
    },
    openTerminal: () => {
        cy.get('.navigation-container #acm-kui-dropdown').click()
        cy.get('#acm-kui-dropdown #terminal-dropdown-content li').should('be.visible').and('have.length', 2).then((c) => {
            cy.wrap(c).get('#new-tab-terminal a').should('have.prop', 'href', Cypress.config().baseUrl + '/kui').and('have.attr', 'target').and('match', /_blank/)
            cy.wrap(c).get('#current-tab-terminal').click()
            cy.get('.bx--header__name').should('contain', 'Visual Web Terminal')
        })
    },
    openInfo: () => {
        cy.get('.navigation-container #acm-info-dropdown').click()
        cy.get('#acm-info-dropdown #info-dropdown-content li').should('be.visible').and('have.length', 2)
        // Since cypress doesn't support multitab testing, we can check to see if the link includes the documentation link.
        // For now we can exclude the doc version, since the docs might not be available for a certain release.
        cy.get('#acm-doc a').should('have.attr', 'href').and('contain', 'https://access.redhat.com/documentation/en-us/red_hat_advanced_cluster_management_for_kubernetes/')
        cy.getCookie('acm-access-token-cookie').should('exist').then((token, c) => {
            acmVersion(token.value).then((version) => {
                cy.get('#acm-about').click().get('.bx--loading', {timeout: 20000}).should('not.exist')
                cy.get('.version-details').should('exist').get('.version-details__no').should('contain', version)
                cy.get('.bx--modal-close').click()
            })
        })
    },
    openUser: () => {
        cy.get('.navigation-container #acm-user-dropdown').click()
        cy.getCookie('acm-access-token-cookie').should('exist').then((token) => {
            oauthTokenEndpoint(token.value).then((endpoint) => {
                cy.get('#acm-user-dropdown #acm-user-dropdown-content li').should('be.visible').and('have.length', 2)
                    .get('#configure-client a').should('have.prop', 'href', endpoint + '/request').and('have.attr', 'target').and('match', /_blank/)
            })
        })
        cy.get('.navigation-container #acm-user-dropdown').click()
        cy.get('#acm-user-dropdown #acm-user-dropdown-content li').should('be.not.visible')
    }
}
