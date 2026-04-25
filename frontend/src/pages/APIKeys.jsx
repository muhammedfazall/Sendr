import { useState, useEffect } from 'react'
import { api } from '../lib/api'
import Layout from '../components/Layout'

export default function APIKeys() {
  const [keys, setKeys] = useState([])
  const [newKeyName, setNewKeyName] = useState('')
  const [createdKey, setCreatedKey] = useState(null) // shown once
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState(null)
  const [copied, setCopied] = useState(false)

  useEffect(() => { loadKeys() }, [])

  async function loadKeys() {
    try {
      const data = await api.listKeys()
      setKeys(data ?? [])
    } catch (e) {
      setError(e.message)
    }
  }

  async function handleCreate(e) {
    e.preventDefault()
    if (!newKeyName.trim()) return
    setCreating(true)
    setError(null)
    try {
      const data = await api.createKey(newKeyName.trim())
      setCreatedKey(data)
      setNewKeyName('')
      loadKeys()
    } catch (e) {
      setError(e.message)
    } finally {
      setCreating(false)
    }
  }

  async function handleRevoke(id) {
    if (!confirm('Revoke this key? This cannot be undone.')) return
    try {
      await api.revokeKey(id)
      setKeys(keys.filter(k => k.id !== id))
    } catch (e) {
      setError(e.message)
    }
  }

  function copyKey() {
    navigator.clipboard.writeText(createdKey.api_key)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Layout>
      <div className="p-8 max-w-2xl">
        <h1 className="text-lg font-semibold mb-1" style={{ color: 'var(--text)' }}>API Keys</h1>
        <p className="text-sm mb-8" style={{ color: 'var(--muted)' }}>
          Keys authenticate your email send requests.
        </p>

        {/* Create form */}
        <form onSubmit={handleCreate} className="flex gap-2 mb-6">
          <input
            value={newKeyName}
            onChange={e => setNewKeyName(e.target.value)}
            placeholder="Key name e.g. production"
            className="flex-1 px-3 py-2 rounded-lg text-sm outline-none transition-colors"
            style={{
              background: 'var(--surface)',
              border: '1px solid var(--border)',
              color: 'var(--text)',
            }}
            onFocus={e => e.target.style.borderColor = 'var(--accent)'}
            onBlur={e => e.target.style.borderColor = 'var(--border)'}
          />
          <button type="submit" disabled={creating || !newKeyName.trim()}
            className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity disabled:opacity-40"
            style={{ background: 'var(--accent)', color: '#000' }}>
            {creating ? 'Creating…' : 'Create key'}
          </button>
        </form>

        {error && (
          <div className="text-xs mb-4 px-3 py-2 rounded-lg"
            style={{ background: 'var(--danger-dim)', color: 'var(--danger)', border: '1px solid rgba(255,77,77,0.2)' }}>
            {error}
          </div>
        )}

        {/* New key reveal — shown once */}
        {createdKey && (
          <div className="rounded-xl border p-4 mb-6"
            style={{ background: 'var(--accent-dim)', borderColor: 'var(--accent-border)' }}>
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium" style={{ color: 'var(--accent)' }}>
                ✓ Key created — copy it now, it won't be shown again
              </span>
              <button onClick={() => setCreatedKey(null)}
                className="text-xs" style={{ color: 'var(--muted)' }}>dismiss</button>
            </div>
            <div className="flex items-center gap-2">
              <code className="flex-1 text-xs p-2 rounded mono overflow-x-auto"
                style={{ background: 'var(--surface)', color: 'var(--text)', border: '1px solid var(--border)' }}>
                {createdKey.api_key}
              </code>
              <button onClick={copyKey}
                className="px-3 py-2 rounded-lg text-xs font-medium transition-all"
                style={{
                  background: copied ? 'var(--accent)' : 'var(--surface)',
                  color: copied ? '#000' : 'var(--muted)',
                  border: '1px solid var(--border)',
                }}>
                {copied ? 'Copied!' : 'Copy'}
              </button>
            </div>
          </div>
        )}

        {/* Keys list */}
        <div className="rounded-xl border overflow-hidden"
          style={{ borderColor: 'var(--border)' }}>
          {keys.length === 0 ? (
            <div className="p-8 text-center text-sm" style={{ color: 'var(--muted)' }}>
              No API keys yet
            </div>
          ) : (
            keys.map((key, i) => (
              <div key={key.id}
                className="flex items-center justify-between px-4 py-3"
                style={{
                  background: 'var(--surface)',
                  borderBottom: i < keys.length - 1 ? '1px solid var(--border)' : 'none',
                }}>
                <div>
                  <div className="text-sm font-medium mb-0.5" style={{ color: 'var(--text)' }}>{key.name}</div>
                  <div className="text-xs mono" style={{ color: 'var(--muted)' }}>
                    mk_live_{key.prefix}…
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-xs" style={{ color: 'var(--muted)' }}>
                    {new Date(key.created_at).toLocaleDateString()}
                  </span>
                  <button onClick={() => handleRevoke(key.id)}
                    className="text-xs px-2.5 py-1 rounded-lg transition-colors hover:text-red-400"
                    style={{ color: 'var(--muted)', border: '1px solid var(--border)' }}>
                    Revoke
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </Layout>
  )
}