/** *****************************************************************************
 * Licensed Materials - Property of Red Hat, Inc.
 * Copyright (c) 2021 Red Hat, Inc.
 ****************************************************************************** */

/// <reference types="cypress" />

import { commonElementSelectors, commonPageMethods } from '../common/commonSelectors'
import { acm23xheaderMethods} from '../header'
import { genericFunctions } from '../../support/genericFunctions'
import { managedClusterDetailMethods } from '../clusters/managedCluster'
import { clusterActions } from '../actions/clusterAction'
import * as constants from "../../support/constants";

export const managedClusterPoolsSelectors = {
    elementTab: {
        clusterPoolsTab: 'Cluster pools'
    },
    elementText: {
        createClusterPoolButton: 'Create cluster pool',
        addUserOrGroup: 'Add user or group'
    },
    clusterPoolTableRowOptionsMenu: {
        scaleClusterPool: 'a[text="Scale cluster pool"]',
        updateReleaseImage: 'a[text="Update release image"]',
        destroyClusterPool: 'a[text="Destroy cluster pool"]',
    },
    clusterSetTableColumnFields: {
        name : '[data-label="Name"]',
        multiClusterNetworkStatus : '[data-label="Multi-cluster network status"]',
        clusters : '[data-label="Clusters"]',
        clusterPools : '[data-label="Cluster pools"]',
        namespaceBindings : '[data-label="Namespace bindings"]'
    },
    createClusterPool: {
        // the following dropdown will work for provider connection and cluster toggle
        commonDropdownToggle: 'button[aria-label="Options menu"]',
        basicInformation: {
          clusterName: '#eman',
          clusterNamespace: '#emanspace',
          clusterSetPlaceHolderText: 'Select a cluster set',
          // clusterSetToggle : 'button[aria-label="Options menu"]',
          clusterSetList: 'clusterSet',
        },
        infrastructureProvider: {
          distributionType: {
            rhocp: '#red-hat-openshift-container-platform',
          },
          provider: {
            aws: '#amazon-web-services',
            gcp: '#google-cloud',
            azure: '#microsoft-azure'
          }
        },
        imageAndConnection: {
          releaseImageInput: '#imageSet'
        },
        masterNode: {
    
        }
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

/**
 * This object contais the group of methods that are part of clusterSet page
 */
export const managedClusterPoolMethods = {
    shouldLoad: () => {
        cy.get('.pf-c-page', {timeout: 10000}).should('contain', 'Clusters')
        cy.get('button', {timeout: 10000}).should('contain','Create cluster pool')
    },

    createClusterPool: ({provider, name, clusterPoolName, clusterSet, releaseImage, region}) => {
        managedClusterPoolMethods.goToClusterPools()

        cy.wait(500).contains(commonElementSelectors.elements.button, managedClusterPoolsSelectors.elementText.createClusterPoolButton).click()
        managedClusterPoolMethods.fillInfrastructureProviderDetails(provider, name);
        managedClusterPoolMethods.fillClusterPoolDetails(clusterPoolName, clusterSet, releaseImage)
        managedClusterPoolMethods.fillNodePoolsDetails(region)
        managedClusterPoolMethods.fillNetworkingDetails()
        managedClusterPoolMethods.fillProxyDetails()
        cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.create).click()
        
        managedClusterPoolMethods.isClusterPoolPreparing(clusterPoolName, provider)
    },

    isClusterPoolPreparing: (clusterPoolName, provider) => {
        managedClusterPoolMethods.shouldLoad()
        commonPageMethods.resourceTable.searchTable(clusterPoolName) // first result will be the most focused one
        cy.wait(1500)
        managedClusterPoolMethods.clusterPoolShowsName(clusterPoolName)
        managedClusterPoolMethods.expandClusterPool() // only if number is more than 0
        managedClusterPoolMethods.isClusterClaimCreating()
        if (provider != '') managedClusterPoolMethods.clusterPoolShowsProvider(provider)
        managedClusterPoolMethods.isClusterAvailableToClaim(false)
    },

    isClusterPoolReady: (clusterPoolName, provider) => {
        clusterActions.checkClusterPoolStatus(clusterPoolName, clusterPoolName + '-ns')
        managedClusterPoolMethods.shouldLoad()
        commonPageMethods.resourceTable.searchTable(clusterPoolName)        
        managedClusterPoolMethods.clusterPoolShowsName(clusterPoolName)
        managedClusterPoolMethods.expandClusterPool() // only if number is more than 0
        managedClusterPoolMethods.isClusterClaimHibernating()
        managedClusterPoolMethods.clusterPoolShowsProvider(provider)
        managedClusterPoolMethods.isClusterAvailableToClaim(true)
        managedClusterPoolMethods.doesClaimClusterAppears(true)  
    },

    claimCluster: (clusterPoolName) => {
        managedClusterPoolMethods.goToClusterPools()

        commonPageMethods.resourceTable.searchTable(clusterPoolName) // should exist, otherwise create

        managedClusterPoolMethods.expandClusterPool()
        managedClusterPoolMethods.isClusterClaimHibernating() 
        managedClusterPoolMethods.clickClaimCluster()

        commonPageMethods.modal.shouldBeOpen()
        cy.get('#clusterClaimName-label').type(clusterPoolName + 'claim')

        cy.get('button[type="submit"]', {timeout: 2000}).contains('Claim').click()

        commonPageMethods.modal.shouldBeOpen()
        genericFunctions.clickButton('View cluster')
        
        managedClusterDetailMethods.isClusterClaim(clusterPoolName + 'claim')
        managedClusterDetailMethods.isClusterStatus('Resuming')
        managedClusterDetailMethods.isClusterPool(clusterPoolName)

        // check cluster pool on managed cluster page - claim exists, status is Resuming, status is Ready

        managedClusterPoolMethods.goToClusterPools()
        managedClusterPoolMethods.isClusterPoolPreparing(clusterPoolName, '')

        // separate testcase: check if addons etc are good on cluster claim
    },

    scaleCluster: (clusterPoolName, plusMinus) => {
        managedClusterPoolMethods.goToClusterPoolClickRowOptions(clusterPoolName)
        cy.get(managedClusterPoolsSelectors.clusterPoolTableRowOptionsMenu.scaleClusterPool).click()
        commonPageMethods.modal.shouldBeOpen()
        cy.get('[name=scale]').invoke('val').then($currentScale => {
            plusMinus == "up" ? cy.get('button[aria-label="Plus"]').click() : cy.get('button[aria-label="Minus"]').click()
            cy.get('button[type="submit"]', {timeout: 2000}).contains('Scale').click().wait(1000)
            cy.get('[data-label="Available clusters"] > span')
                .invoke('text').then(val => {
                    let newVal = (plusMinus == "up" ? (Number($currentScale) + 1) : (Number($currentScale) - 1))
                    expect(val).to.include('out of ' + newVal)
                }) 
        })
    },

    /**
     * check available release images are actually supported
     * for clusterpools only
     * Expects to be on the update release images dialog
     * Uses a regex expression to test if versions shown are supported
     */
    validateSupportedReleaseImages: () => {
        cy.get('#releaseImage-label').click()
        cy.get('#releaseImage > li > button > .pf-c-select__menu-item-main').each(($releaseImage) => {
            var re = new RegExp("^OpenShift (" + constants.supportedOCPReleasesRegex + ").[0-9][0-9]*$")
            if (!(re).test($releaseImage.text())) throw new Error('unexpected release image found: ' + $releaseImage)
        })
    },

    changeReleaseImage: (clusterPoolName) => {
        managedClusterPoolMethods.goToClusterPoolClickRowOptions(clusterPoolName)
        cy.get(managedClusterPoolsSelectors.clusterPoolTableRowOptionsMenu.updateReleaseImage).click()

        managedClusterPoolMethods.valiateChangeReleaseImageByAction()
    },

    changeReleaseImageByAction: (searchFilter) => {
        managedClusterPoolMethods.goToClusterPools()
        commonPageMethods.resourceTable.searchTable(searchFilter)
        cy.wait(700).get('#select-all').click()
        cy.get(managedClusterPoolsSelectors.actionsDropdown).should('exist').click()
        cy.get('#updateReleaseImages').click()

        managedClusterPoolMethods.valiateChangeReleaseImageByAction()
    },

    valiateChangeReleaseImageByAction: () => {
        cy.get('td[data-label="Distribution version"]').invoke('text').then($releaseImage => {
            commonPageMethods.modal.shouldBeOpen()
            managedClusterPoolMethods.validateSupportedReleaseImages()
    
            cy.get('#releaseImage > li > button').first().click()
            cy.get('button[type="submit"]', {timeout: 2000}).contains('Update').click().wait(1000)

            cy.get('td[data-label="Distribution version"]')
                .invoke('text').then(val => {
                    expect(val).to.not.eq($releaseImage)
                })
        })
    },

    /**
    * Expands the first clusterpool to show all clusters tied to that pool
    */
     expandClusterPool: () => {
        cy.get('#expandable-toggle0').then(($expand) => {
            if (!$expand.hasClass('pf-m-expanded')) {
                cy.get('#expandable-toggle0').click()
            }
        })
    },

    clusterPoolShowsName: (clusterPoolName) => {
        cy.get('td[data-label="Name"]:visible').should('include.text', clusterPoolName)
    },

    clusterClaimHasStatus: (status) => {
        cy.get('td[data-label="Status"]:visible').as('status')
        switch (status) {
            case 'Creating':
                cy.get('@status').should('have.text', 'Creating')
                break;
            case 'Hibernating':
                cy.get('@status').should('have.text', 'Hibernating')
                break;
            case 'Destroying':
                cy.get('@status').should('have.text', 'Destroying')
                break;
        }
    },

    isClusterClaimCreating: () => {
        managedClusterPoolMethods.clusterClaimHasStatus('Creating')
    },

    isClusterClaimHibernating: () => {
        managedClusterPoolMethods.clusterClaimHasStatus('Hibernating')
    },

    isClusterClaimDestroying: () => {
        managedClusterPoolMethods.clusterClaimHasStatus('Destroying')
    },

    isClusterAvailableToClaim: (claimable) => {
        cy.get('td[data-label="Available to claim"]:visible').as('available')
        if (claimable) 
            cy.get('@available').should('have.text', 'Yes')
        else 
            cy.get('@available').should('have.text', 'No')
    },

    doesClaimClusterAppears: (claimable) => {
        if (!claimable) {
            cy.get('td[data-key="8"]:visible').should('not.exist')
            return
        }
        cy.get('td[data-key="8"]:visible').should('have.text', 'Claim cluster')
    },

    clickClaimCluster: () => {
        cy.get('td[data-key="8"]:visible').should('have.text', 'Claim cluster').eq(0).click()
    },

    clusterPoolShowsProvider: (provider) => {
        cy.get('td[data-label="Infrastructure provider"]:visible').should('include.text', provider)
    },

    clustersAvailable: (available) => {
        cy.get('td[data-label="Available clusters"]:visible').should('include.text', available)
    },
    
    goToClusterPools: () => {
        acm23xheaderMethods.goToClusters()
        cy.get('.pf-c-nav__link', {timeout: 2000}).filter(`:contains("${managedClusterPoolsSelectors.elementTab.clusterPoolsTab}")`).click()
        managedClusterPoolMethods.shouldLoad()
    },

    /**
    * This functon accepts provider name and name of the credentials
    * @param {*} provider 
    * @param {*} credentialName 
    */
    fillInfrastructureProviderDetails: (provider, credentialName) => {
        switch (provider) {
            case 'Amazon Web Services':
                cy.get(managedClusterPoolsSelectors.createClusterPool.infrastructureProvider.provider.aws).click();
                break;
            case 'Microsoft Azure':
                cy.get(managedClusterPoolsSelectors.createClusterPool.infrastructureProvider.provider.azure).click();
                break;
            case 'Google Cloud':
                cy.get(managedClusterPoolsSelectors.createClusterPool.infrastructureProvider.provider.gcp).click();
                break;
            default:
                cy.log(provider + ' is an invalid provider name! These are the supported provider names: aws, azure, gcp, vmware')
        }
        genericFunctions.selectOrTypeInInputDropDown('connection-label', credentialName)
        genericFunctions.clickNext()
    },

    fillClusterPoolDetails: (clusterPoolName, clusterSet, releaseImage, additionalLabels) => {
        cy.get(managedClusterPoolsSelectors.createClusterPool.basicInformation.clusterName).click().type(clusterPoolName)
        cy.get(managedClusterPoolsSelectors.createClusterPool.basicInformation.clusterNamespace).click().type(clusterPoolName + '-ns');
        if (clusterSet != '') {
          genericFunctions.selectOrTypeInInputDropDown('clusterSet-label', clusterSet)
        }
        genericFunctions.selectOrTypeInInputDropDown('imageSet-group', releaseImage, true)
        genericFunctions.clickNext()
    },

    fillNodePoolsDetails: (region) => {
        if (region != '') {
          genericFunctions.selectOrTypeInInputDropDown('region-label', region, true)
        }
        genericFunctions.clickNext()
    }, 

    fillNetworkingDetails: () => {
        genericFunctions.clickNext()
    },   

    fillProxyDetails: () => {
        genericFunctions.clickNext()
    },    
    
    destroyClusterPool: (clusterPool) => {
        managedClusterPoolMethods.goToClusterPoolClickRowOptions(clusterPool)
        cy.get(managedClusterPoolsSelectors.clusterPoolTableRowOptionsMenu.destroyClusterPool).click()
        commonPageMethods.modal.confirmAction(clusterPool)
        managedClusterPoolMethods.clickSubmitButton()
        cy.wait(5000).get('.pf-c-page__main-section').then(($body) => {
            if (!$body.text().includes("You don't have any cluster pools")) {
                get('[data-label="Name"]').should('not.include.text', clusterPool)
            }
        })
    },

    destroyClusterPoolbyAction: (searchFilter) => {
        managedClusterPoolMethods.goToClusterPools()
        cy.get('.pf-c-page__main-section').then(($body) => {
            if (!$body.text().includes("You don't have any cluster pools")) {
                commonPageMethods.resourceTable.searchTable(searchFilter)
                cy.wait(700).get('#select-all').click()
                cy.get(managedClusterPoolsSelectors.actionsDropdown).should('exist').click()
                cy.get('#destroyClusterPools').click()
                commonPageMethods.modal.shouldBeOpen()
                commonPageMethods.modal.confirmAction('confirm')
                managedClusterPoolMethods.clickSubmitButton()
            }
        })
    },

    goToClusterPoolClickRowOptions: (clusterPool) => {
        managedClusterPoolMethods.goToClusterPools()
        commonPageMethods.resourceTable.openRowMenu(clusterPool)
    },

    clickSubmitButton: () => {
        cy.get('button[type="submit"]', {timeout: 2000}).click()
    }
}