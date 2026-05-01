import { useCallback, useEffect, useRef, useState } from 'react'
import { api } from '../lib/api'
import Layout from '../components/Layout'

const HISTORY_LIMIT = 20

const STATUS_COLOR = {
  pending: { color: '#f59e0b', bg: 'rgba(245,158,11,0.08)', border: 'rgba(245,158,11,0.2)' },
  processing: { color: '#60a5fa', bg: 'rgba(96,165,250,0.08)', border: 'rgba(96,165,250,0.2)' },
  sent: { color: '#00d084', bg: 'rgba(0,208,132,0.08)', border: 'rgba(0,208,132,0.2)' },
  failed: { color: '#ff4d4d', bg: 'rgba(255,77,77,0.08)', border: 'rgba(255,77,77,0.2)' },
}

const STATUS_LABEL = {
  pending: 'Pending',
  processing: 'Processing',
  sent: 'Sent',
  failed: 'Failed',
}

function validateEmailForm(form) {
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.to.trim())) return 'Enter a valid recipient email.'
  if (!form.subject.trim()) return 'Subject is required.'
  if (form.subject.length > 998) return 'Subject must be under 998 characters.'
  if (!form.body.trim()) return 'Body is required.'
  if (form.body.length > 50000) return 'Body must be under 50 KB.'
  return null
}

