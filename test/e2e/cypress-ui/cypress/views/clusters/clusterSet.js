/** *****************************************************************************
 * Licensed Materials - Property of Red Hat, Inc.
 * Copyright (c) 2021 Red Hat, Inc.
 ****************************************************************************** */

/// <reference types="cypress" />

import { commonElementSelectors, commonPageMethods } from '../common/commonSelectors'
import { acm23xheaderMethods} from '../header'
import { genericFunctions } from '../../support/genericFunctions'
import * as clusterAPI from '../../apis/cluster';

export const managedClusterSetSelectors = {
    elementTab: {
        clusterSetTab: 'Cluster sets'
    },
    elementText: {
        createClusterSetButton: 'Create cluster set',
        addUserOrGroup: 'Add user or group'
    },
    clusterSetTableRowOptionsMenu: {
        editNamespaceBinding: 'a[text="Edit namespace bindings"]',
        manageResourceAssignment: 'a[text="Manage resource assignments"]',
        deleteClusterSet: 'a[text="Delete cluster set"]',
    },
    clusterSetTableColumnFields: {
        name : '[data-label="Name"]',
        multiClusterNetworkStatus : '[data-label="Multi-cluster network status"]',
        clusters : '[data-label="Clusters"]',
        clusterPools : '[data-label="Cluster pools"]',
        namespaceBindings : '[data-label="Namespace bindings"]'
    },
    actionsDropdown: '#toggle-id',
    clusterSetDetailsTabs: {
        overview: 'a[text="Overview"]',
        subMarinerAddon: 'a[text="Submariner add-ons"]',
        managedClusters: 'a[text="Managed clusters"]',
        clusterPools: 'a[text="Cluster pools"]',
        accessManagement: 'a[text="Access management"]'
    },
    accessManagementTableColumnFields: {
        name: '[data-label="Name"]',
        displayRole: '[data-label="Display role"]',
        clusterRole: '[data-label="Cluster role"]',
        type: '[data-label="Type"]'
    },
    accessManagementTableRowOptionsMenu: {
        remove: 'button[text="Remove"]'
    }
}

export const clusterSetDetailMethods = {
    shouldLoad: () => {
        cy.get('.pf-c-page', {timeout: 10000}).should('contain', 'Clusters')
        cy.get('.pf-c-nav__link', {timeout: 2000}).filter(':contains("Overview")').should('exist')
        cy.wait(2000)
    },
    goToClusterSetDetail: (clusterSet) => {
        managedClusterSetMethods.goToClusterSets()
        managedClusterSetMethods.shouldLoad()
        commonPageMethods.resourceTable.rowShouldExist(clusterSet)
        cy.get(managedClusterSetSelectors.clusterSetTableColumnFields.name).contains(commonElementSelectors.elements.a, clusterSet).click();
        clusterSetDetailMethods.shouldLoad()
    },
    goToAccessManagement: () => {
        cy.get('.pf-c-nav__link', {timeout: 2000}).filter(':contains("Access management")').click()
        cy.get('button', {timeout: 10000}).should('contain', 'Add user or group')
    },
    goToManagedClusters: () => {
        cy.get('.pf-c-nav__link', {timeout: 2000}).filter(':contains("Managed clusters")').click()
    },
    addUserRoleToClusterSet: (clusterSet, userName, role) => {
        clusterSetDetailMethods.goToClusterSetDetail(clusterSet)
        clusterSetDetailMethods.goToAccessManagement()
        cy.wait(500).contains(commonElementSelectors.elements.button, managedClusterSetSelectors.elementText.addUserOrGroup).click({force: true})
        commonPageMethods.modal.shouldBeOpen()
        genericFunctions.selectOrTypeInInputDropDown('role-label', userName)
        cy.get('#role-label').parent().next().find('button').click().then(() => cy.contains(commonElementSelectors.elements.selectMenuItem, role).click())
        cy.get('button[type="submit"]').click()
    },
    addUserRoleToClusterSetWithoutPermission: (clusterSet) => {
        clusterSetDetailMethods.goToClusterSetDetail(clusterSet)
        clusterSetDetailMethods.goToAccessManagement()
        cy.wait(500).contains(commonElementSelectors.elements.button, managedClusterSetSelectors.elementText.addUserOrGroup).click({force: true})
        commonPageMethods.modal.shouldBeOpen()
        cy.get('#role-label').find(commonElementSelectors.elements.input).click().then(() => cy.get('ul[id="role"]').contains("No results found"))
    },
    removeUserRoleFromClusterSet: (clusterSet, userName, role) => {
        clusterSetDetailMethods.goToClusterSetDetail(clusterSet)
        clusterSetDetailMethods.goToAccessManagement()
        commonPageMethods.resourceTable.searchTable(role)
        cy.wait(1500).get(managedClusterSetSelectors.accessManagementTableColumnFields.name, {timeout: 2000}).contains(userName)
        cy.get('button[aria-label="Actions"]').click()
        cy.get('button.pf-c-dropdown__menu-item', {timeout: 2000}).should('contain', 'Remove').click({ timeout: 10000 })
        cy.get('button[type="submit"]', {timeout: 2000}).click()
    },
    checkUserRoleFromClusterSet: (clusterSet, userName, role) => {
        clusterSetDetailMethods.goToClusterSetDetail(clusterSet)
        clusterSetDetailMethods.goToAccessManagement()
        commonPageMethods.resourceTable.searchTable(userName)
        cy.wait(1500).get(managedClusterSetSelectors.accessManagementTableColumnFields.name, {timeout: 2000}).contains(userName)
        cy.get(managedClusterSetSelectors.accessManagementTableColumnFields.displayRole).contains(role);
    }
}

