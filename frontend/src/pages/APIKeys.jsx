import { useCallback, useEffect, useRef, useState } from 'react'
import { api } from '../lib/api'
import Layout from '../components/Layout'

async function copyText(text) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text)
    return
  }

  const el = document.createElement('textarea')
  el.value = text
  el.setAttribute('readonly', '')
  el.style.position = 'fixed'
  el.style.opacity = '0'
  document.body.appendChild(el)
  el.select()
  document.execCommand('copy')
  document.body.removeChild(el)
}

export default function APIKeys() {
  const [keys, setKeys] = useState([])
  const [newKeyName, setNewKeyName] = useState('')
  const [createdKey, setCreatedKey] = useState(null)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState(null)
  const [copied, setCopied] = useState(false)
  const [pendingRevoke, setPendingRevoke] = useState(null)

  const errorRef = useRef(null)
  const cancelRevokeRef = useRef(null)
  const lastRevokeButtonRef = useRef(null)

  const loadKeys = useCallback(async () => {
    try {
      const data = await api.listKeys()
      setKeys(data ?? [])
    } catch (err) {
      setError(err.message)
    }
  }, [])

  useEffect(() => {
    Promise.resolve().then(loadKeys)
  }, [loadKeys])

  useEffect(() => {
    if (error) errorRef.current?.focus()
  }, [error])

  useEffect(() => {
    if (pendingRevoke) cancelRevokeRef.current?.focus()
  }, [pendingRevoke])

  async function handleCreate(e) {
    e.preventDefault()
    if (!newKeyName.trim()) return

    setCreating(true)
    setError(null)
    try {
      const data = await api.createKey(newKeyName.trim())
      setCreatedKey(data)
      setCopied(false)
      setNewKeyName('')
      loadKeys()
    } catch (err) {
      setError(err.message)
    } finally {
      setCreating(false)
    }
  }

  async function handleRevoke(id) {
    try {
      await api.revokeKey(id)
      setKeys((current) => current.filter((key) => key.id !== id))
      setPendingRevoke(null)
      lastRevokeButtonRef.current?.focus()
    } catch (err) {
      setError(err.message)
    }
  }

  function openRevokeModal(key, button) {
    lastRevokeButtonRef.current = button
    setPendingRevoke(key)
  }

  function closeRevokeModal() {
    setPendingRevoke(null)
    lastRevokeButtonRef.current?.focus()
  }

  async function copyKey() {
    try {
      await copyText(createdKey.api_key)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      setError('Could not copy API key.')
    }
  }

  const errorId = error ? 'api-keys-error' : undefined

  return (
    <Layout>
      <div className="p-8 max-w-2xl">
        <h1 className="text-lg font-semibold mb-1" style={{ color: 'var(--text)' }}>API Keys</h1>
        <p className="text-sm mb-8" style={{ color: 'var(--muted)' }}>Keys authenticate your email send requests.</p>

        <form onSubmit={handleCreate} className="flex gap-2 mb-6" aria-describedby={errorId}>
          <label htmlFor="new-key-name" className="sr-only">API key name</label>
          <input
            id="new-key-name"
            value={newKeyName}
            onChange={(e) => setNewKeyName(e.target.value)}
            placeholder="Key name e.g. production"
            aria-invalid={Boolean(error)}
            aria-describedby={errorId}
            className="flex-1 px-3 py-2 rounded-lg text-sm outline-none transition-colors"
            style={{ background: 'var(--surface)', border: '1px solid var(--border)', color: 'var(--text)' }}
            onFocus={(e) => { e.target.style.borderColor = 'var(--accent)' }}
            onBlur={(e) => { e.target.style.borderColor = 'var(--border)' }}
          />
          <button type="submit" disabled={creating || !newKeyName.trim()} className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity disabled:opacity-40" style={{ background: 'var(--accent)', color: '#000' }}>
            {creating ? 'Creating...' : 'Create key'}
          </button>
        </form>

        {error && (
          <div id="api-keys-error" ref={errorRef} role="alert" tabIndex={-1} className="text-xs mb-4 px-3 py-2 rounded-lg outline-none" style={{ background: 'var(--danger-dim)', color: 'var(--danger)', border: '1px solid rgba(255,77,77,0.2)' }}>
            {error}
          </div>
        )}

        {createdKey && (
          <div className="rounded-xl border p-4 mb-6" style={{ background: 'var(--accent-dim)', borderColor: 'var(--accent-border)' }} role="status">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium" style={{ color: 'var(--accent)' }}>Key created - copy it now, it will not be shown again</span>
              <button type="button" onClick={() => setCreatedKey(null)} className="text-xs" style={{ color: 'var(--muted)' }}>dismiss</button>
            </div>
            <div className="flex items-center gap-2">
              <code className="flex-1 text-xs p-2 rounded mono overflow-x-auto" style={{ background: 'var(--surface)', color: 'var(--text)', border: '1px solid var(--border)' }}>
                {createdKey.api_key}
              </code>
              <button type="button" onClick={copyKey} className="px-3 py-2 rounded-lg text-xs font-medium transition-all" style={{ background: copied ? 'var(--accent)' : 'var(--surface)', color: copied ? '#000' : 'var(--muted)', border: '1px solid var(--border)' }}>
                {copied ? 'Copied!' : 'Copy'}
              </button>
            </div>
          </div>
        )}

        <div className="rounded-xl border overflow-hidden" style={{ borderColor: 'var(--border)' }}>
          {keys.length === 0 ? (
            <div className="p-8 text-center text-sm" style={{ color: 'var(--muted)' }}>No API keys yet</div>
          ) : (
            keys.map((key, index) => (
              <div key={key.id} className="flex items-center justify-between px-4 py-3" style={{ background: 'var(--surface)', borderBottom: index < keys.length - 1 ? '1px solid var(--border)' : 'none' }}>
                <div>
                  <div className="text-sm font-medium mb-0.5" style={{ color: 'var(--text)' }}>{key.name}</div>
                  <div className="text-xs mono" style={{ color: 'var(--muted)' }}>mk_live_{key.prefix}...</div>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-xs" style={{ color: 'var(--muted)' }}>{new Date(key.created_at).toLocaleDateString()}</span>
                  <button
                    type="button"
                    onClick={(e) => openRevokeModal(key, e.currentTarget)}
                    className="text-xs px-2.5 py-1 rounded-lg transition-colors hover:text-red-400"
                    style={{ color: 'var(--muted)', border: '1px solid var(--border)' }}
                  >
                    Revoke
                  </button>
                </div>
              </div>
            ))
          )}
        </div>

        {pendingRevoke && (
          <div className="fixed inset-0 flex items-center justify-center px-4" style={{ background: 'rgba(0,0,0,0.48)' }}>
            <div role="dialog" aria-modal="true" aria-labelledby="revoke-title" className="w-full max-w-sm rounded-xl border p-5" style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
              <h2 id="revoke-title" className="text-sm font-semibold mb-2" style={{ color: 'var(--text)' }}>Revoke API key?</h2>
              <p className="text-sm mb-5" style={{ color: 'var(--muted)' }}>
                This will permanently revoke "{pendingRevoke.name}".
              </p>
              <div className="flex justify-end gap-2">
                <button ref={cancelRevokeRef} type="button" onClick={closeRevokeModal} className="px-3 py-2 rounded-lg text-sm" style={{ color: 'var(--muted)', border: '1px solid var(--border)' }}>Cancel</button>
                <button type="button" onClick={() => handleRevoke(pendingRevoke.id)} className="px-3 py-2 rounded-lg text-sm font-medium" style={{ background: 'var(--danger)', color: '#fff' }}>Revoke</button>
              </div>
            </div>
          </div>
        )}
      </div>
    </Layout>
  )
}
