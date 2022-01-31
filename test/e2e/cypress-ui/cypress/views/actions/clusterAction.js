/** *****************************************************************************
 * Licensed Materials - Property of Red Hat, Inc.
 * Copyright (c) 2021 Red Hat, Inc.
 ****************************************************************************** */

/// <reference types="cypress" />

import * as cluster from '../../apis/cluster'
import * as constants from '../../support/constants'
import * as metrics from '../../apis/metrics'

export const clusterActions = {
    /**
    * Using oc command, export a cluster's kubeconfig as an asset. Overwrites any existing file with the same name.
    * Expects environment to have oc cli cmd installed and available.
    * @param {E} clusterName 
    */
    extractClusterKubeconfig: (clusterName) => {
        cy.ocLogin(Cypress.env('token'), constants.apiUrl).then(() => {
            cy.ocExtractClusterKubeconfig(clusterName)
        })
    },
    shouldHaveManagedClusterForUser: (clusterName) => {
        cluster.getManagedCluster(clusterName).then((resp) => {
            if (!resp.isOkStatusCode) {
                let request_body = `
{
"kind": "ManagedCluster",
"apiVersion": "cluster.open-cluster-management.io/v1",
"metadata": {
    "name": "${clusterName}",
    "labels": {
        "cloud": "AWS",
        "name": "${clusterName}"
    }
},
"spec": {
    "hubAcceptsClient": true,
    "leaseDurationSeconds": 60
}
}`
                cluster.createManagedCluster(request_body)
            }
        })
    },
    deleteManagedClusterForUser: (clusterName) => {
        cluster.deleteManagedCluster(clusterName).then((resp) => expect(resp.isOkStatusCode))
    },
    checkClusterLabels: (clusterName, labels) => {
        cy.log("Check the managed cluster labels");
        cy.waitUntil(() => {
            return cluster.getManagedCluster(clusterName).then(resp => {
                var exist = false
                    if (resp.body['metadata']['labels'].hasOwnProperty(labels.split('=')[0])) {
                        if (resp.body['metadata']['labels'][`${labels.split('=')[0]}`] === labels.split('=')[1]) {
                            exist = true
                        }
                    }
                return exist
            })
        },
        {
            errorMsg: "Can not find the lables " + labels + " in managed cluster " + clusterName,
            interval: 2 * 1000,
            timeout: 10 * 1000
        })
    },
    checkManagedClusterAddonStatus: (clusterName) => {
        cy.log("check the managed cluster addon status");
        cluster.getManagedClusterAddons(clusterName).then(resp => {
            if (resp.isOkStatusCode) {
                for (let i=0; i<resp.body['items'].length; i++) {
                    cy.get(`tr[data-ouia-component-id="${resp.body['items'][i].metadata.name}"]`).should('exist')
                    for (let j=0; j<resp.body['items'][i].status['conditions'].length; j++) {
                        if (resp.body['items'][i].status['conditions'][j].type === 'Available' ) {
                            cy.get(`tr[data-ouia-component-id="${resp.body['items'][i].metadata.name}"] td[data-label="Message"]`).contains(`${resp.body['items'][i].status['conditions'][j].message}`)
                            if (resp.body['items'][i].status['conditions'][j].status === 'True') {
                                cy.get(`tr[data-ouia-component-id="${resp.body['items'][i].metadata.name}"] td[data-label="Status"]`).contains('Available')
                            }
                        }
                    }
                }
            }
        })
    },
    checkClusterPoolStatus: (clusterPoolName, namespace) => {
        cy.log('checking cluster pool status')
        cy.waitUntil(() => {
            return cluster.getClusterPool(clusterPoolName, namespace).then(resp => {
                if (resp.isOkStatusCode) {
                    if (resp.body.status.size === resp.body.status.ready) {
                        return true
                    }
                }
                return false
            })
        },
        {
            errorMsg: "Cluster Pool " + clusterPoolName + " did not finish in given time",
            interval: 300000, // check every 5 minutes
            timeout: 3600000 // wait an hour
        })
    },
    checkClusterStatus: (clusterName) => {
        cy.log('checking cluster status')
        cy.setAPIToken()
        cy.waitUntil(() => {
            return cluster.getManagedCluster(clusterName).then(resp => {
                for (let i=0; i<resp.body['status']['conditions'].length; i++) {
                    var condition = resp.body['status']['conditions'][i]
                    if (condition.type === "ManagedClusterJoined") {
                        if (condition.status == 'True') {
                            return true
                        }
                    }
                }
                return false
            })
        },
        {
            errorMsg: "Cluster  " + clusterName + " did not finish in given time",
            interval: 600000, // check every 10 minutes
            timeout: 3600000 // wait an hour
        })
    },
    checkHiveCluster: (clusterName) => {
        cy.log('check hive clusters')
        return cluster.getManagedCluster(clusterName).then(resp => {
            if (!resp.isOkStatusCode) {
                return false
            }else{
                return (resp.body.metadata.annotations["open-cluster-management/created-via"] == 'hive')
            }
        })
    }
}

export const clusterDeploymentActions = {
    checkClusterDeploymentPowerStatus: (clusterName, finalStatus) => {
        cy.log("Check the cluster deployment power status");
        cy.waitUntil(() => {
            return cluster.getClusterDeployment(clusterName).then(resp => {
                var exist = false
                for (let i=0; i<resp['status']['conditions'].length; i++) {
                    var condition = resp['status']['conditions'][i]
                    if (condition.type === "Hibernating") {
                        if (condition.reason == finalStatus) {
                            exist = true
                        }
                    }
                }
                return exist
            })
        },
        {
            interval: 5 * 1000,
            timeout: 1800 * 1000
        })
    },
}