export const manageResourceAssignmentMethods = {
    shouldLoad: () => {
        cy.get('.pf-c-page', {timeout: 10000}).should('contain', 'Clusters')
        cy.get('button', {timeout: 10000}).should('contain','Review')
        cy.wait(2000)
    },
    addClusterToClusterSet: (clusterName, clusterSet) => {
        manageResourceAssignmentMethods.shouldLoad()
        cy.url().should('include', clusterSet)
        commonPageMethods.resourceTable.searchTable(clusterName)
        cy.wait(1500)
        cy.get(managedClusterSetSelectors.clusterSetTableColumnFields.name).contains(clusterName).parent().prev().find('input').check().should('be.checked');
        cy.get('button[id="save"]').click()
        commonPageMethods.modal.clickDanger('Save')
        cy.wait(4000)
        clusterSetDetailMethods.goToManagedClusters()
        commonPageMethods.resourceTable.rowShouldExist(clusterName)
    },
    removeClusterFromClusterSet: (clusterName, clusterSet) => {
        manageResourceAssignmentMethods.shouldLoad()
        cy.url().should('include', clusterSet)
        commonPageMethods.resourceTable.searchTable(clusterName)
        cy.wait(1500)
        cy.get(managedClusterSetSelectors.clusterSetTableColumnFields.name).contains(clusterName).parent().prev().find('input').uncheck().should('not.be.checked')
        cy.get('button[id="save"]').click()
        commonPageMethods.modal.clickDanger('Save')
        cy.wait(2000)
        clusterSetDetailMethods.goToManagedClusters()
        commonPageMethods.resourceTable.shouldNotExist(clusterName)
    }
}

/**
 * This object contais the group of methods that are part of clusterSet page
 */
