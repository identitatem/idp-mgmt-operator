# IDP Cypress Tests

## Running the tests locally

  > Pre-requisites:
  >  - nodeJS
  >  - OpenShift CLI (oc)
  >  - a browser - either chrome or firefox

1. Run `npm install` to install Cypress and it's dependencies on the root directory.
2. Export the following environment variables:
    - `export CYPRESS_BASE_URL=http://localhost:3001`
    - `export CYPRESS_OPTIONS_HUB_USER=kubeadmin`
    - `export CYPRESS_OPTIONS_HUB_PASSWORD=xxxxxxxxx`
    - `export CYPRESS_BASE_URL=https://multicloud-console.apps.{clusterName}.dev06.red-chesterfield.com` (URL of a working cluster)
3. Run `npx cypress open` to run your test in headed mode.
4. Select test to run.

### Running in Headless Mode
- If you want to run in headless mode and tag a specific test, instead of doing `npx cypress open`, try this instead:
    - `npx cypress run --headless --reporter cypress-multi-reporters --env grepTags=<test-tag>,grepFilterSpecs=true,grepOmitFiltered=true`
        - `<test-tag>` represents test tags you want target. For example, you can target the sample test with `grepTags=login` or multiple tests/suites with `grepTags=login+@IDP`

## Links

These are a few useful links that will help provide technical reference and best practices when developing for the platform.

- [Cypress Docs](https://docs.cypress.io/guides/overview/why-cypress.html)
- [NPM Docs](https://docs.npmjs.com)