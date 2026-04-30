const BASE = import.meta.env.VITE_API_URL

function authHeaders() {
  const token = sessionStorage.getItem('token')
  return token ? { Authorization: `Bearer ${token}` } : {}
}

// Flag to prevent multiple concurrent refresh attempts
let isRefreshing = false
let refreshPromise = null

/**
 * Attempts to refresh the access token using the HttpOnly refresh_token cookie.
 * Returns the new access token on success, or null on failure.
 */
async function refreshAccessToken() {
  // If a refresh is already in progress, wait for it
  if (isRefreshing) return refreshPromise

  isRefreshing = true
  refreshPromise = (async () => {
    try {
      const res = await fetch(`${BASE}/auth/refresh`, {
        method: 'POST',
        credentials: 'include', // sends refresh_token cookie
        headers: {
          'Content-Type': 'application/json',
          ...authHeaders(), // sends the expired JWT so backend can read user_id
        },
      })
      if (!res.ok) return null
      const data = await res.json()
      if (data.token) {
        sessionStorage.setItem('token', data.token)
        return data.token
      }
      return null
    } catch {
      return null
    } finally {
      isRefreshing = false
      refreshPromise = null
    }
  })()

  return refreshPromise
}

async function request(path, options = {}) {
  const res = await fetch(`${BASE}${path}`, {
    ...options,
    credentials: 'include', // always send cookies
    headers: {
      'Content-Type': 'application/json',
      ...authHeaders(),
      ...options.headers,
    },
  })
  if (res.status === 204) return null

  // If 401, try to refresh the token and retry once
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
    // Refresh failed — clear session and redirect to login
    sessionStorage.removeItem('token')
    window.location.href = '/'
    throw new Error('Session expired')
  }

  const data = await res.json().catch(() => ({ message: 'Request failed' }))
  if (!res.ok) throw new Error(data.message || 'Request failed')
  return data
}

export const api = {
  me: () => request('/me'),
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
  logout: () =>
    request('/auth/logout', { method: 'POST' }),
}