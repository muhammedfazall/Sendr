import { useCallback, useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../lib/api'
import Layout from '../components/Layout'

const STATUS_COLOR = {
  pending: { color: '#f59e0b', bg: 'rgba(245,158,11,0.08)', border: 'rgba(245,158,11,0.2)' },
  processing: { color: '#60a5fa', bg: 'rgba(96,165,250,0.08)', border: 'rgba(96,165,250,0.2)' },
  sent: { color: '#00d084', bg: 'rgba(0,208,132,0.08)', border: 'rgba(0,208,132,0.2)' },
  failed: { color: '#ff4d4d', bg: 'rgba(255,77,77,0.08)', border: 'rgba(255,77,77,0.2)' },
}

function validateEmailForm(form) {
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.to.trim())) return 'Enter a valid recipient email.'
  if (!form.subject.trim()) return 'Subject is required.'
  if (form.subject.length > 998) return 'Subject must be under 998 characters.'
  if (!form.textBody.trim() && !form.htmlBody.trim()) return 'Text or HTML body is required.'
  if (form.textBody.length > 50000) return 'Text body must be under 50 KB.'
  if (form.htmlBody.length > 50000) return 'HTML body must be under 50 KB.'
  return null
}

function tryParseJson(value) {
  try {
    return JSON.parse(value)
  } catch {
    return null
  }
}

