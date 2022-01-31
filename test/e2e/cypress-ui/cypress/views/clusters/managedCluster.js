/* Copyright Red Hat */

/// <reference types="cypress" />

import { acm23xheaderMethods, acm23xheaderSelectors } from '../header'
import { commonElementSelectors, commonPageMethods } from '../common/commonSelectors'
import { clusterActions } from '../actions/clusterAction';
import { genericFunctions } from '../../support/genericFunctions'
import * as automation from '../../apis/automation';
import * as constants from '../../support/constants'
import * as managedCluster from '../../apis/cluster'

export const clusterType = {
  EKS: "EKS",
  AKS: "AKS",
  GKE: "GKE",
  IKS: "IKS",
  ROKS: "ROKS",
  AWS: "AWS",
  GCP: "GCP",
  AZURE: "AZURE"
}

export const clusterStatus = {
  READY: "Ready",
  FAILED: "Failed",
  PENDING_IMPORT: "Pending Import",
  CREATING: "Creating",
  DESTROYING: "Destroying",
  HIBERNATING: "Hibernating",
  RESUMING: "Resuming"
}

//const { options } = JSON.parse(Cypress.env('ENV_CONFIG'))

export const managedClustersSelectors = {
  elementText: {
    createClusterButton: 'Create cluster',
    importClusterButton: 'Import cluster'
  },
  createCluster: {
    // the following dropdown will work for provider connection and cluster toggle
    commonDropdownToggle: 'button[aria-label="Options menu"]',
    basicInformation: {
      clusterName: '#eman',
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
        azure: '#microsoft-azure',
        vmware: '#vmware-vsphere',
        openstack: '#red-hat-openstack-platform',
        baremetal: '#bare-metal'
      }
    },
    imageAndConnection: {
      releaseImageInput: '#imageSet'
    },
    masterNode: {

    },
    singleNode: '#singleNode'
  },
  importCluster: {
    clusterName: '#clusterName',
  },
  clusterTableRowOptionsMenu: {
    editLables: 'a[text="Edit labels"]',
    selectChannel: 'a[text="Select channel"]',
    searchCluster: 'a[text="Search cluster"]',
    hibernateCluster: 'a[text="Hibernate cluster"]',
    resumeCluster: 'a[text="Resume cluster"]',
    detachCluster: 'a[text="Detach cluster"]',
    destroyCluster: 'a[text="Destroy cluster"]',
  },
  clusterOverviewPage: {
  },
  managedClusterDetailsTabs: {
    overview: 'a[text="Overview"]',
    nodes: 'a[text="Nodes"]',
    machinePools: 'a[text="Machine pools"]',
    addOns: 'a[text="Add-ons"]'
  },
  clusterTableColumnFields: {
    name: '[data-label="Name"]',
    status: '[data-label="Status"]',
    infraProvider: '[data-label="Infrastructure provider"]',
    distrVersion: '[data-label="Distribution version"]',
    labels: '[data-label="Labels"]',
    nodes: '[data-label="Nodes"]'
  },
  actionsDropdown : '#toggle-id',
  automationTemplate: '#templateName',
}

export const clustersPageMethods = {
  /**
   * verified that the DOM contains text "Cluster management"
   */
  shouldExist: () => {
    cy.contains(pageElements.h1, acm23xheaderSelectors.leftNavigation.listItemsText.infrastructureText.clusters).should('contain', 'Cluster management')
  },

  // shouldHaveLinkToSearchPage: () => {
  //   cy.visit('/multicloud/clusters')
  //   cy.get('.pf-c-table tbody').find('tr').first().then((c) => {
  //     let name = c.find('[data-label="Name"] a').text()
  //     cy.wrap(c).find('.pf-c-dropdown__toggle').click().get('a').contains('Search cluster').click().then(() => cy.url().should('include', `/search?filters={%22textsearch%22:%22cluster%3A${name}%22}`))
  //   })
  // }
}

