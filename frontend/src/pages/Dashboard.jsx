import { useState, useEffect } from 'react'
import { api } from '../lib/api'
import Layout from '../components/Layout'

export default function Dashboard() {
  const [profile, setProfile] = useState(null)
  const [used, setUsed] = useState(null)
  const [error, setError] = useState(null)

  useEffect(() => {
    api.me()
      .then(setProfile)
      .catch((e) => setError(e.message))
  }, [])

  // Fetch today's usage from Redis via a simple endpoint (fallback to 0)
  useEffect(() => {
    if (!profile) return
    // We don't have a /usage endpoint yet — show 0 for now
    setUsed(0)
  }, [profile])

  if (error) return (
    <Layout>
      <div className="p-8 text-sm" style={{ color: 'var(--danger)' }}>{error}</div>
    </Layout>
  )

  if (!profile) return (
    <Layout>
      <div className="p-8">
        <Skeleton />
      </div>
    </Layout>
  )

  const pct = Math.min((used / profile.daily_limit) * 100, 100)
  const planColor = { free: '#666', pro: '#00d084', max: '#a78bfa' }[profile.plan] || '#666'

  return (
    <Layout>
      <div className="p-8 max-w-2xl">
        <h1 className="text-lg font-semibold mb-1" style={{ color: 'var(--text)' }}>Dashboard</h1>
        <p className="text-sm mb-8" style={{ color: 'var(--muted)' }}>
          Welcome back, {profile.name?.split(' ')[0]}
        </p>

        {/* Profile card */}
        <div className="rounded-xl border p-5 mb-4"
          style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
          <div className="flex items-start justify-between mb-4">
            <div>
              <div className="text-sm font-medium mb-0.5" style={{ color: 'var(--text)' }}>{profile.name}</div>
              <div className="text-xs" style={{ color: 'var(--muted)' }}>{profile.email}</div>
            </div>
            <span className="text-xs px-2 py-0.5 rounded-full font-medium mono"
              style={{ background: `${planColor}18`, color: planColor, border: `1px solid ${planColor}30` }}>
              {profile.plan}
            </span>
          </div>

          {/* Usage bar */}
          <div>
            <div className="flex justify-between text-xs mb-2" style={{ color: 'var(--muted)' }}>
              <span>Daily usage</span>
              <span className="mono">{used ?? '—'} / {profile.daily_limit}</span>
            </div>
            <div className="h-1.5 rounded-full overflow-hidden" style={{ background: 'var(--border)' }}>
              <div className="h-full rounded-full transition-all duration-500"
                style={{ width: `${pct}%`, background: pct > 90 ? 'var(--danger)' : 'var(--accent)' }} />
            </div>
            <div className="text-xs mt-1.5" style={{ color: 'var(--muted)' }}>
              Resets at UTC midnight
            </div>
          </div>
        </div>

        {/* Stats row */}
        <div className="grid grid-cols-2 gap-3">
          <Stat label="Plan limit" value={profile.daily_limit.toLocaleString()} unit="emails/day" />
          <Stat label="Remaining today" value={(profile.daily_limit - (used ?? 0)).toLocaleString()} unit="emails" />
        </div>
      </div>
    </Layout>
  )
}

function Stat({ label, value, unit }) {
  return (
    <div className="rounded-xl border p-4" style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
      <div className="text-xs mb-2" style={{ color: 'var(--muted)' }}>{label}</div>
      <div className="text-2xl font-semibold mono mb-0.5" style={{ color: 'var(--text)' }}>{value}</div>
      <div className="text-xs" style={{ color: 'var(--muted)' }}>{unit}</div>
    </div>
  )
}

function Skeleton() {
  return (
    <div className="max-w-2xl space-y-3 animate-pulse">
      <div className="h-5 w-32 rounded" style={{ background: 'var(--border)' }} />
      <div className="h-32 rounded-xl" style={{ background: 'var(--surface)' }} />
    </div>
  )
}