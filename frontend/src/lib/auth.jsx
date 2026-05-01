import { useEffect, useState } from 'react'
import { api, SessionExpiredError } from './api'
import { AuthContext } from './auth-context'

export function AuthProvider({ children }) {
  const [token, setToken] = useState(() => sessionStorage.getItem('token'))
  const [loading] = useState(false)
  const [sessionExpired, setSessionExpired] = useState(false)

  useEffect(() => {
    const handleExpired = () => {
      sessionStorage.removeItem('token')
      setToken(null)
      setSessionExpired(true)
    }

    window.addEventListener('sendr:session-expired', handleExpired)

    return () => window.removeEventListener('sendr:session-expired', handleExpired)
  }, [])

  const login = (t) => {
    sessionStorage.setItem('token', t)
    setToken(t)
    setSessionExpired(false)
  }

  const logout = async () => {
    try {
      await api.logout() // DELETE refresh token from Redis
    } catch (err) {
      if (!(err instanceof SessionExpiredError) && import.meta.env.DEV) {
        console.error('Logout failed', err)
      }
      // Even if the API call fails, clear local state
    }
    sessionStorage.removeItem('token')
    setToken(null)
    setSessionExpired(false)
  }

  return (
    <AuthContext.Provider
      value={{
        token,
        login,
        logout,
        loading,
        sessionExpired,
        clearSessionExpired: () => setSessionExpired(false),
        isAuthed: !!token,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}
