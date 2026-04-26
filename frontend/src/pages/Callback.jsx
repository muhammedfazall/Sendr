import { useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../lib/auth'

export default function Callback() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const handled = useRef(false)

  useEffect(() => {
    if (handled.current) return
    handled.current = true

    const params = new URLSearchParams(window.location.search)
    const token = params.get('token')
    const error = params.get('error')

    console.log('CALLBACK: token=', token, 'error=', error)

    if (token) {
      login(token)
      navigate('/dashboard', { replace: true })
    } else {
      navigate(`/?error=${error || 'auth_failed'}`, { replace: true })
    }
  }, [])

  return (
    <div className="min-h-screen flex items-center justify-center" style={{ background: 'var(--bg)' }}>
      <div className="flex items-center gap-3">
        <div className="w-4 h-4 rounded-full border-2 animate-spin"
          style={{ borderColor: 'var(--accent)', borderTopColor: 'transparent' }} />
        <span className="text-sm" style={{ color: 'var(--muted)' }}>Signing you in…</span>
      </div>
    </div>
  )
}