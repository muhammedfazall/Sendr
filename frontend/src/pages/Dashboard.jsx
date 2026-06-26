import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../lib/api'
import Layout from '../components/Layout'

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

function formatLimit(value, unit = '') {
  if (value < 0) return 'Unlimited'
  const formatted = Number(value ?? 0).toLocaleString()
  return unit ? `${formatted} ${unit}` : formatted
}

function loadDashboardPrefs() {
  try {
    return JSON.parse(localStorage.getItem('sendr:profile-preferences')) || {}
  } catch {
    return {}
  }
}

export default function Dashboard() {
  const [profile, setProfile] = useState(null)
  const [keys, setKeys] = useState([])
  const [history, setHistory] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [notice, setNotice] = useState(null)

  useEffect(() => {
    async function loadDashboard() {
      const [profileResult, keysResult, historyResult] = await Promise.allSettled([
        api.me(),
        api.listKeys(),
        api.listEmails('', 8, 0),
      ])

      if (profileResult.status === 'fulfilled') {
        setProfile(profileResult.value)
      } else {
        setError(profileResult.reason.message)
      }

      if (keysResult.status === 'fulfilled') {
        setKeys(keysResult.value ?? [])
      } else {
        setNotice('API key status could not be loaded.')
      }

      if (historyResult.status === 'fulfilled') {
        setHistory(historyResult.value ?? [])
      } else {
        setNotice('Recent activity could not be loaded.')
      }

      setLoading(false)
    }

    Promise.resolve().then(loadDashboard)
  }, [])

  const stats = useMemo(() => {
    if (!profile) return null

    const usageToday = profile.usage_today ?? 0
    const dailyLimit = profile.daily_limit ?? 0
    const remaining = profile.remaining ?? dailyLimit
    const unlimitedUsage = dailyLimit < 0
    const usagePct = unlimitedUsage || dailyLimit === 0 ? 0 : Math.min((usageToday / dailyLimit) * 100, 100)

    const sent = history.filter((item) => item.status === 'sent').length
    const failed = history.filter((item) => item.status === 'failed').length
    const active = history.filter((item) => item.status === 'pending' || item.status === 'processing').length
    const completed = sent + failed
    const deliveryScore = completed === 0 ? null : Math.round((sent / completed) * 100)

    const maxKeys = profile.max_api_keys ?? 0
    const keysRemaining = maxKeys < 0 ? -1 : Math.max(maxKeys - keys.length, 0)
    const keyPct = maxKeys <= 0 ? 0 : Math.min((keys.length / maxKeys) * 100, 100)

    return {
      usageToday,
      dailyLimit,
      remaining,
      unlimitedUsage,
      usagePct,
      sent,
      failed,
      active,
      deliveryScore,
      maxKeys,
      keysRemaining,
      keyPct,
    }
  }, [history, keys.length, profile])

  if (error) {
    return (
      <Layout>
        <div className="p-8 text-sm" role="alert" style={{ color: 'var(--danger)' }}>{error}</div>
      </Layout>
    )
  }

  if (loading || !profile || !stats) {
    return (
      <Layout>
        <div className="p-8">
          <Skeleton />
        </div>
      </Layout>
    )
  }

  const compact = Boolean(loadDashboardPrefs().compactDashboard)
  const firstName = profile.name?.split(' ')[0] || 'there'
  const planColor = { free: '#666', pro: '#00d084', max: '#a78bfa' }[profile.plan] || '#666'
  const usageRing = `conic-gradient(${stats.usagePct > 90 ? 'var(--danger)' : 'var(--accent)'} ${stats.usagePct}%, var(--border) 0)`

  return (
    <Layout>
      <div className="p-8 max-w-6xl">
        <div className="flex flex-col lg:flex-row lg:items-end lg:justify-between gap-4 mb-7">
          <div>
            <h1 className="text-lg font-semibold mb-1" style={{ color: 'var(--text)' }}>Dashboard</h1>
            <p className="text-sm" style={{ color: 'var(--muted)' }}>Welcome back, {firstName}. Your sending workspace is ready.</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Link to="/send" className="px-3 py-2 rounded-lg text-sm font-medium" style={{ background: 'var(--accent)', color: '#000' }}>
              Send email
            </Link>
            <Link to="/profile" className="px-3 py-2 rounded-lg text-sm font-medium" style={{ background: 'var(--surface)', color: 'var(--text)', border: '1px solid var(--border)' }}>
              Profile
            </Link>
          </div>
        </div>

        {notice && (
          <div role="status" className="text-xs mb-4 px-3 py-2 rounded-lg" style={{ background: 'var(--accent-dim)', color: 'var(--accent)', border: '1px solid var(--accent-border)' }}>
            {notice}
          </div>
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-3 mb-4">
          <Metric label="Usage today" value={formatLimit(stats.usageToday)} hint={stats.unlimitedUsage ? 'Unlimited plan' : `${formatLimit(stats.remaining)} left`} />
          <Metric label="Delivery health" value={stats.deliveryScore === null ? 'Ready' : `${stats.deliveryScore}%`} hint={`${stats.sent} sent, ${stats.failed} failed`} tone={stats.failed > 0 ? 'warn' : 'good'} />
          <Metric label="API keys" value={formatLimit(keys.length)} hint={stats.keysRemaining < 0 ? 'Unlimited keys' : `${formatLimit(stats.keysRemaining)} available`} />
          <Metric label="Plan" value={profile.plan} hint={formatLimit(stats.dailyLimit, 'emails/day')} color={planColor} />
        </div>

        <div className={`grid grid-cols-1 ${compact ? 'xl:grid-cols-2' : 'xl:grid-cols-[1.15fr_0.85fr]'} gap-4`}>
          <section className="rounded-xl border p-5" style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
            <div className="flex flex-col md:flex-row md:items-center gap-5">
              <div className="w-28 h-28 rounded-full p-2 shrink-0" style={{ background: usageRing }}>
                <div className="w-full h-full rounded-full flex flex-col items-center justify-center" style={{ background: 'var(--surface)' }}>
                  <span className="text-2xl font-semibold mono" style={{ color: 'var(--text)' }}>{stats.unlimitedUsage ? 'All' : `${Math.round(stats.usagePct)}%`}</span>
                  <span className="text-xs" style={{ color: 'var(--muted)' }}>used</span>
                </div>
              </div>

              <div className="flex-1 min-w-0">
                <div className="flex items-center justify-between gap-3 mb-3">
                  <h2 className="text-sm font-semibold" style={{ color: 'var(--text)' }}>Sending capacity</h2>
                  <span className="text-xs px-2 py-0.5 rounded-full font-medium mono" style={{ background: `${planColor}18`, color: planColor, border: `1px solid ${planColor}30` }}>
                    {profile.plan}
                  </span>
                </div>
                <div className="flex justify-between text-xs mb-2" style={{ color: 'var(--muted)' }}>
                  <span>Daily usage</span>
                  <span className="mono">{formatLimit(stats.usageToday)} / {formatLimit(stats.dailyLimit)}</span>
                </div>
                <div className="h-2 rounded-full overflow-hidden" style={{ background: 'var(--border)' }}>
                  <div className="h-full rounded-full transition-all duration-500" style={{ width: `${stats.unlimitedUsage ? 100 : stats.usagePct}%`, background: stats.usagePct > 90 ? 'var(--danger)' : 'var(--accent)' }} />
                </div>
                <div className="grid grid-cols-2 gap-3 mt-5">
                  <MiniStat label="Wait time" value={`${profile.rate_wait_secs}s`} />
                  <MiniStat label="Remaining" value={formatLimit(stats.remaining)} />
                </div>
              </div>
            </div>
          </section>

          <section className="rounded-xl border p-5" style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
            <div className="flex items-center justify-between gap-3 mb-4">
              <h2 className="text-sm font-semibold" style={{ color: 'var(--text)' }}>Launchpad</h2>
              <Link to="/keys" className="text-xs" style={{ color: 'var(--accent)' }}>Manage keys</Link>
            </div>
            <div className="space-y-3">
              <Action to="/keys" title={keys.length > 0 ? 'API key active' : 'Create first API key'} value={keys.length > 0 ? `${keys.length} configured` : 'Start here'} />
              <Action to="/send" title="Send test email" value={stats.active > 0 ? `${stats.active} in progress` : 'Queue a message'} />
              <Action to="/history" title="Review activity" value={history.length > 0 ? `${history.length} recent jobs` : 'No jobs yet'} />
            </div>
          </section>
        </div>

        <div className="grid grid-cols-1 xl:grid-cols-[0.85fr_1.15fr] gap-4 mt-4">
          <section className="rounded-xl border p-5" style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
            <h2 className="text-sm font-semibold mb-4" style={{ color: 'var(--text)' }}>API key capacity</h2>
            <div className="flex items-center justify-between text-xs mb-2" style={{ color: 'var(--muted)' }}>
              <span>Keys in use</span>
              <span className="mono">{keys.length} / {formatLimit(stats.maxKeys)}</span>
            </div>
            <div className="h-2 rounded-full overflow-hidden mb-4" style={{ background: 'var(--border)' }}>
              <div className="h-full rounded-full" style={{ width: `${stats.maxKeys < 0 ? 35 : stats.keyPct}%`, background: 'var(--accent)' }} />
            </div>
            <div className="space-y-2">
              {keys.slice(0, 3).map((key) => (
                <div key={key.id} className="flex items-center justify-between gap-3 text-xs rounded-lg px-3 py-2" style={{ background: 'var(--bg)', border: '1px solid var(--border)' }}>
                  <span style={{ color: 'var(--text)' }}>{key.name}</span>
                  <span className="mono" style={{ color: 'var(--muted)' }}>mk_live_{key.prefix}...</span>
                </div>
              ))}
              {keys.length === 0 && <div className="text-xs rounded-lg px-3 py-3" style={{ color: 'var(--muted)', background: 'var(--bg)', border: '1px solid var(--border)' }}>No API keys yet.</div>}
            </div>
          </section>

          <section className="rounded-xl border overflow-hidden" style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
            <div className="flex items-center justify-between gap-3 px-5 py-4" style={{ borderBottom: '1px solid var(--border)' }}>
              <h2 className="text-sm font-semibold" style={{ color: 'var(--text)' }}>Recent activity</h2>
              <Link to="/history" className="text-xs" style={{ color: 'var(--accent)' }}>View all</Link>
            </div>
            {history.length === 0 ? (
              <div className="p-8 text-center text-sm" style={{ color: 'var(--muted)' }}>No email activity yet.</div>
            ) : (
              <div>
                {history.map((item, index) => {
                  const color = STATUS_COLOR[item.status] || STATUS_COLOR.pending
                  return (
                    <div key={item.id} className="grid grid-cols-[1fr_auto] md:grid-cols-[1fr_auto_auto] gap-3 items-center px-5 py-3" style={{ borderBottom: index < history.length - 1 ? '1px solid var(--border)' : 'none' }}>
                      <div className="min-w-0">
                        <div className="text-xs mono truncate" style={{ color: 'var(--text)' }}>{item.id}</div>
                        <div className="text-xs mt-0.5" style={{ color: 'var(--muted)' }}>{new Date(item.created_at).toLocaleString()}</div>
                      </div>
                      <span className="px-2 py-0.5 rounded-full text-xs" style={{ background: color.bg, color: color.color, border: `1px solid ${color.border}` }}>
                        {STATUS_LABEL[item.status] ?? item.status}
                      </span>
                      <span className="hidden md:inline text-xs mono" style={{ color: 'var(--muted)' }}>{item.retries} retries</span>
                    </div>
                  )
                })}
              </div>
            )}
          </section>
        </div>
      </div>
    </Layout>
  )
}

function Metric({ label, value, hint, tone, color }) {
  const toneColor = color || (tone === 'warn' ? '#f59e0b' : tone === 'good' ? 'var(--accent)' : 'var(--text)')

  return (
    <div className="rounded-xl border p-4" style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
      <div className="text-xs mb-2" style={{ color: 'var(--muted)' }}>{label}</div>
      <div className="text-2xl font-semibold mono mb-1 capitalize" style={{ color: toneColor }}>{value}</div>
      <div className="text-xs" style={{ color: 'var(--muted)' }}>{hint}</div>
    </div>
  )
}

function MiniStat({ label, value }) {
  return (
    <div className="rounded-lg px-3 py-3" style={{ background: 'var(--bg)', border: '1px solid var(--border)' }}>
      <div className="text-xs mb-1" style={{ color: 'var(--muted)' }}>{label}</div>
      <div className="text-sm font-semibold mono" style={{ color: 'var(--text)' }}>{value}</div>
    </div>
  )
}

function Action({ to, title, value }) {
  return (
    <Link to={to} className="flex items-center justify-between gap-3 rounded-lg px-3 py-3 transition-colors" style={{ background: 'var(--bg)', border: '1px solid var(--border)' }}>
      <span className="text-sm" style={{ color: 'var(--text)' }}>{title}</span>
      <span className="text-xs" style={{ color: 'var(--muted)' }}>{value}</span>
    </Link>
  )
}

function Skeleton() {
  return (
    <div className="max-w-6xl space-y-4 animate-pulse" role="status" aria-label="Loading dashboard">
      <div className="h-5 w-32 rounded" style={{ background: 'var(--border)' }} />
      <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
        {[0, 1, 2, 3].map((item) => (
          <div key={item} className="h-28 rounded-xl" style={{ background: 'var(--surface)' }} />
        ))}
      </div>
      <div className="h-64 rounded-xl" style={{ background: 'var(--surface)' }} />
    </div>
  )
}
