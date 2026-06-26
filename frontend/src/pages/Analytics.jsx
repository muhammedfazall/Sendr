import { useEffect, useState } from 'react'
import { api } from '../lib/api'
import Layout from '../components/Layout'

const METRICS = [
  { key: 'sent', label: 'Sent', color: '#00d084' },
  { key: 'delivered', label: 'Delivered', color: '#3b82f6' },
  { key: 'opens', label: 'Opens', color: '#a78bfa' },
  { key: 'clicks', label: 'Clicks', color: '#f59e0b' },
  { key: 'bounces', label: 'Bounces', color: '#ef4444' },
  { key: 'spam', label: 'Spam complaints', color: '#ec4899' },
]

export default function Analytics() {
  const [stats, setStats] = useState(null)
  const [error, setError] = useState(null)

  useEffect(() => {
    api.getStats()
      .then(setStats)
      .catch((err) => setError(err.message))
  }, [])

  const sent = stats?.sent ?? 1
  const rows = METRICS.map((m) => ({
    ...m,
    value: stats?.[m.key] ?? 0,
    pct: Math.round(((stats?.[m.key] ?? 0) / sent) * 100),
  }))

  return (
    <Layout>
      <div className="p-8 max-w-5xl">
        <h1 className="text-lg font-semibold mb-1" style={{ color: 'var(--text)' }}>Analytics</h1>
        <p className="text-sm mb-8" style={{ color: 'var(--muted)' }}>Delivery performance and engagement metrics.</p>

        {error && (
          <div role="alert" className="text-xs mb-6 px-3 py-2 rounded-lg" style={{ background: 'var(--danger-dim)', color: 'var(--danger)', border: '1px solid rgba(255,77,77,0.2)' }}>
            {error}
          </div>
        )}

        {!stats && !error && (
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="h-24 rounded-xl animate-pulse" style={{ background: 'var(--surface)' }} />
            ))}
          </div>
        )}

        {stats && (
          <>
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3 mb-8">
              {rows.map((m) => (
                <div key={m.key} className="rounded-xl border p-4" style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
                  <div className="text-xs mb-2" style={{ color: 'var(--muted)' }}>{m.label}</div>
                  <div className="text-2xl font-semibold mono" style={{ color: m.color }}>{m.value.toLocaleString()}</div>
                  <div className="text-xs mt-1" style={{ color: 'var(--muted)' }}>{m.pct}% of sent</div>
                </div>
              ))}
            </div>

            <div className="rounded-xl border p-5" style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
              <h2 className="text-sm font-semibold mb-5" style={{ color: 'var(--text)' }}>Engagement funnel</h2>
              <div className="space-y-4">
                {rows.map((m) => {
                  const maxVal = Math.max(...rows.map((r) => r.value), 1)
                  const barWidth = Math.max((m.value / maxVal) * 100, m.value > 0 ? 2 : 0)
                  return (
                    <div key={m.key} className="flex items-center gap-4">
                      <span className="text-xs w-28 shrink-0 text-right" style={{ color: 'var(--muted)' }}>{m.label}</span>
                      <div className="flex-1 h-5 rounded-md overflow-hidden" style={{ background: 'var(--bg)' }}>
                        <div
                          className="h-full rounded-md transition-all duration-700"
                          style={{ width: `${barWidth}%`, background: m.color, opacity: m.value === 0 ? 0.3 : 0.85 }}
                        />
                      </div>
                      <span className="text-xs mono w-20 shrink-0" style={{ color: 'var(--text)' }}>{m.value.toLocaleString()}</span>
                    </div>
                  )
                })}
              </div>
            </div>

            <div className="mt-6 rounded-xl border p-5" style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
              <h2 className="text-sm font-semibold mb-4" style={{ color: 'var(--text)' }}>Delivery health</h2>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <HealthCard label="Delivered rate" value={stats.sent > 0 ? `${Math.round((stats.delivered / stats.sent) * 100)}%` : '—'} tone={stats.delivered / stats.sent >= 0.95 ? 'good' : stats.delivered / stats.sent >= 0.8 ? 'warn' : 'bad'} />
                <HealthCard label="Open rate" value={stats.delivered > 0 ? `${Math.round((stats.opens / stats.delivered) * 100)}%` : '—'} tone={stats.opens / stats.delivered >= 0.3 ? 'good' : 'warn'} />
                <HealthCard label="Click rate" value={stats.opens > 0 ? `${Math.round((stats.clicks / stats.opens) * 100)}%` : '—'} tone={stats.clicks / stats.opens >= 0.1 ? 'good' : 'warn'} />
                <HealthCard label="Bounce rate" value={stats.sent > 0 ? `${Math.round((stats.bounces / stats.sent) * 100)}%` : '—'} tone={stats.bounces / stats.sent <= 0.02 ? 'good' : stats.bounces / stats.sent <= 0.05 ? 'warn' : 'bad'} />
              </div>
            </div>
          </>
        )}
      </div>
    </Layout>
  )
}

function HealthCard({ label, value, tone }) {
  const toneColor = tone === 'good' ? 'var(--accent)' : tone === 'bad' ? 'var(--danger)' : '#f59e0b'
  return (
    <div className="rounded-lg border p-4" style={{ background: 'var(--bg)', borderColor: 'var(--border)' }}>
      <div className="text-xs mb-2" style={{ color: 'var(--muted)' }}>{label}</div>
      <div className="text-xl font-semibold mono" style={{ color: toneColor }}>{value}</div>
    </div>
  )
}
