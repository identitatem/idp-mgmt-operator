/* Copyright Red Hat */
/// <reference types="Cypress" />

/**
 * Filter Cypress tests based on a given tag or tags. If no tags are present, run tests.
 *
 * @param {[string]} definedTags An array of tags
 * @param {Function} runTest All tests captured within a Cypress run
 * @example npm run open --env suiteTags=api
 * @example npm run open --env suiteTags=api/ui
 */
 const filterTests = (definedTags, runTest) => {
    if (Cypress.env('suiteTags')) {
      const suiteTags = Cypress.env('suiteTags').split('/');
      const found = definedTags.some(($definedTag) => suiteTags.includes($definedTag));
  
      if (found) {
        runTest();
      }
    } else {
      runTest();
    }
  };
  
  export default filterTests;