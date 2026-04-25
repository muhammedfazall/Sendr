const API = import.meta.env.VITE_API_URL

export default function Login() {
  return (
    <div className="min-h-screen flex flex-col items-center justify-center" style={{ background: 'var(--bg)' }}>
      <div className="w-full max-w-sm px-6">

        {/* Logo */}
        <div className="mb-10 text-center">
          <div className="inline-flex items-center gap-2 mb-3">
            <div className="w-7 h-7 rounded-md flex items-center justify-center"
              style={{ background: 'var(--accent)', boxShadow: '0 0 20px rgba(0,208,132,0.3)' }}>
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <path d="M2 7h4M8 4l4 3-4 3" stroke="#000" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </div>
            <span className="text-lg font-semibold tracking-tight" style={{ color: 'var(--text)' }}>Sendr</span>
          </div>
          <p className="text-sm" style={{ color: 'var(--muted)' }}>
            Transactional email API for developers
          </p>
        </div>

        {/* Card */}
        <div className="rounded-xl p-6 border" style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}>
          <h1 className="text-base font-medium mb-1" style={{ color: 'var(--text)' }}>Sign in</h1>
          <p className="text-xs mb-6" style={{ color: 'var(--muted)' }}>
            Continue with your Google account
          </p>

          <a
            href={`${API}/auth/google`}
            className="flex items-center justify-center gap-3 w-full py-2.5 px-4 rounded-lg border text-sm font-medium transition-all duration-150 hover:brightness-125"
            style={{
              background: 'var(--surface)',
              borderColor: 'var(--border-hover)',
              color: 'var(--text)',
            }}
          >
            <svg width="16" height="16" viewBox="0 0 48 48">
              <path fill="#FFC107" d="M43.6 20H24v8h11.3C33.7 33.1 29.3 36 24 36c-6.6 0-12-5.4-12-12s5.4-12 12-12c3 0 5.7 1.1 7.8 2.9l5.7-5.7C33.9 6.7 29.2 4 24 4 12.9 4 4 12.9 4 24s8.9 20 20 20 20-8.9 20-20c0-1.3-.1-2.7-.4-4z" />
              <path fill="#FF3D00" d="M6.3 14.7l6.6 4.8C14.5 15.1 18.9 12 24 12c3 0 5.7 1.1 7.8 2.9l5.7-5.7C33.9 6.7 29.2 4 24 4c-7.6 0-14.2 4.3-17.7 10.7z" />
              <path fill="#4CAF50" d="M24 44c5.2 0 9.8-1.8 13.4-4.7l-6.2-5.2C29.3 35.5 26.8 36 24 36c-5.2 0-9.6-2.9-11.3-7H6.3C9.8 39.5 16.4 44 24 44z" />
              <path fill="#1976D2" d="M43.6 20H24v8h11.3c-.8 2.3-2.3 4.3-4.2 5.8l6.2 5.2C41 35.5 44 30.2 44 24c0-1.3-.1-2.7-.4-4z" />
            </svg>
            Continue with Google
          </a>
        </div>

        <p className="text-center text-xs mt-6" style={{ color: 'var(--muted)' }}>
          By signing in you agree to the terms of service
        </p>
      </div>
    </div>
  )
}