export const managedClusterDetailMethods = {
  shouldLoad: () => {
    cy.get('.pf-c-page', { timeout: 10000 }).should('contain', 'Clusters')
    cy.get('.pf-c-nav__link', { timeout: 2000 }).filter(':contains("Overview")').should('exist')
    cy.wait(2000)
  },

  goToManagedClusterOverview: (clusterName) => {
    acm23xheaderMethods.goToClusters()
    commonPageMethods.resourceTable.rowShouldExist(clusterName)
    cy.get(managedClustersSelectors.clusterTableColumnFields.name).contains(commonElementSelectors.elements.a, clusterName).click();
    managedClusterDetailMethods.shouldLoad()
  },

  clickEditChannels: () => {
    cy.get('[aria-label="Select channels"]').click()
    commonPageMethods.modal.shouldBeOpen()
  },

  getCurrentChannel: () => {
    return cy.get(':nth-child(6) > .pf-c-description-list__description > .pf-c-description-list__text > span', { timeout: 300000 })
  },

  getNewChannelButton: () => {
    return cy.get('[data-label="New channel"]').find('button')
  },

  getNewChannelDropdownButton: (clusterName) => {
    return cy.get('#' + clusterName + '-upgrade-selector-label').find('button')
  },

  saveNewChannel: () => {
    cy.get('[type=Submit]').contains('Save').click()
  },

  goToClusterAddons: () => {
    cy.get('.pf-c-nav__link', { timeout: 2000 }).filter(':contains("Add-ons")').click()
  },

  clusterSectionContainsItem: (term, value) => {
    return cy.get('div .pf-c-card__body').find('dl').contains(term).parent().parent().contains(value)
  },

  isClusterStatus: (status) => {
    managedClusterDetailMethods.clusterSectionContainsItem('Status', status)
  },

  isClusterClaim: (claimName) => {
    managedClusterDetailMethods.clusterSectionContainsItem('Cluster claim name', claimName)
  },
  isClusterProvider: (provider) => {
    managedClusterDetailMethods.clusterSectionContainsItem('Infrastructure provider', provider)
  },
  isClusterVersion: (version) => {
    managedClusterDetailMethods.clusterSectionContainsItem('Distribution version', version)
  },
  isClusterChannel: (channel) => {
    managedClusterDetailMethods.clusterSectionContainsItem('Channel', channel)
  },
  isClusterSet: (clusterSetName) => {
    managedClusterDetailMethods.clusterSectionContainsItem('Cluster set', clusterSetName)
  },
  isClusterPool: (clusterPoolName) => {
    managedClusterDetailMethods.clusterSectionContainsItem('Cluster pool', clusterPoolName)
  },
}
/**
 * This object contais the group of methods that are part of managed clusters page
 */
