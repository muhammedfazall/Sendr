import { useCallback, useEffect, useState } from 'react'
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

export default function MailHistory() {
  const [history, setHistory] = useState([])
  const [historyLoading, setHistoryLoading] = useState(false)
  const [statusFilter, setStatusFilter] = useState('')
  const [offset, setOffset] = useState(0)
  const [hasMoreHistory, setHasMoreHistory] = useState(false)

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
    Promise.resolve().then(() => fetchHistory('', 0, false))
  }, [fetchHistory])

  const handleFilterChange = (filter) => {
    setStatusFilter(filter)
    fetchHistory(filter, 0, false)
  }

  const loadMoreHistory = () => {
    fetchHistory(statusFilter, offset + HISTORY_LIMIT, true)
  }

  return (
    <Layout>
      <div className="p-8 max-w-2xl">
        <h1 className="text-lg font-semibold mb-1" style={{ color: 'var(--text)' }}>Mail History</h1>
        <p className="text-sm mb-8" style={{ color: 'var(--muted)' }}>View your email sending history and status.</p>

        <div>
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