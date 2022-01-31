/* Copyright Red Hat */

/// <reference types="cypress" />
import { commonElementSelectors } from '../common/commonSelectors'
import { genericFunctions } from '../../support/genericFunctions'
import { acm23xheaderMethods} from '../header'
import * as credentialAPI from '../../apis/credentials';

export const credentialsPageSelectors = {
    credentialsTypeLocator: {
        aws: "#aws",
        azr: '#azr',
        gcp: '#gcp',
        vmw: '#vmw',
        ans: '#ans',
        ost: '#ost'
    },
    search : 'input[aria-label="Search input"]',
    resetSearch : 'button[aria-label="Reset"]',
    credentialsName: "#credentialsName",
    namespaceDropdownToggle: 'button[aria-label="Options menu"]',
    actionsDropdown : '#toggle-id',
    selectMenuItem: ".pf-c-select__menu-item",
    credentialsTypesInputSelectors: {
        aws: {
            awsAccessKeyID: '#aws_access_key_id',
            awsSecretAccessKeyID: "#aws_secret_access_key"
        },
        gcp: {
            gcProjectID: '#projectID',
            gcServiceAccountKey: 'textarea' //text area element tag
        },
        azr: {
            baseDomainResourceGroupName: '#baseDomainResourceGroupName',
            clientId: '#clientId',
            clientSecret: '#clientSecret',
            subscriptionId: '#subscriptionId',
            tenantId: '#tenantId'
        },
        vmw: {
            vCenterCredentials: {
                vcenter: '#vCenter',
                username: '#username',
                password: '#password',
                cacertificate: '#cacertificate'
            },
            vSphereCredentials: {
                vmClusterName: '#cluster',
                datacenter: '#datacenter',
                datastore: '#defaultDatastore'
            }
        },
        ans: {
            ansibleHost: '#ansibleHost',
            ansibleToken: '#ansibleToken'
        },
        ost: {
            openstackCloudsYaml: 'textarea',
            openstackCloudName: '#cloud'
        }
    },
    baseDomain: '#baseDomain',
    commonCredentials: {
        pullSecret: '#pullSecret',
        sshPrivatekey: '#ssh-privatekey',
        sshPublicKey: '#ssh-publickey'
    },
    elementText: {
        deleteCredentials : 'Delete credentials',
        deleteCredentialsConfirmation: 'Permanently delete credentials?'
    },
    tableColumnFields : {
        name : '[data-label="Name"]',
        credentialType : '[data-label="Credential type"]',
        namespace : '[data-label="Namespace"]',
        additionalActions : '[data-label="Additional actions"]',
        created : '[data-label="Created"]'
    }
}

