/** *****************************************************************************
 * Licensed Materials - Property of Red Hat, Inc.
 * Copyright (c) 2021 Red Hat, Inc.
 ****************************************************************************** */

import { clustersPageMethods } from '../views/clusters/managedCluster'
import { commonElementSelectors } from './common/commonSelectors'
/// <reference types="cypress"/>
// const ACM_VER=Cypress.env('ACM_VER')
import { credentialsPageMethods } from './credentials/credentials'
import { genericFunctions } from '../support/genericFunctions'
import { managedClustersMethods } from './clusters/managedCluster'

export const acm23xheaderSelectors = {
    mainHeader : ".pf-c-page__header",
    leftNavigation : {
        hamburbgerButton : "button[aria-label='Global navigation']",
        leftSideBar : '#page-sidebar',
        leftSideBarNav : 'nav[aria-label="Global"]',
        leftSideBarNavList : 'nav[aria-label="Global"] ui',
        listItemsText : {
            credentials : "Credentials",
            infrastructureText : {
                infrastructure : 'Infrastructure',
                clusters : "Clusters",
                bareMetalAssets : 'Bare metal assets',
                automation : "Automation",
                infrastructureEnvironment: "Infrastructure environment"
            },       
        }
    },
    headerTools : {
        userDropdown :'nav button[aria-label="user-menu"]',
        text : {
            logout : 'Logout'
        }
    } 
}

export const acm23xheaderMethods = {
        // left navigation methods
        openMenu: () => {
            var menuOpened = cy.get(acm23xheaderSelectors.leftNavigation.leftSideBar).invoke('attr','aria-hidden')
            if(menuOpened == false){
                cy.get(acm23xheaderSelectors.hamburbgerButton).click()
                cy.wait(500).get(acm23xheaderSelectors.leftNavigation.leftSideBarNavList).should('be.visible').and('have.length', 6)
            } 
        },
        // commenting the methods as they are not being used now, will uncomment when need.
        // FIX ME: the commentd methods have elements of the old ui so need to update it for the new 2.3 ui

        // validateMenu: () => {
        //     cy.get('.hamburger-btn').click()
        //     cy.get('#left-nav li').should('be.visible').and('have.length', 5)
        //     cy.get('.hamburger-btn').click()
        //     cy.get('#left-nav').should('not.exist')
        // },
        // goToHome: () => {
        //     leftNav.openMenu()
        //     cy.get('#left-nav').contains('Home') // FIXME: .click() disabling click until welcome page is updated.
        //     // welcomePage.shouldExist()
        // },
        // goToOverview: () => {
        //     leftNav.openMenu()
        //     cy.get('#left-nav').contains('Observe environments').trigger('mouseover')
        //     cy.get('#secondary-nav').contains('Overview').click()
        //     overviewPage.shouldExist()
        // },
        expandInfrastructure: () => {
            var isexpanded = cy.contains(commonElementSelectors.elements.button,acm23xheaderSelectors.leftNavigation.listItemsText.infrastructureText.infrastructure).invoke('attr','aria-expanded')
            if (isexpanded != true){
                cy.contains(commonElementSelectors.elements.button,acm23xheaderSelectors.leftNavigation.listItemsText.infrastructureText.infrastructure).click()
            }
            cy.contains(commonElementSelectors.elements.a,acm23xheaderSelectors.leftNavigation.listItemsText.infrastructureText.clusters).should('exist').click()
        },
        goToClusters: () => {
            acm23xheaderMethods.openMenu()
            cy.contains(commonElementSelectors.elements.a,acm23xheaderSelectors.leftNavigation.listItemsText.infrastructureText.clusters).should('exist').click({force: true})
            managedClustersMethods.shouldLoad()
        },
        goToClustersWithUser: () => {
            acm23xheaderMethods.openMenu()
            cy.contains(commonElementSelectors.elements.a,acm23xheaderSelectors.leftNavigation.listItemsText.infrastructureText.clusters).should('exist').click({force: true})
            cy.get('.pf-c-page').should('contain', 'Clusters')
            cy.wait(4000)
            cy.get('button[id=createCluster]').should('exist').and('have.class', 'pf-c-button pf-m-primary pf-m-aria-disabled')
            cy.get('button[id=importCluster]').should('exist').and('have.class', 'pf-c-button pf-m-secondary pf-m-aria-disabled')
        },
        goToInfrastructureEnvironmentPage: () => {
            acm23xheaderMethods.openMenu()
            cy.contains(commonElementSelectors.elements.a, acm23xheaderSelectors.leftNavigation.listItemsText.infrastructureText.infrastructureEnvironment).should('exist').click()
        },
        gotoCredentials: () => {
          genericFunctions.checkIfElementExists('Add Credential').then((onCredentials) => {
            if(!onCredentials){
              acm23xheaderMethods.openMenu()
              cy.contains(commonElementSelectors.elements.a,acm23xheaderSelectors.leftNavigation.listItemsText.credentials).should('exist').click()
              credentialsPageMethods.shouldLoad()  
            }else{
                cy.log('Already on credentials page!')
            }
          })
        },
        // goToBMAssets: () => {
        //     leftNav.openMenu()
        //     cy.get('#left-nav').contains('Automate infrastructure').trigger('mouseover')
        //     cy.get('#secondary-nav').contains('Bare metal assets').click()
        //     bmAssetPage.shouldExist()
        // },
        // goToApplications: () => {
        //     leftNav.openMenu()
        //     cy.get('#left-nav').contains('Manage applications').click()
        //     applicationPage.shouldExist()
        // },
        // goToGRC: () => {
        //     leftNav.openMenu()
        //     cy.get('#left-nav').contains('Govern risk').click()
        //     grcPage.shouldExist()
        // }
}