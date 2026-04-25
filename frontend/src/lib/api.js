const BASE = import.meta.env.VITE_API_URL

function authHeaders() {
  const token = localStorage.getItem('token')
  return token ? { Authorization: `Bearer ${token}` } : {}
}

async function request(path, options = {}) {
  const res = await fetch(`${BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...authHeaders(),
      ...options.headers,
    },
  })
  if (res.status === 204) return null
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
}