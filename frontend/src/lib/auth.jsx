import { createContext, useContext, useState } from 'react'
import { api } from './api'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [token, setToken] = useState(() => sessionStorage.getItem('token'))

  const login = (t) => {
    sessionStorage.setItem('token', t)
    setToken(t)
  }

  const logout = async () => {
    try {
      await api.logout() // DELETE refresh token from Redis
    } catch {
      // Even if the API call fails, clear local state
    }
    sessionStorage.removeItem('token')
    setToken(null)
  }

  return (
    <AuthContext.Provider value={{ token, login, logout, isAuthed: !!token }}>
      {children}
    </AuthContext.Provider>
  )
}

export const useAuth = () => useContext(AuthContext)