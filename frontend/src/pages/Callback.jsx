import { useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../lib/auth-context'
import { getApiBaseUrl } from '../lib/config'

const API = getApiBaseUrl()

export default function Callback() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const handled = useRef(false)

  // Exchange the HttpOnly auth_token cookie for the JWT
  // useEffect(() => {
  //   if (handled.current) return
  //   handled.current = true

  //   // Check for OAuth error in the redirect
  //   const errorParam = new URLSearchParams(window.location.search).get('error')
  //   if (errorParam) {
  //     navigate(`/?error=${errorParam}`, { replace: true })
  //     return
  //   }

  //   const controller = new AbortController()

  //   fetch(`${API}/auth/token`, {
  //     credentials: 'include',
  //     signal: controller.signal,
  //   })
  //     .then(r => r.json())
  //     .then(data => {
  //       if (data.token) {
  //         login(data.token)
  //         navigate('/dashboard', { replace: true })
  //       } else {
  //         navigate('/?error=auth_failed', { replace: true })
  //       }
  //     })
  //     .catch((err) => {
  //       if (err.name !== 'AbortError') navigate('/?error=auth_failed', { replace: true })
  //     })

  //   return () => controller.abort()
  // }, [login, navigate])

  useEffect(() => {
  if (handled.current) return
  handled.current = true

  const params = new URLSearchParams(window.location.search)

  const error = params.get('error')
  if (error) {
    navigate(`/?error=${error}`, { replace: true })
    return
  }

  const token = params.get('token')

  if (!token) {
    navigate('/?error=auth_failed', { replace: true })
    return
  }

  login(token)
  navigate('/dashboard', { replace: true })
}, [login, navigate])


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
