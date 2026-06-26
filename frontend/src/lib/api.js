import { getApiBaseUrl } from './config'

const BASE = getApiBaseUrl()

export class SessionExpiredError extends Error {
  constructor() {
    super('Session expired')
    this.name = 'SessionExpiredError'
  }
}

function authHeaders() {
  const token = sessionStorage.getItem('token')
  return token ? { Authorization: `Bearer ${token}` } : {}
}

function getJwtExp(token) {
  try {
    const [, payload] = token.split('.')
    const normalized = payload.replace(/-/g, '+').replace(/_/g, '/')
    return JSON.parse(atob(normalized)).exp
  } catch {
    return null
  }
}

function tokenExpiresSoon(token, skewSeconds = 60) {
  const exp = getJwtExp(token)
  return !exp || exp * 1000 - Date.now() < skewSeconds * 1000
}

let isRefreshing = false
let refreshPromise = null

async function refreshAccessToken() {
  if (isRefreshing) return refreshPromise

  isRefreshing = true
  refreshPromise = (async () => {
    try {
      const res = await fetch(`${BASE}/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          ...authHeaders(),
        },
      })
      if (!res.ok) return null

      const data = await res.json()
      if (data.token) {
        sessionStorage.setItem('token', data.token)
        return data.token
      }
      return null
    } catch (err) {
      if (import.meta.env.DEV) console.error('Token refresh failed', err)
      return null
    } finally {
      isRefreshing = false
      refreshPromise = null
    }
  })()

  return refreshPromise
}

async function ensureFreshToken() {
  const token = sessionStorage.getItem('token')
  if (!token || !tokenExpiresSoon(token)) return token
  return refreshAccessToken()
}

async function parseResponse(res) {
  if (res.status === 204) return null

  let data = null
  try {
    data = await res.json()
  } catch (err) {
    if (import.meta.env.DEV) console.error('Failed to parse API response', err)
  }

  if (!res.ok) {
    throw new Error(data?.message || `Request failed (${res.status})`)
  }

  return data
}

async function request(path, options = {}) {
  const refreshedToken = await ensureFreshToken()
  const res = await fetch(`${BASE}${path}`, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...authHeaders(),
      ...(refreshedToken ? { Authorization: `Bearer ${refreshedToken}` } : {}),
      ...options.headers,
    },
  })

  if (res.status === 401 && !options._retried) {
    const newToken = await refreshAccessToken()
    if (newToken) {
      return request(path, {
        ...options,
        _retried: true,
        headers: {
          ...options.headers,
          Authorization: `Bearer ${newToken}`,
        },
      })
    }

    sessionStorage.removeItem('token')
    window.dispatchEvent(new Event('sendr:session-expired'))
    throw new SessionExpiredError()
  }

  return parseResponse(res)
}

export const api = {
  me: () => request('/me'),
  updateProfile: (payload) => request('/me', { method: 'PATCH', body: JSON.stringify(payload) }),
  createKey: (name) => request('/apikeys', { method: 'POST', body: JSON.stringify({ name }) }),
  listKeys: () => request('/apikeys'),
  revokeKey: (id) => request(`/apikeys/${id}`, { method: 'DELETE' }),
  sendEmail: (payload, apiKey) =>
    request('/emails/send', {
      method: 'POST',
      headers: { Authorization: `Bearer ${apiKey}` },
      body: JSON.stringify(payload),
    }),
  getJob: (id, apiKey) =>
    request(`/emails/${id}`, {
      headers: { Authorization: `Bearer ${apiKey}` },
    }),
  listEmails: (status = '', limit = 20, offset = 0) => {
    const params = new URLSearchParams({ limit, offset })
    if (status) params.set('status', status)
    return request(`/emails?${params}`)
  },
  logout: () => request('/auth/logout', { method: 'POST' }),
  // Payment / Plans
  listPlans: () => request('/plans'),
  createPaymentOrder: (planName) =>
    request('/payments/orders', {
      method: 'POST',
      body: JSON.stringify({ plan_name: planName }),
    }),
  verifyPayment: (data) =>
    request('/payments/verify', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
}
