import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../lib/api'
import Layout from '../components/Layout'

const PREFS_KEY = 'sendr:profile-preferences'

function loadPreferences() {
  try {
    return {
      productUpdates: true,
      deliveryAlerts: true,
      compactDashboard: false,
      ...(JSON.parse(localStorage.getItem(PREFS_KEY)) || {}),
    }
  } catch {
    return { productUpdates: true, deliveryAlerts: true, compactDashboard: false }
  }
}

export default function Profile() {
  const [profile, setProfile] = useState(null)
  const [form, setForm] = useState({ name: '' })
  const [prefs, setPrefs] = useState(loadPreferences)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState(null)
  const [success, setSuccess] = useState(null)

  const errorRef = useRef(null)
  const successRef = useRef(null)

  useEffect(() => {
    api
      .me()
      .then((data) => {
        setProfile(data)
        setForm({ name: data?.name ?? '' })
      })
      .catch((err) => setError(err.message))
  }, [])

  useEffect(() => {
    localStorage.setItem(PREFS_KEY, JSON.stringify(prefs))
  }, [prefs])

  useEffect(() => {
    if (error) errorRef.current?.focus()
  }, [error])

  useEffect(() => {
    if (success) successRef.current?.focus()
  }, [success])

  const initials = useMemo(() => {
    const source = form.name || profile?.email || 'U'
    return source
      .split(/\s+/)
      .filter(Boolean)
      .slice(0, 2)
      .map((part) => part[0]?.toUpperCase())
      .join('')
  }, [form.name, profile?.email])

  async function handleSave(e) {
    e.preventDefault()
    const name = form.name.trim()
    if (name.length < 2 || name.length > 80) {
      setError('Name must be between 2 and 80 characters.')
      setSuccess(null)
      return
    }

    setSaving(true)
    setError(null)
    setSuccess(null)
    try {
      const updated = await api.updateProfile({ name })
      setProfile(updated)
      setForm({ name: updated.name ?? name })
      setSuccess('Profile updated.')
    } catch (err) {
      setError(err.message)
    } finally {
      setSaving(false)
    }
  }

  function updatePref(key) {
    setPrefs((current) => ({ ...current, [key]: !current[key] }))
  }

  if (!profile && !error) {
    return (
      <Layout>
        <div className="p-8">
          <div className="max-w-4xl space-y-3 animate-pulse" role="status" aria-label="Loading profile">
            <div className="h-5 w-28 rounded" style={{ background: 'var(--border)' }} />
            <div className="h-48 rounded-xl" style={{ background: 'var(--surface)' }} />
          </div>
        </div>
      </Layout>
    )
  }

  if (!profile) {
    return (
      <Layout>
        <div className="p-8 text-sm" role="alert" style={{ color: 'var(--danger)' }}>{error}</div>
      </Layout>
    )
  }

  const joined = profile?.created_at ? new Date(profile.created_at).toLocaleDateString() : 'Unknown'
  const dailyLimit = profile?.daily_limit < 0 ? 'Unlimited' : profile?.daily_limit?.toLocaleString()
  const maxKeys = profile?.max_api_keys < 0 ? 'Unlimited' : profile?.max_api_keys?.toLocaleString()

  return (
    <Layout>
      <div className="p-8 max-w-5xl">
        <div className="flex flex-col md:flex-row md:items-end md:justify-between gap-4 mb-8">
          <div>
            <h1 className="text-lg font-semibold mb-1" style={{ color: 'var(--text)' }}>Profile</h1>
            <p className="text-sm" style={{ color: 'var(--muted)' }}>Manage your account identity, preferences, and plan details.</p>
          </div>
          <Link to="/pricing" className="self-start md:self-auto px-3 py-2 rounded-lg text-sm font-medium" style={{ background: 'var(--surface)', color: 'var(--text)', border: '1px solid var(--border)' }}>
            Manage plan
          </Link>
        </div>

        {error && (
          <div ref={errorRef} tabIndex={-1} role="alert" className="text-xs mb-4 px-3 py-2 rounded-lg outline-none" style={{ background: 'var(--danger-dim)', color: 'var(--danger)', border: '1px solid rgba(255,77,77,0.2)' }}>
            {error}
          </div>
        )}

        {success && (
          <div ref={successRef} tabIndex={-1} role="status" className="text-xs mb-4 px-3 py-2 rounded-lg outline-none" style={{ background: 'var(--accent-dim)', color: 'var(--accent)', border: '1px solid var(--accent-border)' }}>
            {success}
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-[1.1fr_0.9fr] gap-4">
          <section className="rounded-xl border p-5" style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
            <div className="flex items-center gap-4 mb-6">
              <div className="w-14 h-14 rounded-xl flex items-center justify-center text-lg font-semibold mono" style={{ background: 'var(--accent-dim)', color: 'var(--accent)', border: '1px solid var(--accent-border)' }}>
                {initials}
              </div>
              <div>
                <div className="text-sm font-semibold" style={{ color: 'var(--text)' }}>{profile?.name}</div>
                <div className="text-xs" style={{ color: 'var(--muted)' }}>{profile?.email}</div>
              </div>
            </div>

            <form onSubmit={handleSave} className="space-y-4">
              <div>
                <label htmlFor="profile-name" className="block text-xs mb-1.5" style={{ color: 'var(--muted)' }}>Display name</label>
                <input
                  id="profile-name"
                  value={form.name}
                  onChange={(e) => setForm({ name: e.target.value })}
                  className="w-full px-3 py-2 rounded-lg text-sm outline-none"
                  style={{ background: 'var(--bg)', border: '1px solid var(--border)', color: 'var(--text)' }}
                  onFocus={(e) => { e.target.style.borderColor = 'var(--accent)' }}
                  onBlur={(e) => { e.target.style.borderColor = 'var(--border)' }}
                />
              </div>

              <div>
                <label htmlFor="profile-email" className="block text-xs mb-1.5" style={{ color: 'var(--muted)' }}>Email address</label>
                <input
                  id="profile-email"
                  value={profile?.email ?? ''}
                  readOnly
                  className="w-full px-3 py-2 rounded-lg text-sm outline-none"
                  style={{ background: 'var(--bg)', border: '1px solid var(--border)', color: 'var(--muted)' }}
                />
              </div>

              <button type="submit" disabled={saving || form.name.trim() === profile?.name} className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity disabled:opacity-40" style={{ background: 'var(--accent)', color: '#000' }}>
                {saving ? 'Saving...' : 'Save profile'}
              </button>
            </form>
          </section>

          <section className="rounded-xl border p-5" style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
            <h2 className="text-sm font-semibold mb-4" style={{ color: 'var(--text)' }}>Account details</h2>
            <div className="space-y-3">
              <Detail label="Plan" value={profile?.plan} />
              <Detail label="Daily send limit" value={dailyLimit} />
              <Detail label="API key limit" value={maxKeys} />
              <Detail label="Rate wait" value={`${profile?.rate_wait_secs ?? 0}s`} />
              <Detail label="Joined" value={joined} />
            </div>
          </section>
        </div>

        <section className="mt-4 rounded-xl border p-5" style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
          <h2 className="text-sm font-semibold mb-4" style={{ color: 'var(--text)' }}>Experience preferences</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <Toggle label="Product updates" checked={prefs.productUpdates} onChange={() => updatePref('productUpdates')} />
            <Toggle label="Delivery alerts" checked={prefs.deliveryAlerts} onChange={() => updatePref('deliveryAlerts')} />
            <Toggle label="Compact dashboard" checked={prefs.compactDashboard} onChange={() => updatePref('compactDashboard')} />
          </div>
        </section>
      </div>
    </Layout>
  )
}

function Detail({ label, value }) {
  return (
    <div className="flex items-center justify-between gap-4 text-sm">
      <span style={{ color: 'var(--muted)' }}>{label}</span>
      <span className="font-medium text-right capitalize" style={{ color: 'var(--text)' }}>{value}</span>
    </div>
  )
}

function Toggle({ label, checked, onChange }) {
  return (
    <label className="flex items-center justify-between gap-3 rounded-lg px-3 py-3 cursor-pointer" style={{ background: 'var(--bg)', border: '1px solid var(--border)' }}>
      <span className="text-sm" style={{ color: 'var(--text)' }}>{label}</span>
      <input type="checkbox" checked={checked} onChange={onChange} className="sr-only" />
      <span className="relative w-9 h-5 rounded-full transition-colors" style={{ background: checked ? 'var(--accent)' : 'var(--border)' }} aria-hidden="true">
        <span className="absolute top-0.5 w-4 h-4 rounded-full transition-all" style={{ left: checked ? '18px' : '2px', background: checked ? '#000' : 'var(--muted)' }} />
      </span>
    </label>
  )
}
