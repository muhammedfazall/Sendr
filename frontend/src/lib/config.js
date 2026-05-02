const allowedHosts = new Set([
  'localhost:8080',
  '127.0.0.1:8080',
  'sendr.app',
])

export function getApiBaseUrl() {
  const raw = import.meta.env.VITE_API_URL

  if (!raw) {
    throw new Error('VITE_API_URL is required')
  }

  const url = new URL(raw)

  if (!['http:', 'https:'].includes(url.protocol)) {
    throw new Error('VITE_API_URL must use http or https')
  }

  if (!allowedHosts.has(url.host)) {
    throw new Error(`Untrusted API host: ${url.host}`)
  }

  return url.origin
}
