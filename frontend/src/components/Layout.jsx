import { useEffect, useRef } from 'react'
import { NavLink, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../lib/auth-context'

const nav = [
  { to: '/dashboard', label: 'Dashboard', icon: SquaresIcon },
  { to: '/analytics', label: 'Analytics', icon: ChartIcon },
  { to: '/keys', label: 'API Keys', icon: KeyIcon },
  { to: '/send', label: 'Send Email', icon: SendIcon },
  { to: '/history', label: 'Mail History', icon: HistoryIcon },
  { to: '/templates', label: 'Templates', icon: TemplateIcon },
  { to: '/pricing', label: 'Pricing', icon: PricingIcon },
  { to: '/profile', label: 'Profile', icon: UserIcon },
]

export default function Layout({ children }) {
  const { logout } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const mainRef = useRef(null)

  useEffect(() => {
    mainRef.current?.focus()
  }, [location.pathname])

  const handleLogout = () => {
    logout()
    navigate('/')
  }

  return (
    <div className="flex min-h-screen" style={{ background: 'var(--bg)' }}>
      {/* Sidebar */}
      <aside className="w-52 shrink-0 flex flex-col border-r py-6 px-3"
        style={{ borderColor: 'var(--border)', background: 'var(--surface)' }}>

        {/* Logo */}
        <div className="flex items-center gap-2 px-3 mb-8">
          <div className="w-6 h-6 rounded flex items-center justify-center"
            style={{ background: 'var(--accent)' }}>
            <svg width="12" height="12" viewBox="0 0 14 14" fill="none">
              <path d="M2 7h4M8 4l4 3-4 3" stroke="#000" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          </div>
          <span className="text-sm font-semibold" style={{ color: 'var(--text)' }}>Sendr</span>
        </div>

        {/* Nav */}
        <nav className="flex-1 space-y-0.5">
          {nav.map(({ to, label, icon: Icon }) => (
            <NavLink key={to} to={to}
              className={({ isActive }) =>
                `flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm transition-all duration-100 ${
                  isActive ? 'font-medium' : 'font-normal'
                }`
              }
              style={({ isActive }) => ({
                background: isActive ? 'var(--accent-dim)' : 'transparent',
                color: isActive ? 'var(--accent)' : 'var(--muted)',
                border: isActive ? '1px solid var(--accent-border)' : '1px solid transparent',
              })}
            >
              <Icon size={14} />
              {label}
            </NavLink>
          ))}
        </nav>

        {/* Logout */}
        <button onClick={handleLogout}
          className="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm w-full transition-colors duration-100 hover:text-red-400"
          style={{ color: 'var(--muted)' }}>
          <LogoutIcon size={14} />
          Sign out
        </button>
      </aside>

      {/* Main */}
      <main ref={mainRef} tabIndex={-1} className="flex-1 overflow-auto outline-none">
        {children}
      </main>
    </div>
  )
}

function SquaresIcon({ size = 16 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none">
      <rect x="1" y="1" width="6" height="6" rx="1.5" stroke="currentColor" strokeWidth="1.2" />
      <rect x="9" y="1" width="6" height="6" rx="1.5" stroke="currentColor" strokeWidth="1.2" />
      <rect x="1" y="9" width="6" height="6" rx="1.5" stroke="currentColor" strokeWidth="1.2" />
      <rect x="9" y="9" width="6" height="6" rx="1.5" stroke="currentColor" strokeWidth="1.2" />
    </svg>
  )
}

function KeyIcon({ size = 16 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none">
      <circle cx="6" cy="8" r="3.5" stroke="currentColor" strokeWidth="1.2" />
      <path d="M9 8h6M13 8v2" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
    </svg>
  )
}

function SendIcon({ size = 16 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none">
      <path d="M14 2L7 9M14 2L9 14l-2-5-5-2 12-5z" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function LogoutIcon({ size = 16 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none">
      <path d="M6 2H3a1 1 0 00-1 1v10a1 1 0 001 1h3M11 11l3-3-3-3M14 8H6" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function PricingIcon({ size = 16 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none">
      <path d="M8 1v14M4.5 4h5.25a2.25 2.25 0 010 4.5H4.5M4.5 8.5h6a2.25 2.25 0 010 4.5H4.5" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function UserIcon({ size = 16 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none">
      <circle cx="8" cy="5" r="3" stroke="currentColor" strokeWidth="1.2" />
      <path d="M2.5 14c.7-2.6 2.7-4 5.5-4s4.8 1.4 5.5 4" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
    </svg>
  )
}

function ChartIcon({ size = 16 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none">
      <path d="M2 14V3M6 14V7M10 14V5M14 14v-4" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function TemplateIcon({ size = 16 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none">
      <path d="M3 2h10a1 1 0 011 1v10a1 1 0 01-1 1H3a1 1 0 01-1-1V3a1 1 0 011-1zM5 6h6M5 9h4" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function HistoryIcon({ size = 16 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none">
      <path d="M8 14A6 6 0 118 2a6 6 0 010 12z" stroke="currentColor" strokeWidth="1.2" />
      <path d="M8 5v3l2 2" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}