export const managedClusterSetMethods = {
    shouldLoad: () => {
        cy.get('.pf-c-page', {timeout: 10000}).should('contain', 'Clusters')
        cy.get('button', {timeout: 10000}).should('contain','Create cluster set')
    },
    // The function used to go to clusterset page when login as admin user.
    goToClusterSets: () => {
        acm23xheaderMethods.goToClusters()
        cy.get('.pf-c-nav__link', {timeout: 2000}).filter(`:contains("${managedClusterSetSelectors.elementTab.clusterSetTab}")`).click()
        managedClusterSetMethods.shouldLoad()
    },
    goToClusterSetTable: (clusterSet) => {
        managedClusterSetMethods.goToClusterSets()
        commonPageMethods.resourceTable.rowShouldExist(clusterSet)
    },
    // The function used to go to clusterset page when login as non-admin user.
    goToClusterSetsWithUser: () => {
        acm23xheaderMethods.goToClustersWithUser()
        cy.get('.pf-c-nav__link', {timeout: 2000}).filter(`:contains("${managedClusterSetSelectors.elementTab.clusterSetTab}")`).click()
        cy.wait(1000).get('button', {timeout: 2000}).should('contain','Create cluster set').and('have.class', 'pf-m-aria-disabled')
    },
    createClusterSet: (clusterSet) => {
        managedClusterSetMethods.goToClusterSets()
        // if the clusterset doesn't exist, let's create it. Otherwise we won't bother.
        clusterAPI.getClusterSet(clusterSet).then(resp => {
            if (resp.status == 404) {
                cy.get('.pf-c-page__main-section').then(($body) => {
                    if (!$body.text().includes("You don't have any cluster sets.")) {
                        cy.wait(3000).get('button[id="createClusterSet"]').should('exist').click()
                    }else{
                        cy.wait(3000).contains(commonElementSelectors.elements.button, managedClusterSetSelectors.elementText.createClusterSetButton).click()
                    }
                    cy.get('#clusterSetName').type(clusterSet)
                    cy.get('button[type="submit"]', {timeout: 2000}).click()
                    cy.get('div[role="dialog"]', {timeout: 2000}).contains("Cluster set successfully created").then(() => cy.contains('button', 'Close').click())
                })
            }
        })
    },
    deleteClusterSet: (clusterSet) => {
        managedClusterSetMethods.goToClusterSetTable(clusterSet)
        commonPageMethods.resourceTable.openRowMenu(clusterSet)
        cy.get(managedClusterSetSelectors.clusterSetTableRowOptionsMenu.deleteClusterSet).click()
        commonPageMethods.modal.confirmAction(clusterSet)
        cy.get('button[type="submit"]', {timeout: 2000}).click()
    },
    deleteClusterSetbyAction: (clusterSet) => {
        managedClusterSetMethods.goToClusterSetTable(clusterSet)
        cy.get(managedClusterSetSelectors.clusterSetTableColumnFields.name).contains(commonElementSelectors.elements.a,clusterSet).parent().parent().prev().find('input').check().should('be.checked');
        cy.get(managedClusterSetSelectors.actionsDropdown).should('exist').click();
        cy.get(commonElementSelectors.elements.a).contains("Delete cluster sets").should('exist').click();
        commonPageMethods.modal.shouldBeOpen()
        commonPageMethods.modal.confirmAction('confirm')
        cy.get('button[type="submit"]', {timeout: 2000}).click()
    },
    manageResourceAssignmentByOption: (clusterSet) => {
        managedClusterSetMethods.goToClusterSetTable(clusterSet)
        commonPageMethods.resourceTable.openRowMenu(clusterSet)
        cy.get(managedClusterSetSelectors.clusterSetTableRowOptionsMenu.manageResourceAssignment).click().then(() => cy.url().should('include', clusterSet + "/manage-resources"))
    },
    addNamespaceBindingByOption: (clusterSet, ns) => {
        managedClusterSetMethods.goToClusterSetTable(clusterSet)
        commonPageMethods.resourceTable.openRowMenu(clusterSet)
        cy.get(managedClusterSetSelectors.clusterSetTableRowOptionsMenu.editNamespaceBinding).click()
        commonPageMethods.modal.shouldBeOpen()
        genericFunctions.selectOrTypeInInputDropDown('namespaces-label', ns)
        cy.get('button[aria-label="Options menu"]').click()
        cy.get('button[type="submit"]', {timeout: 2000}).click()
    },
    deleteNamespaceBindingByOption: (clusterSet, ns) => {
        managedClusterSetMethods.goToClusterSetTable(clusterSet)
        commonPageMethods.resourceTable.openRowMenu(clusterSet)
        cy.get(managedClusterSetSelectors.clusterSetTableRowOptionsMenu.editNamespaceBinding).click()
        commonPageMethods.modal.shouldBeOpen()
        cy.get('.pf-c-chip__text').contains(ns).then(() => cy.get('button[aria-label="Remove"]').click())
        cy.get('button[type="submit"]', {timeout: 2000}).click()
    }
}