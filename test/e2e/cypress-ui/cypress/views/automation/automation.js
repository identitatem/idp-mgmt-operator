/* Copyright Red Hat */

/// <reference types="cypress" />
import { genericFunctions } from '../../support/genericFunctions';
import { commonElementSelectors } from '../common/commonSelectors';
import { commonPageMethods } from '../common/commonSelectors';
import { getClusterCurator } from '../../apis/automation'


const ansibleJobData = require('../../fixtures/automation/automationTestData').ansibleTestData
const autoTemplateName = 'clc-ansible-template-auto'

export const automationPageSelectors ={

    createAnsibleTemplate : 'Create Ansible template',
    pageText : {
        noAnsibleJobTemplates : "You don't have any Ansible job templates"
    },
    basicInformation : {
        templateName: '#Template',
        credentialFormGroup : 'ansibleSecrets-form-group',
        ansibleCredentialInput : '#ansibleSecrets-input-toggle-select-typeahead',
        ansibleCredentialInputToggle : '#ansibleSecrets-input-toggle'
    },
    ansibleJobTemplates: {
        addAnsibleJobTemplateButton: 'Add an Ansible job template',
        install: {
            preInstallTemplateGroup: '#installPreJob-form-group',
            postInstallTemplateGroup: '#installPostJob-form-group'
        },
        upgrade: {
            preUpgradeTemplateGroup: '#upgradePreJob-form-group',
            postUpgradeTemplateGroup: '#upgradePostJob-form-group'
        },
        addAnsiblejob: {
            jobName: '#job-name',
            jobSettings: '#job-settings'
        }
        
    },
    review: {
        name: 'dt:contains("Template name")',
        credential: 'dt:contains("Ansible Automation Platform credential")',
        install:{
            preJobTemplateName: 'dt:contains("Pre-install Ansible job templates")',
            postJobTemplateName: 'dt:contains("Post-install Ansible job templates")'
        },
        upgrade: {
            preJobTemplateName: 'dt:contains("Pre-upgrade Ansible job templates")',
            postJobTemplateName: 'dt:contains("Post-upgrade Ansible job templates")'
        }
    }
}

export const automationPageMethods = {
    gotoAutomationPage: () =>{
        cy.visit('multicloud/ansible-automations')
    },
    isJobTemplateEmpty: () =>{
       var isTemplateEmpty = genericFunctions.isEmptyPage(automationPageSelectors.pageText.noAnsibleJobTemplates);
       return isTemplateEmpty;
    },
    clickCreateJobTemplate:()=>{
        cy.get(`button:contains(${automationPageSelectors.createAnsibleTemplate})`).click()
        cy.log('"Create Ansible template" button clicked!')
    },
    /**
     * This function accepts Template name and Credential value as inputs
     * @param {*} templateName
     * @param {*} credential
     */
    fillUpBasicInformation: (templateName, credential) => {
        cy.get(automationPageSelectors.basicInformation.templateName).click().type(templateName);
        cy.get(automationPageSelectors.basicInformation.ansibleCredentialInputToggle).click();
        cy.get('button').contains(new RegExp("^" + credential + "$", "g")).click(); //click exact match
        // cy.get(automationPageSelectors.basicInformation.ansibleCredentialInput).click().type(credential)
        // genericFunctions.selectOrTypeInInputDropDown(automationPageSelectors.basicInformation.credentialFormGroup, credential);
        // cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click();
    },
    /**
     * This method accepts flags, isPre,isPost as true or false for pre/post job templates, isInstall,isUpgrade as true or false for install/upgrade.
     * @param {boolean} templatePhase  
     */
     clickAddAnsiblejobTemplate: (templatePhase) => {
         switch(templatePhase){
             case 'preInstall':{
                cy.get(automationPageSelectors.ansibleJobTemplates.install.preInstallTemplateGroup).children().eq('1').find('button').click();
                break;
             }
             case 'postInstall': {
                cy.get(automationPageSelectors.ansibleJobTemplates.install.postInstallTemplateGroup).children().eq('1').find('button').click();
                break;
             }
             case 'preUpgrade': {
                cy.get(automationPageSelectors.ansibleJobTemplates.upgrade.preUpgradeTemplateGroup).children().eq('1').find('button').click();
                break;
             }
             case 'postUpgrade': {
                cy.get(automationPageSelectors.ansibleJobTemplates.upgrade.postUpgradeTemplateGroup).children().eq('1').find('button').click();
                break;
             }
             default:{
                cy.log('invalid option passed');
             } 
         }
    },
    /**
     * This method accepts jobTemplateName and extraVariables values
     * @param {*} jobTemplateName
     * @param {*} extraVariables
     */
    addAnsiblejob: (jobTemplateName, extraVariables) => {
        cy.get('input[placeholder="Enter or select Ansible job template name"]').click();
        cy.get(`button:contains(${jobTemplateName})`).click();
   },
    selectRunPhase: (phase) => {
        phase.toLowerCase() === "install" ? cy.get('button:contains("Install")').last().click():
                                            cy.get('button:contains("Upgrade")').last().click();
    },
    launchAddAnsibleJobFrame: (stage) => {
        stage.toLowerCase().includes("pre-") ?
        cy.get(`button:contains(${automationPageSelectors.ansibleJobTemplates.addAnsibleJobTemplateButton})`)
            .first().click():
        cy.get(`button:contains(${automationPageSelectors.ansibleJobTemplates.addAnsibleJobTemplateButton})`)
            .last().click()
    },
    clickNext: () => {
        cy.get(`button:contains(${commonElementSelectors.elementsText.next})`).click();
    },
    clickSave: () => {
        cy.get(`button:contains(${commonElementSelectors.elementsText.save})`).click();
    },
    clickAdd: () => {
        cy.get(`button:contains(${commonElementSelectors.elementsText.add})`).click();
    }
}

