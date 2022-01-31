/** *****************************************************************************
 * Licensed Materials - Property of Red Hat, Inc.
 * Copyright (c) 2020 Red Hat, Inc.
 ****************************************************************************** */

 const fs = require('fs')
 const path = require('path')
 const jsYaml = require('js-yaml')
 const _ = require('lodash')
 
 
 // This process will be used to convert the options.yaml, that was originally built for
 // the E2E tests, into the config.e2e.yaml that is needed by console-ui tests.  The canary
 // tests are still using the original options.yaml format so this conversion is required
 // when canary tests are running.
 // See https://github.com/open-cluster-management/canary-scripts/blob/2.1-integration/utils.sh
 // where the canary builds the options.yaml
 
 const transformOptions = () => {
   try {
     const optionsPath = 'options.yaml'
     const optionsYaml = fs.readFileSync(optionsPath)
     const { options } = jsYaml.safeLoad(optionsYaml)
     const connections = getProviderConnection(options)
     const clusters = getCreateCluster(options)
     const credentials = getCredentials(options)
     const transformedOptions = { connections, clusters, credentials } //clusters
     const yaml = jsYaml.safeDump(transformedOptions)
     const configPath = path.join(__dirname, '..', 'config', 'config.e2e.yaml')
     return fs.writeFileSync(configPath, yaml)
   } catch(e) {
     // eslint-disable-next-line no-console
     console.error('Failed to transform options: ', e)
   }
 }
 
 const getProviderConnection = ({ cloudConnection }) => {
   const { apiKeys } = cloudConnection
 
   const azure = {
     name: _.get(apiKeys, 'azure.name', 'azure credentials name not provided'),
     namespace: _.get(apiKeys, 'azure.namespace', 'azure credential namespace not provided'),
     provider: 'Microsoft Azure',
     baseDnsDomain: _.get(apiKeys, 'azure.baseDnsDomain', ''),
     baseDomainResourceGroupName: _.get(apiKeys, 'azure.baseDomainResourceGroupName', 'Azure base domain resource group name not provided'),
     clientId: _.get(apiKeys, 'azure.clientID', 'Azure client id not provided'),
     clientSecret: _.get(apiKeys, 'azure.clientSecret', 'Azure client secret not provided'),
     subscriptionId: _.get(apiKeys, 'azure.subscriptionID', 'Azure subscription id not provided'),
     tenantID: _.get(apiKeys, 'azure.tenantID', 'Azure tenant id not provided'),
   }
   const aws = {
     name: _.get(apiKeys, 'aws.name', 'aws credentials name not provided'),
     namespace: _.get(apiKeys, 'aws.namespace', 'aws credential namespace not provided'),
     provider: 'Amazon Web Services',
     awsAccessKeyID: _.get(apiKeys, 'aws.awsAccessKeyID', 'AWS access key id not provided'),
     awsSecretAccessKeyID: _.get(apiKeys, 'aws.awsSecretAccessKeyID', 'AWS secret access key not provided'),
     baseDnsDomain: _.get(apiKeys, 'aws.baseDnsDomain', '')
   }
   const gcp = {
     name: _.get(apiKeys, 'gcp.name', 'gcp credentials name not provided'),
     namespace: _.get(apiKeys, 'gcp.namespace', 'gcp credential namespace not provided'),
     vendor: 'Google Cloud',
     gcpProjectID: _.get(apiKeys, 'gcp.gcpProjectID', 'GCP project id not provided'),
     gcpServiceAccountJsonKey: _.get(apiKeys, 'gcp.gcpServiceAccountJsonKey', 'GCP service account json key not provided'),
     baseDnsDomain: _.get(apiKeys, 'gcp.baseDnsDomain', '')
   }
   const vmware = {
     name: _.get(apiKeys, 'vmware.name', 'gcp credentials name not provided'),
     namespace: _.get(apiKeys, 'vmware.namespace', 'gcp credential namespace not provided'),
     provider: 'VMware vSphere',
     username: _.get(apiKeys, 'vmware.username', 'VMware vSphere user name not provided'),
     password: _.get(apiKeys, 'vmware.password', 'VMware vSphere password not provided'),
     vcenterServer: _.get(apiKeys, 'vmware.vcenterServer', 'VMware vSphere server not provided'),
     cacertificate: _.get(apiKeys, 'vmware.cacertificate', 'VMware vSphere CA certificate not provided'),
     vmClusterName: _.get(apiKeys, 'vmware.vmClusterName', 'VMware vSphere cluster name not provided'),
     datacenter: _.get(apiKeys, 'vmware.datacenter', 'VMware vSphere datacenter not provided'),
     datastore: _.get(apiKeys, 'vmware.datastore', 'VMware vSphere datastore not provided'),
     //folder: _.get(apiKeys, 'vmware.folder', 'VMware vSphere datastore folder not provided'),
     baseDnsDomain: _.get(apiKeys, 'vmware.baseDnsDomain', '')
   }
   const ansible = {
    name: _.get(apiKeys, 'ansible.name', 'ansible credentials name not provided'),
    namespace: _.get(apiKeys, 'ansible.namespace', 'ansible credential namespace not provided'),
    vendor: 'Ansible',
    ansibleHost: _.get(apiKeys, 'ansible.ansibleHost', 'GCP project id not provided'),
    ansibleToken: _.get(apiKeys, 'ansible.ansibleToken', 'GCP service account json key not provided'),
  }
   return filterTestOptions({ azure, aws, gcp, vmware, ansible }, Object.keys(apiKeys))
 }
 
 const getCreateCluster = ({ cloudConnection }) => {
   const { apiKeys } = cloudConnection
 
   const azure = {
     name: 'console-ui-test-cluster-azure',
     releaseImage: _.get(apiKeys, 'vmware.network', 'VMware vSphere network not provided'),
     region: _.get(apiKeys, 'azure.region', ''),
     timeout: 90 // minutes
   }
   const aws = {
     name: 'console-ui-test-cluster-aws',
     releaseImage: '',
     region: _.get(apiKeys, 'aws.region', ''),
     timeout: 80 // minutes
   }
   const gcp = {
     name: 'console-ui-test-cluster-google',
     releaseImage: '',
     region: _.get(apiKeys, 'gcp.region', ''),
     timeout: 80 // minutes
   }
   const vmware = {
     name: 'console-ui-test-cluster-vmware',
     releaseImage: '',
     network: _.get(apiKeys, 'vmware.network', 'VMware vSphere network not provided'),
     apiVIP: _.get(apiKeys, 'vmware.apiVIP', 'VMware vSphere API virtual IP not provided'),
     ingressVIP: _.get(apiKeys, 'vmware.ingressVIP', 'VMware vSphere Ingress virtual IP not provided'),
     timeout: 80 // minutes
   }
 
   return filterTestOptions({ azure, aws, gcp, vmware }, Object.keys(apiKeys))
 }
 
 const getCredentials = ({ cloudConnection }) => {
   return {
     pullSecret: _.get(cloudConnection, 'pullSecret', 'OpenShift pull secret not provided'),
     sshPrivatekey: _.get(cloudConnection, 'sshPrivatekey', 'SSH private key not provided'),
     sshPublickey: _.get(cloudConnection, 'sshPublickey', 'SSH public key not provided')
   }
 }
 
 const getConfiguredOptions = (options, configuredProviders) =>{
   const configurations = {}
   for (const provider in options) {
     if (configuredProviders.includes(provider)) {
       configurations[provider] = options[provider]
     }
   }
   return configurations
 }
 
 const filterTestOptions = (options, configuredProviders) => {
   const { aws, gcp, azure, vmware } = options
   const { TEST_GROUP } = process.env
   switch(TEST_GROUP) {
   case 'aws':
     return { aws }
   case 'google':
     return { gcp }
   case 'azure':
     return { azure }
   case 'vmware':
     return { vmware }
   case 'provision-all':
   case 'all':
   default:
     return getConfiguredOptions(options, configuredProviders)
   }
 }
 
 transformOptions()
 