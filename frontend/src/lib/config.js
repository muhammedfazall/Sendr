const allowedHosts = new Set([
  'localhost:8080',
  '127.0.0.1:8080',
  'sendr.app',
  'sendr-api.onrender.com',
])

const PROD_API_URL = 'https://sendr-api.onrender.com'
const DEV_API_URL = 'http://localhost:8080'

export function getApiBaseUrl() {
  const raw = import.meta.env.VITE_API_URL || (import.meta.env.PROD ? PROD_API_URL : DEV_API_URL)

  const url = new URL(raw)

  if (!['http:', 'https:'].includes(url.protocol)) {
    throw new Error('VITE_API_URL must use http or https')
  }

  if (!allowedHosts.has(url.host)) {
    throw new Error(`Untrusted API host: ${url.host}`)
  }

  return url.origin
}
