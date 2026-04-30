import { useState, useEffect, useRef, useCallback } from 'react'
import { api } from '../lib/api'
import Layout from '../components/Layout'

const STATUS_COLOR = {
  pending:    { color: '#f59e0b', bg: 'rgba(245,158,11,0.08)',   border: 'rgba(245,158,11,0.2)' },
  processing: { color: '#60a5fa', bg: 'rgba(96,165,250,0.08)',   border: 'rgba(96,165,250,0.2)' },
  sent:       { color: '#00d084', bg: 'rgba(0,208,132,0.08)',    border: 'rgba(0,208,132,0.2)'  },
  failed:     { color: '#ff4d4d', bg: 'rgba(255,77,77,0.08)',    border: 'rgba(255,77,77,0.2)'  },
}

const STATUS_LABEL = {
  pending: '⏳ Pending', processing: '⚡ Processing', sent: '✓ Sent', failed: '✗ Failed',
}

export default function SendEmail() {
  const [keys, setKeys] = useState([])
  const [selectedKey, setSelectedKey] = useState('')
  const [form, setForm] = useState({ to: '', subject: '', body: '' })
  const [sending, setSending] = useState(false)
  const [job, setJob] = useState(null)
  const [error, setError] = useState(null)
  const pollRef = useRef(null)
  const apiKeyRef = useRef('')  // API key stays in memory only — never persisted

  const [history, setHistory] = useState([])
  const [historyLoading, setHistoryLoading] = useState(false)
  const [statusFilter, setStatusFilter] = useState('')

  const fetchHistory = useCallback((filter = '') => {
    setHistoryLoading(true)
    api.listEmails(filter)
      .then(data => setHistory(data ?? []))
      .catch(() => setHistory([]))
      .finally(() => setHistoryLoading(false))
  }, [])

  const loaded = useRef(false)

  useEffect(() => {
    api.listKeys().then(data => {
      const k = data ?? []
      setKeys(k)
      if (k.length > 0) setSelectedKey(k[0].prefix)
    }).catch(() => {})
  }, [])

  useEffect(() => {
    if (!loaded.current) {
      loaded.current = true
      fetchHistory('')
    }
  }, [fetchHistory])

  useEffect(() => {
    if (!job || job.status === 'sent' || job.status === 'failed') {
      clearInterval(pollRef.current)
      return
    }
    clearInterval(pollRef.current)
    pollRef.current = setInterval(async () => {
      try {
        const updated = await api.getJob(job.id, job._apiKey)
        setJob(j => ({ ...updated, _apiKey: j._apiKey }))
      } catch {
        clearInterval(pollRef.current)
      }
    }, 2000)
    return () => clearInterval(pollRef.current)
  }, [job])

  async function handleSend(e) {
    e.preventDefault()
    if (!selectedKey) return setError('Select an API key first')
    setSending(true)
    setError(null)
    setJob(null)

    const fullKey = apiKeyRef.current
    if (!fullKey) {
      setSending(false)
      setError('Paste your full API key in the field above first.')
      return
    }

    try {
      const data = await api.sendEmail(form, fullKey)
      setJob({ ...data, id: data.job_id, status: 'pending', _apiKey: fullKey })
      fetchHistory()
    } catch (e) {
      setError(e.message)
    } finally {
      setSending(false)
    }
  }

  function handleFilterChange(f) {
    setStatusFilter(f)
    fetchHistory(f)
  }

  const s = STATUS_COLOR[job?.status] || STATUS_COLOR.pending

  return (
    <Layout>
      <div className="p-8 max-w-2xl">
        <h1 className="text-lg font-semibold mb-1" style={{ color: 'var(--text)' }}>Send Email</h1>
        <p className="text-sm mb-8" style={{ color: 'var(--muted)' }}>
          Test your email pipeline end to end.
        </p>

        <div className="max-w-lg">
          <form onSubmit={handleSend} className="space-y-3">
            <div>
              <label className="block text-xs mb-1.5" style={{ color: 'var(--muted)' }}>API Key (prefix)</label>
              <select value={selectedKey} onChange={e => setSelectedKey(e.target.value)}
                className="w-full px-3 py-2 rounded-lg text-sm outline-none mono"
                style={{ background: 'var(--surface)', border: '1px solid var(--border)', color: 'var(--text)' }}>
                {keys.length === 0 && <option value="">No keys — create one first</option>}
                {keys.map(k => (
                  <option key={k.id} value={k.prefix}>mk_live_{k.prefix}… ({k.name})</option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-xs mb-1.5" style={{ color: 'var(--muted)' }}>
                Full API Key <span style={{ color: 'var(--accent)' }}>*</span>
              </label>
              <input
                type="password"
                placeholder="mk_live_..."
                onChange={e => apiKeyRef.current = e.target.value}
                className="w-full px-3 py-2 rounded-lg text-sm outline-none mono"
                style={{ background: 'var(--surface)', border: '1px solid var(--border)', color: 'var(--text)' }}
                onFocus={e => e.target.style.borderColor = 'var(--accent)'}
                onBlur={e => e.target.style.borderColor = 'var(--border)'}
              />
              <p className="text-xs mt-1" style={{ color: 'var(--muted)' }}>
                Paste the full key from when you created it. Never shown again.
              </p>
            </div>

            <Field label="To" type="email" placeholder="recipient@example.com"
              value={form.to} onChange={v => setForm(f => ({ ...f, to: v }))} />
            <Field label="Subject" placeholder="Hello from Sendr"
              value={form.subject} onChange={v => setForm(f => ({ ...f, subject: v }))} />

            <div>
              <label className="block text-xs mb-1.5" style={{ color: 'var(--muted)' }}>Body</label>
              <textarea
                rows={4}
                placeholder="Email body..."
                value={form.body}
                onChange={e => setForm(f => ({ ...f, body: e.target.value }))}
                className="w-full px-3 py-2 rounded-lg text-sm outline-none resize-none"
                style={{ background: 'var(--surface)', border: '1px solid var(--border)', color: 'var(--text)' }}
                onFocus={e => e.target.style.borderColor = 'var(--accent)'}
                onBlur={e => e.target.style.borderColor = 'var(--border)'}
              />
            </div>

            {error && (
              <div className="text-xs px-3 py-2 rounded-lg"
                style={{ background: 'var(--danger-dim)', color: 'var(--danger)', border: '1px solid rgba(255,77,77,0.2)' }}>
                {error}
              </div>
            )}

            <button type="submit" disabled={sending || !form.to || !form.subject || !form.body}
              className="w-full py-2.5 rounded-lg text-sm font-medium transition-opacity disabled:opacity-40"
              style={{ background: 'var(--accent)', color: '#000' }}>
              {sending ? 'Queuing…' : 'Send email'}
            </button>
          </form>

          {job && (
            <div className="mt-6 rounded-xl border p-4"
              style={{ background: s.bg, borderColor: s.border }}>
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs font-medium" style={{ color: s.color }}>
                  {job.status === 'pending' && '⏳ Queued — waiting for worker'}
                  {job.status === 'processing' && '⚡ Processing…'}
                  {job.status === 'sent' && '✓ Delivered'}
                  {job.status === 'failed' && '✗ Failed — moved to DLQ'}
                </span>
                {(job.status === 'pending' || job.status === 'processing') && (
                  <div className="w-3 h-3 rounded-full border border-t-transparent animate-spin"
                    style={{ borderColor: s.color, borderTopColor: 'transparent' }} />
                )}
              </div>
              <div className="text-xs mono" style={{ color: 'var(--muted)' }}>
                job_id: {job.id}
              </div>
            </div>
          )}
        </div>

        {/* History table */}
        <div className="mt-12">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-sm font-semibold" style={{ color: 'var(--text)' }}>History</h2>
            <div className="flex gap-1.5">
              {['', 'pending', 'sent', 'failed'].map(f => (
                <button key={f}
                  onClick={() => handleFilterChange(f)}
                  className="text-xs px-2.5 py-1 rounded-md transition-colors"
                  style={{
                    background: statusFilter === f ? 'var(--accent)' : 'var(--surface)',
                    color: statusFilter === f ? '#000' : 'var(--muted)',
                    border: '1px solid var(--border)',
                  }}>
                  {f === '' ? 'All' : f.charAt(0).toUpperCase() + f.slice(1)}
                </button>
              ))}
            </div>
          </div>

          {historyLoading ? (
            <div className="text-xs py-8 text-center" style={{ color: 'var(--muted)' }}>Loading…</div>
          ) : history.length === 0 ? (
            <div className="text-xs py-8 text-center" style={{ color: 'var(--muted)' }}>No emails yet.</div>
          ) : (
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
                  {history.map((j, i) => {
                    const c = STATUS_COLOR[j.status] || STATUS_COLOR.pending
                    return (
                      <tr key={j.id}
                        style={{ borderBottom: i < history.length - 1 ? '1px solid var(--border)' : 'none' }}>
                        <td className="px-4 py-3 mono" style={{ color: 'var(--muted)' }}>
                          {j.id.slice(0, 8)}…
                        </td>
                        <td className="px-4 py-3">
                          <span className="px-2 py-0.5 rounded-full text-xs"
                            style={{ background: c.bg, color: c.color, border: `1px solid ${c.border}` }}>
                            {STATUS_LABEL[j.status] ?? j.status}
                          </span>
                        </td>
                        <td className="px-4 py-3" style={{ color: 'var(--text)' }}>{j.retries}</td>
                        <td className="px-4 py-3" style={{ color: 'var(--muted)' }}>
                          {new Date(j.created_at).toLocaleString()}
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </Layout>
  )
}

function Field({ label, type = 'text', placeholder, value, onChange }) {
  return (
    <div>
      <label className="block text-xs mb-1.5" style={{ color: 'var(--muted)' }}>{label}</label>
      <input type={type} placeholder={placeholder} value={value}
        onChange={e => onChange(e.target.value)}
        className="w-full px-3 py-2 rounded-lg text-sm outline-none"
        style={{ background: 'var(--surface)', border: '1px solid var(--border)', color: 'var(--text)' }}
        onFocus={e => e.target.style.borderColor = 'var(--accent)'}
        onBlur={e => e.target.style.borderColor = 'var(--border)'}
      />
    </div>
  )
}