export default function SendEmail() {
  const [keys, setKeys] = useState([])
  const [selectedKey, setSelectedKey] = useState('')
  const [form, setForm] = useState({ to: '', subject: '', textBody: '', htmlBody: '', bodyType: 'text' })
  const [sending, setSending] = useState(false)
  const [job, setJob] = useState(null)
  const [error, setError] = useState(null)
  const [templates, setTemplates] = useState([])
  const [selectedTemplateId, setSelectedTemplateId] = useState('')
  const [templateDataText, setTemplateDataText] = useState('')

  const pollRef = useRef(null)
  const apiKeyRef = useRef('')
  const apiKeyInputRef = useRef(null)
  const errorRef = useRef(null)

  const clearApiKeyInput = useCallback(() => {
    apiKeyRef.current = ''
    if (apiKeyInputRef.current) apiKeyInputRef.current.value = ''
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
    api.listTemplates()
      .then((data) => setTemplates(data ?? []))
      .catch(() => {})
  }, [])

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
      } catch {
        clearInterval(pollRef.current)
        pollRef.current = null
      }
    }, 2000)

    return () => {
      clearInterval(pollRef.current)
      pollRef.current = null
    }
  }, [job])

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
      const payload = {
        to: [form.to.trim()],
        subject: form.subject.trim(),
      }
      if (form.textBody.trim()) payload.text_body = form.textBody
      if (form.htmlBody.trim()) payload.html_body = form.htmlBody

      if (selectedTemplateId) {
        payload.template_id = selectedTemplateId
        const parsed = tryParseJson(templateDataText)
        if (parsed) payload.template_data = parsed
      }

      const data = await api.sendEmail(payload, fullKey)
      setJob({ ...data, id: data.job_id, status: 'pending', _apiKey: fullKey })
      clearApiKeyInput()
    } catch (err) {
      setError(err.message)
    } finally {
      setSending(false)
    }
  }, [clearApiKeyInput, form, selectedKey])

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

            {templates.length > 0 && (
              <div>
                <label htmlFor="send-template" className="block text-xs mb-1.5" style={{ color: 'var(--muted)' }}>Template (optional)</label>
                <select
                  id="send-template"
                  value={selectedTemplateId}
                  onChange={(e) => setSelectedTemplateId(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg text-sm outline-none"
                  style={{ background: 'var(--surface)', border: '1px solid var(--border)', color: 'var(--text)' }}
                >
                  <option value="">None — write inline</option>
                  {templates.map((tpl) => (
                    <option key={tpl.id} value={tpl.id}>{tpl.name}</option>
                  ))}
                </select>
                {templates.length > 0 && (
                  <Link to="/templates" className="text-xs mt-1 inline-block" style={{ color: 'var(--accent)' }}>Manage templates</Link>
                )}
              </div>
            )}

            {selectedTemplateId && (
              <div>
                <label htmlFor="template-data" className="block text-xs mb-1.5" style={{ color: 'var(--muted)' }}>Template data (JSON)</label>
                <textarea
                  id="template-data"
                  rows={3}
                  value={templateDataText}
                  onChange={(e) => setTemplateDataText(e.target.value)}
                  placeholder={'{"Name": "Alice", "Message": "Welcome!"}'}
                  className="w-full px-3 py-2 rounded-lg text-sm outline-none resize-none font-mono"
                  style={{ background: 'var(--surface)', border: '1px solid var(--border)', color: 'var(--text)' }}
                />
                <p className="text-xs mt-1" style={{ color: 'var(--muted)' }}>Available as {'{{.Name}}'}, {'{{.Message}}'} in the template.</p>
              </div>
            )}

            <Field id="send-to" label="To" type="email" placeholder="recipient@example.com" value={form.to} errorId={errorId} error={error} onChange={(value) => setForm((current) => ({ ...current, to: value }))} />
            <Field id="send-subject" label="Subject" placeholder="Hello from Sendr" value={form.subject} errorId={errorId} error={error} onChange={(value) => setForm((current) => ({ ...current, subject: value }))} />

            <div>
              <div className="flex items-center gap-2 mb-1.5">
                <label className="block text-xs" style={{ color: 'var(--muted)' }}>Body</label>
                <button
                  type="button"
                  onClick={() => setForm((current) => ({ ...current, bodyType: current.bodyType === 'text' ? 'html' : 'text' }))}
                  className="text-xs px-2 py-0.5 rounded transition-colors"
                  style={{
                    background: 'var(--surface)',
                    color: 'var(--accent)',
                    border: '1px solid var(--border)',
                  }}
                >
                  {form.bodyType === 'text' ? 'Switch to HTML' : 'Switch to Text'}
                </button>
              </div>
              {form.bodyType === 'text' ? (
                <textarea
                  id="send-body"
                  rows={4}
                  placeholder="Email body..."
                  value={form.textBody}
                  aria-invalid={Boolean(error)}
                  aria-describedby={errorId}
                  onChange={(e) => setForm((current) => ({ ...current, textBody: e.target.value }))}
                  className="w-full px-3 py-2 rounded-lg text-sm outline-none resize-none font-mono"
                  style={{ background: 'var(--surface)', border: '1px solid var(--border)', color: 'var(--text)' }}
                  onFocus={(e) => { e.target.style.borderColor = 'var(--accent)' }}
                  onBlur={(e) => { e.target.style.borderColor = 'var(--border)' }}
                />
              ) : (
                <textarea
                  id="send-html-body"
                  rows={4}
                  placeholder={'<html><body><p>Hello!</p></body></html>'}
                  value={form.htmlBody}
                  aria-invalid={Boolean(error)}
                  aria-describedby={errorId}
                  onChange={(e) => setForm((current) => ({ ...current, htmlBody: e.target.value }))}
                  className="w-full px-3 py-2 rounded-lg text-sm outline-none resize-none font-mono"
                  style={{ background: 'var(--surface)', border: '1px solid var(--border)', color: 'var(--text)' }}
                  onFocus={(e) => { e.target.style.borderColor = 'var(--accent)' }}
                  onBlur={(e) => { e.target.style.borderColor = 'var(--border)' }}
                />
              )}
            </div>

            {error && (
              <div id="send-email-error" ref={errorRef} role="alert" tabIndex={-1} className="text-xs px-3 py-2 rounded-lg outline-none" style={{ background: 'var(--danger-dim)', color: 'var(--danger)', border: '1px solid rgba(255,77,77,0.2)' }}>
                {error}
              </div>
            )}

            <button type="submit" disabled={sending || !form.to || !form.subject || (!form.textBody && !form.htmlBody)} className="w-full py-2.5 rounded-lg text-sm font-medium transition-opacity disabled:opacity-40" style={{ background: 'var(--accent)', color: '#000' }}>
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
