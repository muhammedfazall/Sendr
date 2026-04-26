import { useState, useEffect, useRef } from 'react'
import { api } from '../lib/api'
import Layout from '../components/Layout'

const STATUS_COLOR = {
  pending:    { color: '#f59e0b', bg: 'rgba(245,158,11,0.08)',   border: 'rgba(245,158,11,0.2)' },
  processing: { color: '#60a5fa', bg: 'rgba(96,165,250,0.08)',   border: 'rgba(96,165,250,0.2)' },
  sent:       { color: '#00d084', bg: 'rgba(0,208,132,0.08)',    border: 'rgba(0,208,132,0.2)'  },
  failed:     { color: '#ff4d4d', bg: 'rgba(255,77,77,0.08)',    border: 'rgba(255,77,77,0.2)'  },
}

export default function SendEmail() {
  const [keys, setKeys] = useState([])
  const [selectedKey, setSelectedKey] = useState('')
  const [form, setForm] = useState({ to: '', subject: '', body: '' })
  const [sending, setSending] = useState(false)
  const [job, setJob] = useState(null)
  const [error, setError] = useState(null)
  const pollRef = useRef(null)

  useEffect(() => {
    api.listKeys().then(data => {
      const k = data ?? []
      setKeys(k)
      if (k.length > 0) setSelectedKey(k[0].prefix)
    }).catch(() => {})

    return () => clearInterval(pollRef.current)
  }, [])

  // Poll job status every 2s until terminal state
  useEffect(() => {
    if (!job || job.status === 'sent' || job.status === 'failed') {
      clearInterval(pollRef.current)
      return
    }
    clearInterval(pollRef.current)
    pollRef.current = setInterval(async () => {
      try {
        // We need the full key to call /emails/:id — store it when sending
        const updated = await api.getJob(job.id, job._apiKey)
        setJob(j => ({ ...updated, _apiKey: j._apiKey }))
      } catch (e) {
        clearInterval(pollRef.current)
      }
    }, 2000)
    return () => clearInterval(pollRef.current)
  }, [job?.status])

  async function handleSend(e) {
    e.preventDefault()
    if (!selectedKey) return setError('Select an API key first')
    setSending(true)
    setError(null)
    setJob(null)

    // Find the full key — user needs to paste it or we store it on create.
    // For now we surface a notice asking them to use the full key from the Keys page.
    const fullKey = localStorage.getItem(`apiKey_${selectedKey}`)
    if (!fullKey) {
      setSending(false)
      setError('Full API key not found. Use the "Copy" button when creating a key, then paste it in the field below.')
      return
    }

    try {
      const data = await api.sendEmail(form, fullKey)
      setJob({ ...data, id: data.job_id, status: 'pending', _apiKey: fullKey })
    } catch (e) {
      setError(e.message)
    } finally {
      setSending(false)
    }
  }

  const s = STATUS_COLOR[job?.status] || STATUS_COLOR.pending

  return (
    <Layout>
      <div className="p-8 max-w-lg">
        <h1 className="text-lg font-semibold mb-1" style={{ color: 'var(--text)' }}>Send Email</h1>
        <p className="text-sm mb-8" style={{ color: 'var(--muted)' }}>
          Test your email pipeline end to end.
        </p>

        <form onSubmit={handleSend} className="space-y-3">
          {/* API key selector */}
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

          {/* Full key input */}
          <div>
            <label className="block text-xs mb-1.5" style={{ color: 'var(--muted)' }}>
              Full API Key <span style={{ color: 'var(--accent)' }}>*</span>
            </label>
            <input
              type="password"
              placeholder="mk_live_..."
              onChange={e => localStorage.setItem(`apiKey_${selectedKey}`, e.target.value)}
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

        {/* Job status */}
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