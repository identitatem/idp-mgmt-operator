/** *****************************************************************************
 * Licensed Materials - Property of Red Hat, Inc.
 * Copyright (c) 2020 Red Hat, Inc.
 ****************************************************************************** */

export const getOpt = (opts, key, defaultValue) => {
  if (opts && opts[key]) {
    return opts[key]
  }

  return defaultValue
}