export const uiValidations = {
    validateTemplateAdded: (name, stage) => {
        switch(stage){
            case 'pre-install':{
               cy.get(`${automationPageSelectors.ansibleJobTemplates.install.preInstallTemplateGroup}`).as("group")
               break;
            }
            case 'post-install': {
                cy.get(`${automationPageSelectors.ansibleJobTemplates.install.postInstallTemplateGroup}`).as("group")
              break;
            }
            case 'pre-upgrade': {
                cy.get(`${automationPageSelectors.ansibleJobTemplates.upgrade.preUpgradeTemplateGroup}`).as("group")
              break;
            }
            case 'post-upgrade': {
                cy.get(`${automationPageSelectors.ansibleJobTemplates.upgrade.postUpgradeTemplateGroup}`).as("group")
              break;
            }
        }
        cy.get("@group")
            .within( () =>{
                cy.get(`#${name}`).should('exist');
            })
    },
    validateReviewPage: (name, credName, ansibleTemplateName,phase) => {
        cy.get(automationPageSelectors.review.name).next().should('have.text',name);
        cy.get(automationPageSelectors.review.credential).next().should('have.text',credName);
        if (phase === "install") {
            cy.get(automationPageSelectors.review.install.preJobTemplateName).next().should('have.text',ansibleTemplateName);
            cy.get(automationPageSelectors.review.install.postJobTemplateName).next().should('have.text',ansibleTemplateName);
        }
        else {
            cy.get(automationPageSelectors.review.upgrade.preJobTemplateName).next().should('have.text',ansibleTemplateName);
            cy.get(automationPageSelectors.review.upgrade.postJobTemplateName).next().should('have.text',ansibleTemplateName);
        }
    },
    validateAutomationRecord: (name) => {
        commonPageMethods.resourceTable.rowCount().then( count => {
            expect(count).to.be.at.least(1)
        })
        commonPageMethods.resourceTable.rowShouldExist(name.split('-').pop());
    }
}

export const createAnsibleTemplate = ({name},phase) => {
    let templateName = `${autoTemplateName}-${phase}`
    automationPageMethods.gotoAutomationPage();
    automationPageMethods.clickCreateJobTemplate();
    automationPageMethods.fillUpBasicInformation(templateName, name);
    automationPageMethods.clickNext();
    automationPageMethods.selectRunPhase(phase);
    // add pre template
    automationPageMethods.launchAddAnsibleJobFrame(`pre-${phase}`);
    automationPageMethods.addAnsiblejob(ansibleJobData.templateName);
    automationPageMethods.clickSave();
    uiValidations.validateTemplateAdded(ansibleJobData.templateName,`pre-${phase}`);
    // add post template
    automationPageMethods.launchAddAnsibleJobFrame(`post-${phase}`);
    automationPageMethods.addAnsiblejob(ansibleJobData.templateName);
    automationPageMethods.clickSave();
    uiValidations.validateTemplateAdded(ansibleJobData.templateName,`post-${phase}`);
    // next
    if (phase === "install")
        automationPageMethods.clickNext();
    automationPageMethods.clickNext();
    // overview
    uiValidations.validateReviewPage(templateName,name,ansibleJobData.templateName,phase);
    // add
    automationPageMethods.clickAdd(); 
}

export const validateAnsibleTemplateCreation = ({namespace},phase) => {
    let templateName = `${autoTemplateName}-${phase}`
    automationPageMethods.gotoAutomationPage();
    getClusterCurator(templateName,namespace)
    .then( statusCode => {
        expect(statusCode).to.eq(200)
    });
    uiValidations.validateAutomationRecord(templateName,phase);
}