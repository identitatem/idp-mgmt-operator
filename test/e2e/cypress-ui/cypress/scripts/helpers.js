
/** *****************************************************************************
 * Licensed Materials - Property of Red Hat, Inc.
 * Copyright (c) 2021 Red Hat, Inc.
 ****************************************************************************** */

const fs = require('fs')
const path = require('path')
const del = require('del')
const junitMerger = require('junit-report-merger')

exports.cleanReports = () => {
  const reportPath = path.join(__dirname, '..', '..', 'results')
  del.sync([reportPath])
}
