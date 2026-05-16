import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './lib/auth'
import { useAuth } from './lib/auth-context'
import Login from './pages/Login'
import Callback from './pages/Callback'
import Dashboard from './pages/Dashboard'
import APIKeys from './pages/APIKeys'
import SendEmail from './pages/SendEmail'
import MailHistory from './pages/MailHistory'
import Pricing from './pages/Pricing'
import Profile from './pages/Profile'

function ProtectedRoute({ children }) {
  const { isAuthed, loading, sessionExpired, clearSessionExpired } = useAuth()
  if (loading) return <Splash />
  if (sessionExpired) return <SessionExpired onDismiss={clearSessionExpired} />
  return isAuthed ? children : <Navigate to="/" replace />
}

function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<Login />} />
      <Route path="/callback" element={<Callback />} />
      <Route path="/dashboard" element={<ProtectedRoute><Dashboard /></ProtectedRoute>} />
      <Route path="/keys" element={<ProtectedRoute><APIKeys /></ProtectedRoute>} />
      <Route path="/send" element={<ProtectedRoute><SendEmail /></ProtectedRoute>} />
      <Route path="/history" element={<ProtectedRoute><MailHistory /></ProtectedRoute>} />
      <Route path="/profile" element={<ProtectedRoute><Profile /></ProtectedRoute>} />
      <Route path="*" element={<Navigate to="/" replace />} />
      <Route path="/pricing" element={<ProtectedRoute><Pricing /></ProtectedRoute>} />
    </Routes>
  )
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <AppRoutes />
      </BrowserRouter>
    </AuthProvider>
  )
}

function Splash() {
  return (
    <div className="min-h-screen flex items-center justify-center" style={{ background: 'var(--bg)' }}>
      <div className="text-sm" style={{ color: 'var(--muted)' }}>Loading...</div>
    </div>
  )
}

function SessionExpired({ onDismiss }) {
  return (
    <div className="min-h-screen flex items-center justify-center" style={{ background: 'var(--bg)' }}>
      <div className="rounded-xl border p-6 w-full max-w-sm" style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
        <h1 className="text-base font-medium mb-2" style={{ color: 'var(--text)' }}>Session expired</h1>
        <p className="text-sm mb-5" style={{ color: 'var(--muted)' }}>Sign in again to continue.</p>
        <button
          type="button"
          onClick={onDismiss}
          className="w-full py-2.5 rounded-lg text-sm font-medium"
          style={{ background: 'var(--accent)', color: '#000' }}
        >
          Back to sign in
        </button>
      </div>
    </div>
  )
}