export const managedClustersMethods = {
  shouldLoad: () => {
    cy.get('.pf-c-page').should('contain', 'Clusters')
    cy.wait(4000)
    cy.get('button[id=createCluster]').should('exist').and('not.have.class', 'pf-m-aria-disabled')
    cy.get('button[id=importCluster]').should('exist').and('not.have.class', 'pf-m-aria-disabled')
    // cy.get('.pf-c-spinner', { timeout: 20000 }).should('not.exist')
  },

  /**
   * This functon accepts provider name and name of the credentials
   * @param {*} provider 
   * @param {*} credentialName 
   */
  fillInfrastructureProviderDetails: (provider, credentialName) => {
    switch (provider) {
      case 'Amazon Web Services':
        cy.get(managedClustersSelectors.createCluster.infrastructureProvider.provider.aws).click();
        break;
      case 'Microsoft Azure':
        cy.get(managedClustersSelectors.createCluster.infrastructureProvider.provider.azure).click();
        break;
      case 'Google Cloud':
        cy.get(managedClustersSelectors.createCluster.infrastructureProvider.provider.gcp).click();
        break;
      case 'VMware vSphere':
        cy.get(managedClustersSelectors.createCluster.infrastructureProvider.provider.vmware).click();
        break;
      case 'Red Hat OpenStack Platform':
        cy.get(managedClustersSelectors.createCluster.infrastructureProvider.provider.openstack).click();
        break;
      default:
        cy.log('Invalid provider name! These are the supported provider names: aws, azure, gcp, vmware')
    }
    // cy.get('#namespaceName-form-group').find('input').click()
    // cy.get(credentialsPageSelectors.namespaceDropdownToggle).should('exist').click()
    // cy.contains(credentialsPageSelectors.selectMenuItem, namespace).click()
    genericFunctions.selectOrTypeInInputDropDown('connection-label', credentialName)
    cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click()
  },

  fillClusterDetails: (clusterName, clusterSet, releaseImage, additionalLabels, enableSNO) => {
    cy.get(managedClustersSelectors.createCluster.basicInformation.clusterName).click().type(clusterName);
    if (clusterSet != '') {
      genericFunctions.selectOrTypeInInputDropDown('clusterSet-label', clusterSet)
    }
    genericFunctions.selectOrTypeInInputDropDown('imageSet-group', releaseImage, true)
    if (Boolean(enableSNO)) cy.get(managedClustersSelectors.createCluster.singleNode).click();
    if (additionalLabels != '') {
      cy.get('#additional').type(additionalLabels)
    }
    cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click()
  },

  /**
    * check available release images are actually supported
    * for clusterpools only
    * Expects to be on the update release images dialog
    * Uses a regex expression to test if versions shown are supported
    */
  validateSupportedReleaseImages: () => {
    cy.get('#imageSet').click()
    cy.get('.tf--list-box__menu-item > div > div:first-of-type').each(($releaseImage) => {
      var re = new RegExp("^OpenShift " + constants.supportedOCPReleasesRegex + ".[0-9][0-9]*$")
      if (!(re).test($releaseImage.eq(0).text()))
        throw new Error('Unexpected release image found: ' + $releaseImage.eq(0).text()
          + '\nSupported release major versions expected: ' + constants.supportedOCPReleasesRegex)
    })
  },

  fillMasterNodeDetails: (region) => {
    if (region != '') {
      genericFunctions.selectOrTypeInInputDropDown('region-label', region, true)
    }
    cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click()
  },

  fillOpenstackNodePools: () => {
    cy.get('#masterpool-control-plane-pool').click().then(() => cy.get('#masterType').clear().type('ocp-master'))
    cy.get('#workerpool-worker-pool-1').click().then(() => cy.get('#workerType').clear().type('ocp-master'))
    cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click()
  },

  // for now this methods only click on next button and uses default values
  fillWorkerPoolDetails: () => {
    cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click()
  },

  // for now this methods only click on next button and uses default values
  fillProxyDetails: () => {
    cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click()
  },

  // for now this methods only click on next button and uses default values
  fillNetworkingDetails: () => {
    cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click()
  },

  fillVMWareNetworkingDetails: (network, apiVIP, ingressVIP) => {
    cy.get('[data-testid=text-networkType]').type(network)
    cy.get('[data-testid=text-apiVIP]').type(apiVIP)
    cy.get('[data-testid=text-ingressVIP]').type(ingressVIP)

    cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click()
  },

  fillOpenstackNetworkingDetails: (extNetwork, apiFIP, ingressFIP, clusterCIDR, netHostPrefix, serviceCIDR, machineCIDR) => {
    cy.get('[data-testid="text-externalNetworkName"]').clear().type(extNetwork)
    cy.get('[data-testid="text-apiFloatingIP"]').clear().type(apiFIP)
    cy.get('[data-testid="text-ingressFloatingIP"]').clear().type(ingressFIP)
    cy.get('#networkgroup-network-').then(($body) => {
      if ($body.hasClass('#collapsed')) cy.get('#networkgroup-network-').click()
      cy.get('input[id="clusterNetwork"]').clear().type(clusterCIDR)
      cy.get('input[id="hostPrefix"]').clear().type(netHostPrefix)
      cy.get('input[id="serviceNetwork"]').clear().type(serviceCIDR)
      cy.get('input[id="machineCIDR"]').clear().type(machineCIDR)
    })

    cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click()
  },

  // for now this methods only click on next button and uses default values
  fillAutomationDetails: (templateName) => {
    if (templateName != null ) 
      cy.get(managedClustersSelectors.automationTemplate).type(templateName).type('{enter}')
    cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click()
  },

  clickCreateAndVerify: (clusterName) => {
    cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.create).click()
    commonPageMethods.notification.shouldExist('success')
    cy.location().should( loc => 
      expect(loc.pathname).to.eq(`/multicloud/clusters/${clusterName}/overview`)
    )
    acm23xheaderMethods.goToClusters()
    cy.get(commonElementSelectors.elements.pageSearch, { timeout: 15000 }).click().type(clusterName).wait(3000)
    cy.get(commonElementSelectors.elements.a).contains(clusterName).should('exist').click()
  },

  checkClusterTable: (clusterName) => {
    acm23xheaderMethods.goToClusters()
    cy.wait(500).get(commonElementSelectors.elements.pageSearch, { timeout: 15000 }).click().type(clusterName)
    cy.wait(1500).get(commonElementSelectors.elements.a).contains(clusterName).should('exist')
    cy.get(commonElementSelectors.elements.resetSearch, { timeout: 15000 }).click()
  },

  editClusterLabelsByOptions: (clusterName, label) => {
    acm23xheaderMethods.goToClusters()
    commonPageMethods.resourceTable.rowShouldExist(clusterName)
    commonPageMethods.resourceTable.openRowMenu(clusterName)
    cy.get(managedClustersSelectors.clusterTableRowOptionsMenu.editLables, { timeout: 5000 }).click()
    commonPageMethods.modal.shouldBeOpen()
    cy.get('input[id="labels-input"]').type(label)
    cy.get('button[type="submit"]').click()
  },

  searchClusterByOptions: (clusterName) => {
    acm23xheaderMethods.goToClusters()
    commonPageMethods.resourceTable.rowShouldExist(clusterName)
    commonPageMethods.resourceTable.openRowMenu(clusterName)
    cy.get(managedClustersSelectors.clusterTableRowOptionsMenu.searchCluster, { timeout: 5000 }).click().then(() => cy.url().should('include', `/search?filters={%22textsearch%22:%22cluster%3A${clusterName}%22}`))
  },

  hibernateClusterByOptions: (clusterName) => {
    acm23xheaderMethods.goToClusters()
    commonPageMethods.resourceTable.rowShouldExist(clusterName)
    commonPageMethods.resourceTable.openRowMenu(clusterName)
    cy.get(managedClustersSelectors.clusterTableRowOptionsMenu.hibernateCluster, { timeout: 5000 }).click()
    commonPageMethods.modal.shouldBeOpen()
    commonPageMethods.modal.clickDanger('Hibernate')
  },

  resumeClusterByOptions: (clusterName) => {
    acm23xheaderMethods.goToClusters()
    commonPageMethods.resourceTable.rowShouldExist(clusterName)
    commonPageMethods.resourceTable.openRowMenu(clusterName)
    cy.get(managedClustersSelectors.clusterTableRowOptionsMenu.resumeCluster, { timeout: 5000 }).click()
    commonPageMethods.modal.shouldBeOpen()
    commonPageMethods.modal.clickDanger('Resume')
  },

  hibernateClusterByAction: (clusterName) => {
    acm23xheaderMethods.goToClusters()
    commonPageMethods.resourceTable.rowShouldExist(clusterName)
    cy.get('input[id="select-all"]').uncheck().should('not.be.checked')
    cy.get(managedClustersSelectors.clusterTableColumnFields.name).contains(commonElementSelectors.elements.a, clusterName).parent().parent().prev().find('input').check().should('be.checked');
    cy.get(managedClustersSelectors.actionsDropdown).should('exist').click();
    cy.get(commonElementSelectors.elements.a).contains("Hibernate clusters").should('exist').click();
    commonPageMethods.modal.shouldBeOpen()
    commonPageMethods.modal.clickDanger('Hibernate')
  },

  resumeClusterByAction: (clusterName) => {
    acm23xheaderMethods.goToClusters()
    commonPageMethods.resourceTable.rowShouldExist(clusterName)
    cy.get('input[id="select-all"]').uncheck().should('not.be.checked')
    cy.get(managedClustersSelectors.clusterTableColumnFields.name).contains(commonElementSelectors.elements.a, clusterName).parent().parent().prev().find('input').check().should('be.checked');
    cy.get(managedClustersSelectors.actionsDropdown).should('exist').click();
    cy.get(commonElementSelectors.elements.a).contains("Resume clusters").should('exist').click();
    commonPageMethods.modal.shouldBeOpen()
    commonPageMethods.modal.clickDanger('Resume')
  },

  clickCreate: () => {
    cy.wait(500).contains(commonElementSelectors.elements.button, managedClustersSelectors.elementText.createClusterButton).click()
  },

  createCluster: ({ provider, name, clusterName, clusterSet, releaseImage, additionalLabels, region }) => {
    acm23xheaderMethods.goToClusters()
    managedCluster.getManagedCluster(clusterName).then((resp) => {
      if (resp.status === 404) {
        managedClustersMethods.clickCreate()
        managedClustersMethods.fillInfrastructureProviderDetails(provider, name);
        managedClustersMethods.fillClusterDetails(clusterName, clusterSet, releaseImage, additionalLabels);
        managedClustersMethods.fillMasterNodeDetails(region);
        managedClustersMethods.fillWorkerPoolDetails();
        managedClustersMethods.fillNetworkingDetails();
        managedClustersMethods.fillAutomationDetails();
        managedClustersMethods.clickCreateAndVerify(clusterName);
      }
    })
    managedClustersMethods.checkClusterTable(clusterName)
  },

  createVMWareCluster: ({ provider, name, clusterName, clusterSet, releaseImage, additionalLabels, network, apiVIP, ingressVIP }) => {
    acm23xheaderMethods.goToClusters()
    managedClustersMethods.clickCreate()
    managedClustersMethods.fillInfrastructureProviderDetails(provider, name)
    managedClustersMethods.fillClusterDetails(clusterName, clusterSet, releaseImage, additionalLabels)
    managedClustersMethods.fillMasterNodeDetails('')
    managedClustersMethods.fillVMWareNetworkingDetails(network, apiVIP, ingressVIP)
    managedClustersMethods.fillProxyDetails()
    managedClustersMethods.fillAutomationDetails()
    managedClustersMethods.clickCreateAndVerify(clusterName);
  },

  createOpenStackCluster: ({ provider, name, clusterName, clusterSet, releaseImage, additionalLabels, extNetwork, apiFIP, ingressFIP, clusterCIDR, netHostPrefix, serviceCIDR, machineCIDR}) => {
    acm23xheaderMethods.goToClusters()
    managedClustersMethods.clickCreate()
    managedClustersMethods.fillInfrastructureProviderDetails(provider, name)
    managedClustersMethods.fillClusterDetails(clusterName, clusterSet, releaseImage, additionalLabels)
    managedClustersMethods.fillOpenstackNodePools()
    managedClustersMethods.fillOpenstackNetworkingDetails(extNetwork, apiFIP, ingressFIP, clusterCIDR, netHostPrefix, serviceCIDR, machineCIDR)
    managedClustersMethods.fillProxyDetails()
    managedClustersMethods.fillAutomationDetails()
    managedClustersMethods.clickCreateAndVerify(clusterName);
  },

  destroyCluster: (clusterName) => {
    acm23xheaderMethods.goToClusters();
    managedCluster.getManagedCluster(clusterName).then((resp) => {
      if (resp.isOkStatusCode) { 
        acm23xheaderMethods.goToClusters()
        commonPageMethods.resourceTable.rowShouldExist(clusterName)
        commonPageMethods.resourceTable.openRowMenu(clusterName)
        cy.get(managedClustersSelectors.clusterTableRowOptionsMenu.destroyCluster).click()
        commonPageMethods.modal.shouldBeOpen()
        commonPageMethods.modal.confirmAction(clusterName)
        commonPageMethods.modal.clickDanger('Destroy')
      }
    })
  },

  destroyClusterWithClusterClaim: (clusterClaimName, namespace) => {
    acm23xheaderMethods.goToClusters()
    managedCluster.getClusterClaim(clusterClaimName, namespace).then(resp => {
      if (resp.isOkStatusCode) {
        managedClustersMethods.destroyCluster(resp.body.spec.namespace)
      }
    })
  },

  importCluster: (clusterName, clusterSet, additionalLabels, kubeconfig) => {
    acm23xheaderMethods.goToClusters()
    cy.wait(700).contains(commonElementSelectors.elements.button, managedClustersSelectors.elementText.importClusterButton).click()
    managedClustersMethods.fillImportClusterDetails(clusterName, clusterSet, additionalLabels, kubeconfig);
    cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.import).click();
    cy.contains('h1', clusterName, { timeout: 15000 }).should('exist');
    // wait before failing
    // check on main first before retrying
  },

  importClusterToken: (clusterName, clusterSet, additionalLabels, server, token) => {
    acm23xheaderMethods.goToClusters()
    cy.wait(700).contains(commonElementSelectors.elements.button, managedClustersSelectors.elementText.importClusterButton).click()
    managedClustersMethods.fillImportClusterDetailsToken(clusterName, clusterSet, additionalLabels, server, token);
    cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.import).click();
    cy.contains('h1', clusterName, { timeout: 15000 }).should('exist');
    // wait before failing
    // check on main first before retrying
  },

  fillImportClusterDetails: (clusterName, clusterSet, additionalLabels, kubeConfig) => {
    cy.get(managedClustersSelectors.importCluster.clusterName).type(clusterName)
    if (clusterSet != '') {
      //genericFunctions.selectOrTypeInInputDropDown('managedClusterSet-label', clusterSet)
      cy.get('#managedClusterSet-label').find(commonElementSelectors.elements.dropDownToggleButton).click()
      cy.contains(commonElementSelectors.elements.selectMenuItem, clusterSet).click()
    }
    if (additionalLabels != '') {
      cy.get('#additionalLabels').type(additionalLabels)
    }
    //genericFunctions.selectOrTypeInInputDropDown('import-mode-label','Kubeconfig', true)
    cy.get('#import-mode-label').find('.pf-c-select').click().click()
    cy.contains(commonElementSelectors.elements.selectMenuItem, 'Kubeconfig').click()
    cy.get('#kubeConfigEntry').invoke('val', kubeConfig).type(' ')
  },

  fillImportClusterDetailsToken: (clusterName, clusterSet, additionalLabels, server, token) => {
    cy.get(managedClustersSelectors.importCluster.clusterName).type(clusterName);
    if (clusterSet != '') {
      cy.get('#managedClusterSet-label').find(commonElementSelectors.elements.dropDownToggleButton).click()
      cy.contains(commonElementSelectors.elements.selectMenuItem, clusterSet).click()
    }
    if (additionalLabels != '') {
      cy.get('#additionalLabels').type(additionalLabels)
    }
    //genericFunctions.selectOrTypeInInputDropDown('import-mode-label','Kubeconfig', true)
    cy.get('#import-mode-label').find('.pf-c-select').click().click()
    cy.contains(commonElementSelectors.elements.selectMenuItem, 'Enter your server URL and API token for the existing cluster').click();
    cy.get('#server').type(server, { delay: 0 })
    cy.get('#token').type(token, { delay: 0 })
  },

  detachCluster: (clusterName) => {
    acm23xheaderMethods.goToClusters()
    commonPageMethods.resourceTable.rowShouldExist(clusterName)
    cy.wait(700)
    commonPageMethods.resourceTable.openRowMenu(clusterName)
    cy.get(managedClustersSelectors.clusterTableRowOptionsMenu.detachCluster, { timeout: 5000 }).click()
    commonPageMethods.modal.shouldBeOpen()
    commonPageMethods.modal.confirmAction(clusterName)
    commonPageMethods.modal.clickDanger('Detach')
    managedClustersMethods.removeStatus(clusterName, 'Detach')
  },

  detachClusterByLabel: (label) => {
    acm23xheaderMethods.goToClusters()
    commonPageMethods.resourceTable.searchTable(label)
    // pass if nothing found
    // get all cluster names to verify after
    cy.wait(700).get('#select-all').click()
    cy.get(managedClustersSelectors.actionsDropdown).should('exist').click()
    cy.get('#detachCluster').click()
    commonPageMethods.modal.shouldBeOpen()
    commonPageMethods.modal.confirmAction('confirm')
    cy.get('button[type="submit"]', { timeout: 2000 }).click()
    // verify all clusters with owner=clc-automation are in detached mode or gone (shows 'No Result Found')
    // verify backend
  },

  checkCluster: (clusterName, clusterStatus) => {
    acm23xheaderMethods.goToClusters()
    commonPageMethods.resourceTable.rowShouldExist(clusterName)
    cy.get(managedClustersSelectors.clusterTableColumnFields.status).find('button').should('contain', clusterStatus);
  },

  clusterLabelExists: (clusterName, labelName, labelVal) => {
    cy.log("Check the managed cluster labels for " + labelName + "=" + labelVal);
    cy.waitUntil(() => {
      return managedCluster.getManagedCluster(clusterName).then(resp => {
        var exist = false
        if (resp.body.metadata.labels[labelName] === labelVal) {
          exist = true
        }
        return exist
      })
    },
    {
      errorMsg: "Can not find the label " + labelName + " in managed cluster " + clusterName + "\n",
      interval: 2 * 1000,
      timeout: 2000 * 1000
    })
  },

  vendorLabelExists: (clusterName, vendor) => {
    cy.log("Checking if vendor label exists and is correct")
    managedClustersMethods.clusterLabelExists(clusterName, "vendor", vendor)
  },

  ownerLabelExists: (clusterName, owner) => {
    cy.log("Checking if owner label exists and is correct")
      managedClustersMethods.clusterLabelExists(clusterName, "owner", owner)
  },

  checkClusterImportStatus: (clusterName) => {
    cy.log("Check the cluster's import status");
    cy.waitUntil(() => {
      return managedCluster.getManagedCluster(clusterName).then(resp => {
        var exist = false
        for (let i = 0; i < resp.body['status']['conditions'].length; i++) {
          var condition = resp.body['status']['conditions'][i]
          if (condition.type === "ManagedClusterConditionAvailable" && condition.status == "True") {
            exist = true
          }
        }
        return exist
      })
    },
    {
      interval: 2 * 1000,
      timeout: 1800 * 1000
    })
  },

  checkClusterStatus: (clusterName) => {
    if (clusterActions.checkClusterStatus(clusterName)) {
      acm23xheaderMethods.goToClusters();
      commonPageMethods.resourceTable.rowShouldExist(clusterName)
      cy.get(`tr[data-ouia-component-id="${clusterName}"] td[data-label="Status"] > span > div > span > button`, { timeout: 300000 }) // 5 minutes for it to appear
        .should('have.text', clusterStatus.READY)
    }
  },

  /**
   * Using oc command, export a cluster's kubeconfig as an asset
   * @param {E} clusterName 
   */
  extractClusterKubeconfig: (clusterName) => {
    cy.ocLogin(Cypress.env('token'), constants.apiUrl).then(() => {
      cy.ocExtractClusterKubeconfig(clusterName)
    })
  },

  removeStatus: (name, type = '') => {
    const time = Cypress.env(`${type.toUpperCase()}_TIMEOUT`) || 20
    const maxTimeout = Date.now() + (time * 60 * 1000)

    function poll() {
      cy.get('.pf-c-page').then(page => {
        if (page.find('table').length > 0 && page.find(`tr[data-ouia-component-id="${name}"]`).length > 0) {
          cy.get(`tr[data-ouia-component-id="${name}"] td[data-label="Status"]`, { timeout: 30000 }).invoke('text').then({ timeout: time * 60 * 1000 }, status => {
            if (Date.now() > maxTimeout) {
              cy.log('Timed out waiting for cluster (15m)')
              type === 'detach' && expect(status).to.eq('Detached')
              type === 'destroy' && cy.get(`tr[data-ouia-component-id="${name}"]`).should('not.exist')
            } else {
              switch (status) {
                case 'Detached':
                  return expect(status).to.eq('Detached')
                case '':
                case undefined:
                case null:
                  return cy.get(`tr[data-row-name="${name}"]`).should('not.exist')
                case 'Detaching':
                case 'Destroying':
                default:
                  cy.log(`${name}'s status is still ${status}, waiting...`)
                  cy.wait(15 * 1000)
                  return poll()
              }
            }
          })
        } else {
          return cy.get(`tr[data-ouia-component-id="${name}"]`).should('not.exist')
        }
      })
    }
    return poll()
  },  
  createSNOCluster: (providerType, releaseImage, jobTemplateName) => {
    let connection = options.connections.apiKeys[`${providerType}`];
    let cluster = options.clusters[`${providerType}`];
    let clusterName = `${cluster.clusterName}-sno-${Date.now()}`;
    cy.visit('/multicloud/clusters');
    cy.wait(5 * 1000).contains('button','Create cluster').should($el => {
      expect(Cypress.dom.isDetached($el)).to.eq(false)
    }).click();
    managedClustersMethods.fillInfrastructureProviderDetails(connection.provider, connection.name);
    managedClustersMethods.fillClusterDetails(clusterName, '', releaseImage, cluster.additionalLabels + ',clc-cluster-type=qe-sno', true);
    managedClustersMethods.fillMasterNodeDetails(cluster.region);
    managedClustersMethods.fillWorkerPoolDetails();
    managedClustersMethods.fillNetworkingDetails();
    managedClustersMethods.fillAutomationDetails(jobTemplateName);
    managedClustersMethods.clickCreateAndVerify(clusterName);
    cy.contains('h1', clusterName, { timeout: 5000 }).should('exist');
    return cy.wrap(clusterName);
  },
  destroySNOClusters: () => {
    cy.visit('/multicloud/clusters')
    managedCluster.getAllSNOClusters().then ( list => {
      if (list != null) {
        list.forEach(clusterName => {
          commonPageMethods.resourceTable.searchTable(clusterName.split('-').pop())
          commonPageMethods.resourceTable.openRowMenu(clusterName)
          cy.get(managedClustersSelectors.clusterTableRowOptionsMenu.destroyCluster).click()
          commonPageMethods.modal.shouldBeOpen()
          commonPageMethods.modal.confirmAction(clusterName)
          commonPageMethods.modal.clickDanger('Destroy')
          managedClustersUIValidations.validateManagedClusterDestroyed(clusterName)
        })
      }
    })
  },
}

