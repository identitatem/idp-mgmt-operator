/* Copyright Red Hat */
/// <reference types="cypress" />
import { commonElementSelectors } from '../views/common/commonSelectors'
export const genericFunctions = {
    checkIfElementExists: (element) => {
        return cy.wait(500).get('body').then($body => {
            return cy.wrap($body.text().includes(element))
        })

        // var elementFound = false;
        // cy.wait(500).get('body').then($body => {
        //     // cy.log("dhruval")
        //     // cy.log($body.text())
        //     if ($body.text().includes(element)) {
        //         elementFound = true;
        //         cy.log(element + 'found in the body');
        //     }
        // })
        // cy.log(elementFound)
        // return elementFound;
    },

    /**
     * This functions accepts Div id of the group that contains the dropdown, value to be selected from dropdown.
     * If you want to type text in input dropdown then pass typeText = true
     * @param {*} divId 
     * @param {*} valueToBeSelected 
     * @param {*} typeText 
     * @param {*} isNotInput 
     */
    selectOrTypeInInputDropDown: (divId, valueToBeSelected, typeText) => {
        cy.get('#' + divId).find(commonElementSelectors.elements.input).click();
        if (typeText) {
            cy.get('#' + divId).find(commonElementSelectors.elements.input).type(valueToBeSelected).type('{enter}');
        } else {
            cy.contains(commonElementSelectors.elements.selectMenuItem, valueToBeSelected).click();
        }
    },

    /**
     * This function accepts text to verify if a page is empty or not
     * for eg you can pass "You don't have any credentials." to verify that there are no credentians on credentials page
     * @param {*} emptyText 
     */
    isEmptyPage: (emptyText) => {
        cy.get('.pf-c-page__main-section').then(($body) => {
            if ($body.text().includes(emptyText)) {
                cy.log("No value found hence, returning true");
                return true;
            } else {
                cy.log("value present hence, returning false");
                return false;
            }
        })
    },

    clickNext: () => {
        cy.contains(commonElementSelectors.elements.button, commonElementSelectors.elementsText.next).click()
    },

    compareDropdownValues: (listId, valuesToBeCompared) => {
        // cy.get(listId).find('option').then(options =>{
        //     const actual = [...options].map(o => o.value);
        //     expect(actual).to.deep.eq(valuesToBeCompared);
        // })
        cy.get(listId).find('li').each((item, index) => {
            // cy.log(cy.wrap(item))
            cy.wrap(item).should('contain.text', valuesToBeCompared[index])
        })
    },

    /**
     * This method accepts text or id of the element and clicks on that element
     * @param {*} textOrId 
     */
    clickButton: (textOrId) => {
        cy.contains('button', textOrId).click()
    },

    /**
     * accepts provider values and test case cloud provider type
     * @param {*} provider 
     * @param {*} testcaseProvider
     * @returns it or it.only based on the condition to execute only one test case
     */
    itOnly: (provider, testcaseProvider) => {
        console.log('inside skipif' + provider)
        // for(ach in provider){
        if (provider == 'aws' && testcaseProvider == 'aws') {
            return it.only;
        } else if (provider == 'gcp' && testcaseProvider == 'gcp') {
            return it.only;
        } else if (provider == 'azure' && testcaseProvider == 'azure') {
            return it.only;
        } else if (provider == 'vmware' && testcaseProvider == 'vmware') {
            return it.only;
        } else if (provider == 'ansible' && testcaseProvider == 'ansible') {
            return it.only;
        } else if (provider != 'all' && testcaseProvider == 'clusterStatus') {
            return it.only;
        } else {
            return it;
        }
    },

    getProviderFromTags: (tags) => {
        if (tags.includes('aws')) {
            return 'aws';
        } else if (tags.includes('gcp')) {
            return 'gcp';
        } else if (tags.includes('azure')) {
            return 'azure';
        } else if (tags.includes('vmware')) {
            return 'vmware';
        } else if (tags.includes('all')) {
            return 'all';
        }
    },

    /**
 * Filter Cypress tests based on a given tag or tags. If no tags are present, run tests.
 *
 * @param {[string]} definedTags An array of tags
 * @param {Function} runTest All tests captured within a Cypress run
 * @example npm run open --env suiteTags=api
 * @example npm run open --env suiteTags=api/ui
 */
    filterTests: (definedTags, runTest) => {
        if (Cypress.env('suiteTags')) {
            const suiteTags = Cypress.env('suiteTags').split('/');
            const found = definedTags.some(($definedTag) => suiteTags.includes($definedTag));

            if (found) {
                runTest();
            }
        } else {
            runTest();
        }
    },

    /***
     * @param {String} ansibleHost ansible tower url
     * @param {String} ansibleToken ansible tower token
     */
    fetchAnsibleJobList: (ansibleHost, ansibleToken) => {
        cy.log('dhruval' + ansibleHost)
        var jobTemplates = [];
        cy.request({
            method: 'GET',
            url: ansibleHost+'/api/v2/job_templates/',
            json: true,
            headers: {
                Authorization: 'Bearer ' + ansibleToken
            }
        }).then((response) => {
            
            var jobTemplateCount = response.body['count']
            cy.log(jobTemplateCount)
            for (var i = 0; i < jobTemplateCount; i++) {
                jobTemplates.push((response.body['results'][i]['name']))
                cy.log(response.body['results'][i]['name'])
            }
        });
        return jobTemplates;
    }
    // }
    ,
    recurse: (commandFn, checkFn, limit = 10, interval = 30 * 1000) => {
        if (limit < 0) {
            throw new Error('Recursion limit reached')
        }
        cy.log(`**${commandFn}** remaining attempts **${limit}**`)
        return commandFn().then( (x) => {
            if (checkFn(x)) {
              cy.log( `**${commandFn}** completed` )
              return
            }
            cy.wait(interval);
            genericFunctions.recurse(commandFn, checkFn, limit - 1, interval)
        })
    }
}
