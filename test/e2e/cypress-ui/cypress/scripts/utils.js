/* Copyright Red Hat */

export const getOpt = (opts, key, defaultValue) => {
  if (opts && opts[key]) {
    return opts[key]
  }

  return defaultValue
}