export const clusterMetricsActions = {
    waitForLables: (clusterName) => {
        cy.log("Check the cluster metrics to make sure the labels added")
        cy.waitUntil( () => {
            return cluster.getManagedCluster(clusterName).then( resp => {
                if (resp.isOkStatusCode) {
                    if (resp.body.metadata.labels.vendor != "" && resp.body.metadata.labels.vendor != "auto-detect" && resp.body.metadata.labels.cloud != "auto-detect") {
                        if (resp.body.metadata.labels.vendor === 'OpenShift') {
                            return (resp.body.metadata.labels.hasOwnProperty('clusterID') && resp.body.metadata.labels.clusterID != "")
                        } else {
                            return true
                        }
                    } else {
                        return false
                    }
                } else {
                    return false
                }
            })
        },
        {
            interval: 10 * 1000,
            timeout: 600 * 1000
        })
    },
    waitForMetrics: (clusterName) => {
        cy.log("Check the cluster mertics was generated")
        cluster.getManagedCluster(clusterName).then( resp => {
            cy.waitUntil( () => {
                if (!resp.body.metadata.labels.hasOwnProperty('clusterID') && resp.body.metadata.labels.vendor != 'OpenShift') {
                    return metrics.getClusterMetrics(clusterName).then( metrics => {
                        return (metrics.data.result.length > 0)
                    })
                }else{
                    return metrics.getClusterMetrics(resp.body.metadata.labels.clusterID).then( metrics => {
                        return (metrics.data.result.length > 0)
                    })
                }
            },
            {
                interval: 10 * 1000,
                timeout: 600 * 1000
            })
        })
    },
    // The checkClusterMetrics used to handle the test cases of RHACM4K-1735
    checkClusterMetrics: (clusterName, type) => {
        switch (type) {
            case 'Destroy': {
                cy.log("Check the cluster metrics when the cluster is destroy")
                // (TODO) need to find a way to check the metrics if the cluster was destroy
                break;
            }
            case 'Detach':{
                cy.log("Check the cluster metrics when the cluster is detached")
                cluster.getClusterDeployment(clusterName).then( resp => {
                    cy.waitUntil( () => {
                        // if the resp was 404 that means the cluster was not created by hive, so we need to use clusterName do search
                        if (resp === 404) {
                            return metrics.getClusterMetrics(clusterName).then( metrics => {
                                return (metrics.data.result.length == 0)
                            })
                        }else{
                            return metrics.getClusterMetrics(resp.spec.clusterMetadata.clusterID).then( metrics => {
                                return (metrics.data.result.length == 0)
                            })
                        }
                    },
                    {
                        interval: 10 * 1000,
                        timeout: 18000 * 1000 
                    })
                })
                break;
            }
            case 'Hibernating':{
                cy.log("Check the cluster metrics when the cluster is Hibernating")
                cluster.getClusterDeployment(clusterName).then( resp => {
                    cy.waitUntil( () => {
                        return metrics.getClusterMetrics(resp.spec.clusterMetadata.clusterID).then( metrics => {
                            return (metrics.data.result.length == 0)
                        })
                    },
                    {
                        interval: 10 * 1000,
                        timeout: 18000 * 1000 
                    })
                })
                break;
            }
            default:{
                clusterMetricsActions.waitForLables(clusterName)
                clusterMetricsActions.waitForMetrics(clusterName)
                cy.log("Check the cluster metrics")
                cluster.getManagedCluster(clusterName).then( resp => {
                    if (resp.isOkStatusCode) {
                        if (!resp.body.metadata.labels.hasOwnProperty('clusterID') && resp.body.metadata.labels.vendor != 'OpenShift') {
                            // The cluster was not the OCP cluster
                            metrics.getClusterMetrics(clusterName).then( metrics => {
                                expect(metrics.status).to.be.eq('success')
                                expect(metrics.data.result[0].metric.cloud).to.be.eq(resp.body.metadata.labels.cloud)
                                expect(metrics.data.result[0].metric.created_via.toLowerCase()).to.be.eq(resp.body.metadata.annotations["open-cluster-management/created-via"])
                                expect(metrics.data.result[0].metric.managed_cluster_id).to.be.eq(clusterName)
                                expect(metrics.data.result[0].metric.vendor).to.be.eq(resp.body.metadata.labels.vendor)
                                expect(metrics.data.result[0].metric.version).to.be.eq(resp.body.status.version.kubernetes)
                            })
                        } else {
                            // The cluster was OCP cluster
                            metrics.getClusterMetrics(resp.body.metadata.labels.clusterID).then( metrics => {
                                expect(metrics.status).to.be.eq('success')
                                expect(metrics.data.result[0].metric.cloud).to.be.eq(resp.body.metadata.labels.cloud)
                                expect(metrics.data.result[0].metric.created_via.toLowerCase()).to.be.eq(resp.body.metadata.annotations["open-cluster-management/created-via"])
                                expect(metrics.data.result[0].metric.managed_cluster_id).to.be.eq(resp.body.metadata.labels.clusterID)
                                expect(metrics.data.result[0].metric.vendor).to.be.eq(resp.body.metadata.labels.vendor)
                                expect(metrics.data.result[0].metric.version).to.be.eq(resp.body.metadata.labels.openshiftVersion)
                            })
                        }
                    }
                })
                break;
            }
        }
    }
}