export const credentialsPageMethods = {
  
    whenAddProviderConnectionAction: () => {
        // var noCredentials = genericFunctions.checkIfElementExists("Add credentials button to create")
        // cy.log(noCredentials)
        cy.get('.pf-c-page__main-section').then(($body) => {
            if($body.text().includes("You don't have any credentials.")){
                cy.contains('a','Add credential').click();
            }else{
                cy.contains('Add credential').click() ;
            }
        })
    },
    shouldLoad: () => {
        cy.get('.pf-c-page').should('contain', 'Credentials');
        cy.get('.pf-c-spinner', { timeout: 20000 }).should('not.exist');
    },
    fillCommonCredentials: (pullSecret, sshPrivatekey, sshPublickey) => {
        cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click();
        cy.get(credentialsPageSelectors.commonCredentials.pullSecret).should('exist').paste(pullSecret);
        // cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click();
        cy.get(credentialsPageSelectors.commonCredentials.sshPrivatekey).scrollIntoView().should('exist').paste(sshPrivatekey);
        cy.get(credentialsPageSelectors.commonCredentials.sshPublicKey).scrollIntoView().should('exist').paste(sshPublickey);
        cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click();
    },
    fillBasicInformation: (name, namespace, baseDomain) => {
        cy.get(credentialsPageSelectors.credentialsName).should('exist').click().type(name);
        genericFunctions.selectOrTypeInInputDropDown('namespaceName-form-group',namespace);
        if(baseDomain){
            cy.get(credentialsPageSelectors.baseDomain).should('exist').click().type(baseDomain);
        }
    },
    verifyCredential: (name) => {
        cy.get(credentialsPageSelectors.search, { timeout: 15000 }).click().type(name);
        cy.wait(1500).get(commonElementSelectors.elements.a).contains(name).should('exist');
        cy.get(credentialsPageSelectors.resetSearch).click();
    },
    addAWSProviderCredential: ({name, namespace, baseDnsDomain, awsAccessKeyID, awsSecretAccessKeyID,  pullSecret, sshPrivatekey, sshPublickey}) => {
        acm23xheaderMethods.gotoCredentials();
        credentialAPI.getCredential(name, namespace).then(resp => {
            // When the secret was not exists, create it.
            if (resp == 404) {
                credentialsPageMethods.whenAddProviderConnectionAction();
                cy.get(credentialsPageSelectors.credentialsTypeLocator.aws,{ timeout: 15000 }).click();
                credentialsPageMethods.fillBasicInformation(name, namespace, baseDnsDomain);
                cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click();
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.aws.awsAccessKeyID).click().type(awsAccessKeyID);
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.aws.awsSecretAccessKeyID).should('exist').click().type(awsSecretAccessKeyID);
                credentialsPageMethods.fillCommonCredentials(pullSecret, sshPrivatekey, sshPublickey);
                cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.add).click();
                cy.wait(1000);
            }
            credentialsPageMethods.verifyCredential(name);
        })
    },
    addGCPproviderCredential: ({name, namespace, baseDnsDomain, gcpProjectID, gcpServiceAccountJsonKey, pullSecret, sshPrivatekey, sshPublickey}) => {
        acm23xheaderMethods.gotoCredentials()
        credentialAPI.getCredential(name, namespace).then(resp => {
            if (resp == 404 ) {
                credentialsPageMethods.whenAddProviderConnectionAction()
                cy.get(credentialsPageSelectors.credentialsTypeLocator.gcp,{ timeout: 15000 }).click();
                credentialsPageMethods.fillBasicInformation(name, namespace, baseDnsDomain);
                cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click();
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.gcp.gcProjectID).click().type(gcpProjectID);
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.gcp.gcServiceAccountKey).should('exist').paste(gcpServiceAccountJsonKey);
                credentialsPageMethods.fillCommonCredentials(pullSecret, sshPrivatekey, sshPublickey);
                cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.add).click();
                cy.wait(1000);
            }
            credentialsPageMethods.verifyCredential(name);
        })
    },
    addAZUREproviderCredential: ({name, namespace, baseDnsDomain, baseDomainResourceGroupName, clientID, clientSecret, subscriptionID, tenantID, pullSecret, sshPrivatekey, sshPublickey}, cloudName) => {
        acm23xheaderMethods.gotoCredentials();
        credentialAPI.getCredential(name, namespace).then(resp => {
            if (resp == 404 ) {
                credentialsPageMethods.whenAddProviderConnectionAction();
                cy.get(credentialsPageSelectors.credentialsTypeLocator.azr, { timeout: 15000 }).click();
                credentialsPageMethods.fillBasicInformation(name, namespace, baseDnsDomain);
                if(cloudName){
                    genericFunctions.selectOrTypeInInputDropDown('azureCloudName-form-group',cloudName);
                }
                cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click();
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.azr.baseDomainResourceGroupName).should('exist').click().type(baseDomainResourceGroupName);
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.azr.clientId).click().type(clientID);
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.azr.clientSecret).click().type(clientSecret);
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.azr.subscriptionId).click().type(subscriptionID);
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.azr.tenantId).click().type(tenantID);
                credentialsPageMethods.fillCommonCredentials(pullSecret, sshPrivatekey, sshPublickey);
                cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.add).click();
                cy.wait(1000);
            }
        credentialsPageMethods.verifyCredential(name);
        })
    },
    addVMwareproviderCredential: ({name, namespace, baseDnsDomain, vcenterServer, username, password, cacertificate, vmClusterName, datacenter, datastore, pullSecret, sshPrivatekey, sshPublickey}) => {
        acm23xheaderMethods.gotoCredentials();
        credentialAPI.getCredential(name, namespace).then(resp => {
            if (resp == 404 ) {
                credentialsPageMethods.whenAddProviderConnectionAction();
                cy.get(credentialsPageSelectors.credentialsTypeLocator.vmw, { timeout: 15000 }).click();
                credentialsPageMethods.fillBasicInformation(name, namespace, baseDnsDomain);
                cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click();
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.vmw.vCenterCredentials.vcenter).click().type(vcenterServer);
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.vmw.vCenterCredentials.username).click().type(username);
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.vmw.vCenterCredentials.password).click().type(password);
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.vmw.vCenterCredentials.cacertificate).paste(cacertificate);
                cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click();
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.vmw.vSphereCredentials.vmClusterName).click().type(vmClusterName);
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.vmw.vSphereCredentials.datacenter).click().type(datacenter);
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.vmw.vSphereCredentials.datastore).click().type(datastore);
                credentialsPageMethods.fillCommonCredentials(pullSecret, sshPrivatekey, sshPublickey);
                cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.add).click();
                cy.wait(1000);
            }        
        credentialsPageMethods.verifyCredential(name);
        })
    },
    addAnsibleTowerCredential: ({name, namespace, ansibleHost, ansibleToken}) => {
        acm23xheaderMethods.gotoCredentials();
        credentialAPI.getCredential(name, namespace).then(resp => {
            if (resp == 404 ) {
                credentialsPageMethods.whenAddProviderConnectionAction();
                cy.get(credentialsPageSelectors.credentialsTypeLocator.ans, { timeout: 15000 }).click();
                credentialsPageMethods.fillBasicInformation(name, namespace,"",);
                cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click();
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.ans.ansibleHost).click().type(ansibleHost);
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.ans.ansibleToken).click().type(ansibleToken);
                cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click();
                cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.add).click();
                cy.wait(1000);
            }
        credentialsPageMethods.verifyCredential(name);
        })
    },

    addOpenStackproviderCredential: ({name, namespace, baseDnsDomain, cloudsFile, cloudName, pullSecret, sshPrivatekey, sshPublickey}) => {
        acm23xheaderMethods.gotoCredentials();
        credentialAPI.getCredential(name, namespace).then(resp => {
            if (resp == 404 ) {
                credentialsPageMethods.whenAddProviderConnectionAction();
                cy.get(credentialsPageSelectors.credentialsTypeLocator.ost, { timeout: 15000 }).click();
                credentialsPageMethods.fillBasicInformation(name, namespace, baseDnsDomain);
                cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click();
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.ost.openstackCloudsYaml).should('exist').paste(cloudsFile);
                cy.get(credentialsPageSelectors.credentialsTypesInputSelectors.ost.openstackCloudName).click().type(cloudName);
                credentialsPageMethods.fillCommonCredentials(pullSecret, sshPrivatekey, sshPublickey);
                cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.add).click();
                cy.wait(1000);
            }
        credentialsPageMethods.verifyCredential(name);
        })
    },

    deleteCredential: (connectionName) => {
        acm23xheaderMethods.gotoCredentials();
        cy.get(credentialsPageSelectors.search).click().type(connectionName)
        if(connectionName == "all" || connectionName == "All"){
            cy.get(commonElementSelectors.elements.checkAll).check().should('be.checked');
        }else{
            cy.get(credentialsPageSelectors.tableColumnFields.name).contains(commonElementSelectors.elements.a,connectionName).parent().parent().prev().find('input').check().should('be.checked');
        }
        cy.get(credentialsPageSelectors.actionsDropdown).should('exist').click();
        cy.get(commonElementSelectors.elements.a).contains(credentialsPageSelectors.elementText.deleteCredentials).should('exist').click();
        cy.get('div').should('be.visible').contains(credentialsPageSelectors.elementText.deleteCredentialsConfirmation);
        cy.get('div').should('be.visible').contains('button','Delete').click();
        cy.wait(1000);
        // cy.get('.pf-c-modal-box').should('not.exist')
        cy.get('.pf-c-page__main-section').then(($body) => {
            if($body.text().includes("You don't have any credentials.")){
                cy.log("No credentials found");
            }else{
                cy.wrap($body).find(credentialsPageSelectors.tableColumnFields.name).contains(connectionName).should('not.exist');
            }
        })
        cy.get(credentialsPageSelectors.resetSearch).click();
        // cy.get('[data-label="Name"]').contains(connectionName).should('not.exist')
    
    }
}