import { useCallback, useEffect, useState } from 'react'
import { api } from '../lib/api'
import Layout from '../components/Layout'

const emptyForm = { name: '', subject_template: '', html_template: '', text_template: '' }

export default function Templates() {
  const [templates, setTemplates] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [editing, setEditing] = useState(null)
  const [form, setForm] = useState(emptyForm)
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(null)

  const load = useCallback(() => {
    setLoading(true)
    api.listTemplates()
      .then((data) => setTemplates(data ?? []))
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  useEffect(load, [load])

  const handleSave = async (e) => {
    e.preventDefault()
    if (!form.name.trim()) return
    setSaving(true)
    setError(null)
    try {
      if (editing) {
        const updated = await api.updateTemplate(editing.id, form)
        setTemplates((prev) => prev.map((t) => (t.id === editing.id ? updated : t)))
      } else {
        const created = await api.createTemplate(form)
        setTemplates((prev) => [created, ...prev])
      }
      setEditing(null)
      setForm(emptyForm)
    } catch (err) {
      setError(err.message)
    } finally {
      setSaving(false)
    }
  }

  const handleEdit = (tpl) => {
    setEditing(tpl)
    setForm({ name: tpl.name, subject_template: tpl.subject_template, html_template: tpl.html_template, text_template: tpl.text_template })
    setError(null)
  }

  const handleDelete = async (id) => {
    setDeleting(id)
    try {
      await api.deleteTemplate(id)
      setTemplates((prev) => prev.filter((t) => t.id !== id))
    } catch (err) {
      setError(err.message)
    } finally {
      setDeleting(null)
    }
  }

  const handleCancel = () => {
    setEditing(null)
    setForm(emptyForm)
    setError(null)
  }

  const inputStyle = {
    background: 'var(--surface)',
    border: '1px solid var(--border)',
    color: 'var(--text)',
  }

  return (
    <Layout>
      <div className="p-8 max-w-4xl">
        <div className="flex items-center justify-between gap-4 mb-1">
          <h1 className="text-lg font-semibold" style={{ color: 'var(--text)' }}>Templates</h1>
        </div>
        <p className="text-sm mb-8" style={{ color: 'var(--muted)' }}>Create and manage reusable email templates with dynamic data.</p>

        {error && (
          <div role="alert" className="text-xs mb-4 px-3 py-2 rounded-lg" style={{ background: 'var(--danger-dim)', color: 'var(--danger)', border: '1px solid rgba(255,77,77,0.2)' }}>
            {error}
          </div>
        )}

        {/* Form */}
        <form onSubmit={handleSave} className="rounded-xl border p-5 mb-8" style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
          <h2 className="text-sm font-semibold mb-4" style={{ color: 'var(--text)' }}>{editing ? 'Edit template' : 'New template'}</h2>
          <div className="space-y-3">
            <div>
              <label className="block text-xs mb-1" style={{ color: 'var(--muted)' }}>Name</label>
              <input
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                placeholder="Weekly newsletter"
                className="w-full px-3 py-2 rounded-lg text-sm outline-none"
                style={inputStyle}
              />
            </div>
            <div>
              <label className="block text-xs mb-1" style={{ color: 'var(--muted)' }}>Subject template</label>
              <input
                value={form.subject_template}
                onChange={(e) => setForm((f) => ({ ...f, subject_template: e.target.value }))}
                placeholder={'Hello {{.Name}} — your weekly update'}
                className="w-full px-3 py-2 rounded-lg text-sm outline-none font-mono"
                style={inputStyle}
              />
            </div>
            <div>
              <label className="block text-xs mb-1" style={{ color: 'var(--muted)' }}>HTML template</label>
              <textarea
                rows={5}
                value={form.html_template}
                onChange={(e) => setForm((f) => ({ ...f, html_template: e.target.value }))}
                placeholder={'<html><body><h1>Hi {{.Name}}</h1><p>{{.Message}}</p></body></html>'}
                className="w-full px-3 py-2 rounded-lg text-sm outline-none resize-none font-mono"
                style={inputStyle}
              />
            </div>
            <div>
              <label className="block text-xs mb-1" style={{ color: 'var(--muted)' }}>Text template</label>
              <textarea
                rows={3}
                value={form.text_template}
                onChange={(e) => setForm((f) => ({ ...f, text_template: e.target.value }))}
                placeholder={'Hi {{.Name}} — {{.Message}}'}
                className="w-full px-3 py-2 rounded-lg text-sm outline-none resize-none font-mono"
                style={inputStyle}
              />
            </div>
            <div className="flex gap-2 pt-2">
              <button type="submit" disabled={saving || !form.name.trim()}
                className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity disabled:opacity-40"
                style={{ background: 'var(--accent)', color: '#000' }}
              >
                {saving ? 'Saving...' : editing ? 'Update' : 'Create'}
              </button>
              {editing && (
                <button type="button" onClick={handleCancel}
                  className="px-4 py-2 rounded-lg text-sm"
                  style={{ background: 'var(--bg)', color: 'var(--muted)', border: '1px solid var(--border)' }}
                >
                  Cancel
                </button>
              )}
            </div>
          </div>
        </form>

        {/* List */}
        {loading ? (
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="h-16 rounded-xl animate-pulse" style={{ background: 'var(--surface)' }} />
            ))}
          </div>
        ) : templates.length === 0 ? (
          <div className="rounded-xl border p-8 text-center text-sm" style={{ background: 'var(--surface)', borderColor: 'var(--border)', color: 'var(--muted)' }}>
            No templates yet. Create one above.
          </div>
        ) : (
          <div className="space-y-2">
            {templates.map((tpl) => (
              <div key={tpl.id} className="rounded-xl border p-4 flex items-center justify-between gap-4" style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
                <div className="min-w-0 flex-1">
                  <div className="text-sm font-medium truncate" style={{ color: 'var(--text)' }}>{tpl.name}</div>
                  <div className="text-xs mt-0.5" style={{ color: 'var(--muted)' }}>
                    {tpl.subject_template || '(no subject template)'}
                    <span className="ml-3">{new Date(tpl.created_at).toLocaleDateString()}</span>
                  </div>
                </div>
                <div className="flex gap-2 shrink-0">
                  <button onClick={() => handleEdit(tpl)}
                    className="px-3 py-1.5 rounded-lg text-xs transition-colors"
                    style={{ background: 'var(--bg)', color: 'var(--text)', border: '1px solid var(--border)' }}
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => handleDelete(tpl.id)}
                    disabled={deleting === tpl.id}
                    className="px-3 py-1.5 rounded-lg text-xs transition-colors disabled:opacity-40"
                    style={{ background: 'var(--danger-dim)', color: 'var(--danger)', border: '1px solid rgba(255,77,77,0.2)' }}
                  >
                    {deleting === tpl.id ? '...' : 'Delete'}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </Layout>
  )
}