export default function SendEmail() {
  const [keys, setKeys] = useState([])
  const [selectedKey, setSelectedKey] = useState('')
  const [form, setForm] = useState({ to: '', subject: '', body: '' })
  const [sending, setSending] = useState(false)
  const [job, setJob] = useState(null)
  const [error, setError] = useState(null)
  const [history, setHistory] = useState([])
  const [historyLoading, setHistoryLoading] = useState(false)
  const [statusFilter, setStatusFilter] = useState('')
  const [offset, setOffset] = useState(0)
  const [hasMoreHistory, setHasMoreHistory] = useState(false)

  const pollRef = useRef(null)
  const apiKeyRef = useRef('')
  const apiKeyInputRef = useRef(null)
  const errorRef = useRef(null)

  const clearApiKeyInput = useCallback(() => {
    apiKeyRef.current = ''
    if (apiKeyInputRef.current) apiKeyInputRef.current.value = ''
  }, [])

  const fetchHistory = useCallback((filter, nextOffset = 0, append = false) => {
    setHistoryLoading(true)
    api
      .listEmails(filter, HISTORY_LIMIT, nextOffset)
      .then((data) => {
        const rows = data ?? []
        setHistory((current) => (append ? [...current, ...rows] : rows))
        setHasMoreHistory(rows.length === HISTORY_LIMIT)
        setOffset(nextOffset)
      })
      .catch(() => {
        if (!append) setHistory([])
        setHasMoreHistory(false)
      })
      .finally(() => setHistoryLoading(false))
  }, [])

  useEffect(() => {
    api
      .listKeys()
      .then((data) => {
        const nextKeys = data ?? []
        setKeys(nextKeys)
        if (nextKeys.length > 0) setSelectedKey(nextKeys[0].prefix)
      })
      .catch(() => {})
  }, [])

  useEffect(() => {
    Promise.resolve().then(() => fetchHistory('', 0, false))
  }, [fetchHistory])

  useEffect(() => {
    if (!job || job.status === 'sent' || job.status === 'failed') {
      clearInterval(pollRef.current)
      pollRef.current = null
      return
    }

    clearInterval(pollRef.current)
    pollRef.current = setInterval(async () => {
      try {
        const updated = await api.getJob(job.id, job._apiKey)
        setJob((current) => ({
          ...updated,
          _apiKey: updated.status === 'sent' || updated.status === 'failed' ? '' : current._apiKey,
        }))
        if (updated.status === 'sent' || updated.status === 'failed') {
          fetchHistory(statusFilter, 0, false)
        }
      } catch {
        clearInterval(pollRef.current)
        pollRef.current = null
      }
    }, 2000)

    return () => {
      clearInterval(pollRef.current)
      pollRef.current = null
    }
  }, [fetchHistory, job, statusFilter])

  useEffect(() => {
    if (error) errorRef.current?.focus()
  }, [error])

  const handleSend = useCallback(async (e) => {
    e.preventDefault()

    const validationError = validateEmailForm(form)
    if (validationError) {
      setError(validationError)
      return
    }
    if (!selectedKey) {
      setError('Select an API key first.')
      return
    }

    const fullKey = apiKeyRef.current.trim()
    if (!fullKey) {
      setError('Paste your full API key first.')
      return
    }

    setSending(true)
    setError(null)
    setJob(null)

    try {
      const data = await api.sendEmail({
        to: form.to.trim(),
        subject: form.subject.trim(),
        body: form.body,
      }, fullKey)
      setJob({ ...data, id: data.job_id, status: 'pending', _apiKey: fullKey })
      clearApiKeyInput()
      fetchHistory(statusFilter, 0, false)
    } catch (err) {
      setError(err.message)
    } finally {
      setSending(false)
    }
  }, [clearApiKeyInput, fetchHistory, form, selectedKey, statusFilter])

  const handleFilterChange = (filter) => {
    setStatusFilter(filter)
    fetchHistory(filter, 0, false)
  }

  const loadMoreHistory = () => {
    fetchHistory(statusFilter, offset + HISTORY_LIMIT, true)
  }

  const statusStyle = STATUS_COLOR[job?.status] || STATUS_COLOR.pending
  const errorId = error ? 'send-email-error' : undefined

  return (
    <Layout>
      <div className="p-8 max-w-2xl">
        <h1 className="text-lg font-semibold mb-1" style={{ color: 'var(--text)' }}>Send Email</h1>
        <p className="text-sm mb-8" style={{ color: 'var(--muted)' }}>Test your email pipeline end to end.</p>

        <div className="max-w-lg">
          <form onSubmit={handleSend} className="space-y-3" aria-describedby={errorId}>
            <div>
              <label htmlFor="send-key-prefix" className="block text-xs mb-1.5" style={{ color: 'var(--muted)' }}>API Key prefix</label>
              <select
                id="send-key-prefix"
                value={selectedKey}
                onChange={(e) => setSelectedKey(e.target.value)}
                className="w-full px-3 py-2 rounded-lg text-sm outline-none mono"
                style={{ background: 'var(--surface)', border: '1px solid var(--border)', color: 'var(--text)' }}
              >
                {keys.length === 0 && <option value="">No keys - create one first</option>}
                {keys.map((key) => (
                  <option key={key.id} value={key.prefix}>mk_live_{key.prefix}... ({key.name})</option>
                ))}
              </select>
            </div>

            <div>
              <label htmlFor="send-full-key" className="block text-xs mb-1.5" style={{ color: 'var(--muted)' }}>Full API Key</label>
              <input
                id="send-full-key"
                ref={apiKeyInputRef}
                type="password"
                placeholder="mk_live_..."
                autoComplete="off"
                spellCheck="false"
                aria-invalid={Boolean(error)}
                aria-describedby={errorId}
                onChange={(e) => { apiKeyRef.current = e.target.value }}
                className="w-full px-3 py-2 rounded-lg text-sm outline-none mono"
                style={{ background: 'var(--surface)', border: '1px solid var(--border)', color: 'var(--text)' }}
                onFocus={(e) => { e.target.style.borderColor = 'var(--accent)' }}
                onBlur={(e) => { e.target.style.borderColor = 'var(--border)' }}
              />
              <p className="text-xs mt-1" style={{ color: 'var(--muted)' }}>Paste the full key from when you created it. Never shown again.</p>
            </div>

            <Field id="send-to" label="To" type="email" placeholder="recipient@example.com" value={form.to} errorId={errorId} error={error} onChange={(value) => setForm((current) => ({ ...current, to: value }))} />
            <Field id="send-subject" label="Subject" placeholder="Hello from Sendr" value={form.subject} errorId={errorId} error={error} onChange={(value) => setForm((current) => ({ ...current, subject: value }))} />

            <div>
              <label htmlFor="send-body" className="block text-xs mb-1.5" style={{ color: 'var(--muted)' }}>Body</label>
              <textarea
                id="send-body"
                rows={4}
                placeholder="Email body..."
                value={form.body}
                aria-invalid={Boolean(error)}
                aria-describedby={errorId}
                onChange={(e) => setForm((current) => ({ ...current, body: e.target.value }))}
                className="w-full px-3 py-2 rounded-lg text-sm outline-none resize-none"
                style={{ background: 'var(--surface)', border: '1px solid var(--border)', color: 'var(--text)' }}
                onFocus={(e) => { e.target.style.borderColor = 'var(--accent)' }}
                onBlur={(e) => { e.target.style.borderColor = 'var(--border)' }}
              />
            </div>

            {error && (
              <div id="send-email-error" ref={errorRef} role="alert" tabIndex={-1} className="text-xs px-3 py-2 rounded-lg outline-none" style={{ background: 'var(--danger-dim)', color: 'var(--danger)', border: '1px solid rgba(255,77,77,0.2)' }}>
                {error}
              </div>
            )}

            <button type="submit" disabled={sending || !form.to || !form.subject || !form.body} className="w-full py-2.5 rounded-lg text-sm font-medium transition-opacity disabled:opacity-40" style={{ background: 'var(--accent)', color: '#000' }}>
              {sending ? 'Queuing...' : 'Send email'}
            </button>
          </form>

          {job && (
            <div className="mt-6 rounded-xl border p-4" style={{ background: statusStyle.bg, borderColor: statusStyle.border }} role="status">
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs font-medium" style={{ color: statusStyle.color }}>
                  {job.status === 'pending' && 'Queued - waiting for worker'}
                  {job.status === 'processing' && 'Processing...'}
                  {job.status === 'sent' && 'Delivered'}
                  {job.status === 'failed' && 'Failed - moved to DLQ'}
                </span>
                {(job.status === 'pending' || job.status === 'processing') && (
                  <div className="w-3 h-3 rounded-full border border-t-transparent animate-spin" style={{ borderColor: statusStyle.color, borderTopColor: 'transparent' }} />
                )}
              </div>
              <div className="text-xs mono" style={{ color: 'var(--muted)' }}>job_id: {job.id}</div>
            </div>
          )}
        </div>

        <div className="mt-12">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-sm font-semibold" style={{ color: 'var(--text)' }}>History</h2>
            <div className="flex gap-1.5" role="group" aria-label="Filter email history">
              {['', 'pending', 'processing', 'sent', 'failed'].map((filter) => (
                <button
                  key={filter}
                  type="button"
                  onClick={() => handleFilterChange(filter)}
                  className="text-xs px-2.5 py-1 rounded-md transition-colors"
                  style={{
                    background: statusFilter === filter ? 'var(--accent)' : 'var(--surface)',
                    color: statusFilter === filter ? '#000' : 'var(--muted)',
                    border: '1px solid var(--border)',
                  }}
                >
                  {filter === '' ? 'All' : filter.charAt(0).toUpperCase() + filter.slice(1)}
                </button>
              ))}
            </div>
          </div>

          {historyLoading && history.length === 0 ? (
            <div className="text-xs py-8 text-center" style={{ color: 'var(--muted)' }}>Loading...</div>
          ) : history.length === 0 ? (
            <div className="text-xs py-8 text-center" style={{ color: 'var(--muted)' }}>No emails yet.</div>
          ) : (
            <>
              <div className="rounded-xl overflow-hidden" style={{ border: '1px solid var(--border)' }}>
                <table className="w-full text-xs" style={{ borderCollapse: 'collapse' }}>
                  <thead>
                    <tr style={{ background: 'var(--surface)', borderBottom: '1px solid var(--border)' }}>
                      <th className="text-left px-4 py-2.5 font-medium" style={{ color: 'var(--muted)' }}>Job ID</th>
                      <th className="text-left px-4 py-2.5 font-medium" style={{ color: 'var(--muted)' }}>Status</th>
                      <th className="text-left px-4 py-2.5 font-medium" style={{ color: 'var(--muted)' }}>Retries</th>
                      <th className="text-left px-4 py-2.5 font-medium" style={{ color: 'var(--muted)' }}>Created</th>
                    </tr>
                  </thead>
                  <tbody>
                    {history.map((item, index) => {
                      const color = STATUS_COLOR[item.status] || STATUS_COLOR.pending
                      return (
                        <tr key={item.id} style={{ borderBottom: index < history.length - 1 ? '1px solid var(--border)' : 'none' }}>
                          <td className="px-4 py-3 mono" style={{ color: 'var(--muted)' }}>{item.id.slice(0, 8)}...</td>
                          <td className="px-4 py-3">
                            <span className="px-2 py-0.5 rounded-full text-xs" style={{ background: color.bg, color: color.color, border: `1px solid ${color.border}` }}>
                              {STATUS_LABEL[item.status] ?? item.status}
                            </span>
                          </td>
                          <td className="px-4 py-3" style={{ color: 'var(--text)' }}>{item.retries}</td>
                          <td className="px-4 py-3" style={{ color: 'var(--muted)' }}>{new Date(item.created_at).toLocaleString()}</td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
              {hasMoreHistory && (
                <button type="button" onClick={loadMoreHistory} disabled={historyLoading} className="mt-3 px-3 py-2 rounded-lg text-xs font-medium disabled:opacity-50" style={{ background: 'var(--surface)', color: 'var(--text)', border: '1px solid var(--border)' }}>
                  {historyLoading ? 'Loading...' : 'Load more'}
                </button>
              )}
            </>
          )}
        </div>
      </div>
    </Layout>
  )
}

function Field({ id, label, type = 'text', placeholder, value, errorId, error, onChange }) {
  return (
    <div>
      <label htmlFor={id} className="block text-xs mb-1.5" style={{ color: 'var(--muted)' }}>{label}</label>
      <input
        id={id}
        type={type}
        placeholder={placeholder}
        value={value}
        aria-invalid={Boolean(error)}
        aria-describedby={errorId}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2 rounded-lg text-sm outline-none"
        style={{ background: 'var(--surface)', border: '1px solid var(--border)', color: 'var(--text)' }}
        onFocus={(e) => { e.target.style.borderColor = 'var(--accent)' }}
        onBlur={(e) => { e.target.style.borderColor = 'var(--border)' }}
      />
    </div>
  )
}