export const managedClustersUIValidations = {
  validateManagedClusterInstalled: (clusterName) => {
    managedCluster.checkClusterDeployment(clusterName)
    managedCluster.checkManagedClusterInfoStatus(clusterName)
    cy.wait(5 * 1000)
    cy.visit(`/multicloud/clusters/${clusterName}/overview`)
    return cy.get('dt:contains("Status")').next().invoke('text').then( txt => {
      expect(txt).to.be.oneOf(['Importing','Posthook','Ready'])
    })
  },
  validateAddons: (clusterName) => {
    cy.visit(`/multicloud/clusters/${clusterName}/settings`)
    commonPageMethods.resourceTable.rowCount().then(counts => {
      function getAddonsStatus() {
        let arr = [];
        cy.get('td[data-label="Status"]').each(el => {
          cy.get(el).invoke('text').then(txt => {
            arr.push(txt)
          })
        })
        return cy.wrap(arr)
      }
      // timeout = 15 mins, polling every 30s
      genericFunctions.recurse(
        () => getAddonsStatus(),
        (status) => !(status.includes('Progressing') || status.includes('Unknown')),
        30,
        30 * 1000
      )
      cy.get('td[data-label="Status"]').each(el => cy.get(el).should("have.text", "Available"))
    });
  },
  validateNodes: (clusterName,) => {
    cy.visit(`/multicloud/clusters/${clusterName}/nodes`)
    commonPageMethods.resourceTable.rowCount().then(counts => {
      expect(counts).to.be.at.least(1)
    })
  },
  validateMachinePools: (clusterName) => {
    cy.visit(`/multicloud/clusters/${clusterName}/machinepools`)
    commonPageMethods.resourceTable.rowCount().then(counts => {
      expect(counts).to.be.at.least(1)
    })
  },
  validateManagedClusterDestroyed: (clusterName) => {
    managedCluster.checkClusterDeploymentDeleted(clusterName)
    managedCluster.checkManagedClusterInfoDeleted(clusterName)
    cy.wait(30 * 1000)
    acm23xheaderMethods.goToClusters()
    commonPageMethods.resourceTable.searchTable(clusterName.split('-').pop());
    commonPageMethods.resourceTable.rowCount().then(counts => {
      expect(counts).to.eq(0)
    })
  },
  validateAnsibleJobHooks: (clusterName, stage) => {
    let selector = stage.toLowerCase() == 'prehook' ? 'Prehook' : 'Posthook'
    automation.checkAnsibleJobResult(clusterName, stage)
    cy.visit(`/multicloud/clusters/${clusterName}/overview`)
    cy.get('dt:contains("Status")').next().invoke('text').then( txt => {
      if ( txt != 'Ready') {
        cy.contains('div', selector).next().invoke('text').then( txt => 
          expect(txt).to.be.oneOf(['In progress','Complete'])
        )
      }
    })
  }
}
