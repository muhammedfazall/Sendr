import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import '../landing.css'

/* ─── Inline SVG Icons ─── */
function SendrLogo({ size = 28 }) {
  return (
    <div className="sendr-logo" style={{ width: size, height: size }}>
      <svg width={size * 0.5} height={size * 0.5} viewBox="0 0 14 14" fill="none">
        <path d="M2 7h4M8 4l4 3-4 3" stroke="#000" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    </div>
  )
}

function ArrowRight({ size = 16 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none">
      <path d="M3 8h10M9 4l4 4-4 4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function CheckIcon({ size = 16 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none">
      <path d="M3 8.5l3 3 7-7" stroke="var(--accent)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

/* ─── Animated Counter ─── */
function AnimatedNumber({ target, suffix = '', duration = 2000 }) {
  const [count, setCount] = useState(0)
  const ref = useRef(null)
  const animated = useRef(false)

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting && !animated.current) {
          animated.current = true
          const start = performance.now()
          const step = (now) => {
            const progress = Math.min((now - start) / duration, 1)
            const eased = 1 - Math.pow(1 - progress, 3)
            setCount(Math.floor(eased * target))
            if (progress < 1) requestAnimationFrame(step)
          }
          requestAnimationFrame(step)
        }
      },
      { threshold: 0.3 }
    )
    if (ref.current) observer.observe(ref.current)
    return () => observer.disconnect()
  }, [target, duration])

  return <span ref={ref}>{count.toLocaleString()}{suffix}</span>
}

/* ─── Scroll-reveal wrapper ─── */
function Reveal({ children, delay = 0, className = '' }) {
  const ref = useRef(null)
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => { if (entry.isIntersecting) setVisible(true) },
      { threshold: 0.15 }
    )
    if (ref.current) observer.observe(ref.current)
    return () => observer.disconnect()
  }, [])

  return (
    <div
      ref={ref}
      className={`reveal ${visible ? 'reveal--visible' : ''} ${className}`}
      style={{ transitionDelay: `${delay}ms` }}
    >
      {children}
    </div>
  )
}

/* ─── Code Snippet with typing effect ─── */
function CodeBlock() {
  return (
    <div className="code-window">
      <div className="code-window__bar">
        <span className="code-dot code-dot--red" />
        <span className="code-dot code-dot--yellow" />
        <span className="code-dot code-dot--green" />
        <span className="code-window__title mono">send-email.sh</span>
      </div>
      <pre className="code-window__body mono">
        <code>
          <span className="code-muted">{'# Send an email with a single API call\n'}</span>
          <span className="code-keyword">curl</span>{' -X POST \\\n'}
          {'  https://api.sendr.app/emails/send \\\n'}
          {'  -H '}<span className="code-string">"Authorization: Bearer mk_live_..."</span>{' \\\n'}
          {'  -H '}<span className="code-string">"Content-Type: application/json"</span>{' \\\n'}
          {'  -d '}<span className="code-string">{'\'{\n'}</span>
          <span className="code-string">{'    "to": "user@example.com",\n'}</span>
          <span className="code-string">{'    "subject": "Welcome aboard!",\n'}</span>
          <span className="code-string">{'    "body": "<h1>Hello from Sendr</h1>"\n'}</span>
          <span className="code-string">{"  }'"}</span>
        </code>
      </pre>
      <div className="code-window__response">
        <span className="code-muted">{'// Response: '}</span>
        <span className="code-accent">{'202 Accepted'}</span>
        {' — '}
        <span className="code-string">{'"job_id": "a1b2c3d4..."'}</span>
      </div>
    </div>
  )
}

/* ─── Feature Card ─── */
function FeatureCard({ icon, title, desc, delay }) {
  return (
    <Reveal delay={delay}>
      <div className="feature-card">
        <div className="feature-card__icon">{icon}</div>
        <h3 className="feature-card__title">{title}</h3>
        <p className="feature-card__desc">{desc}</p>
      </div>
    </Reveal>
  )
}

/* ─── Pricing Card ─── */
function PricingCard({ name, price, period, features, highlighted, delay }) {
  return (
    <Reveal delay={delay}>
      <div className={`pricing-card ${highlighted ? 'pricing-card--highlighted' : ''}`}>
        {highlighted && <div className="pricing-card__badge">Most Popular</div>}
        <h3 className="pricing-card__name">{name}</h3>
        <div className="pricing-card__price">
          <span className="pricing-card__amount">{price}</span>
          {period && <span className="pricing-card__period">/{period}</span>}
        </div>
        <ul className="pricing-card__features">
          {features.map((f, i) => (
            <li key={i}><CheckIcon size={14} /> {f}</li>
          ))}
        </ul>
        <Link to="/login" className={`pricing-card__cta ${highlighted ? 'pricing-card__cta--primary' : ''}`}>
          Get Started <ArrowRight size={14} />
        </Link>
      </div>
    </Reveal>
  )
}

/* ─── Step Card ─── */
function StepCard({ number, title, desc, delay }) {
  return (
    <Reveal delay={delay}>
      <div className="step-card">
        <div className="step-card__number">{number}</div>
        <h3 className="step-card__title">{title}</h3>
        <p className="step-card__desc">{desc}</p>
      </div>
    </Reveal>
  )
}

/* ═══════════════════════════════════════════
   LANDING PAGE
   ═══════════════════════════════════════════ */
export default function Landing() {
  const [scrolled, setScrolled] = useState(false)

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 40)
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  return (
    <div className="landing">
      {/* ── Floating Gradient Orbs ── */}
      <div className="orb orb--1" />
      <div className="orb orb--2" />
      <div className="orb orb--3" />

      {/* ═══ NAVBAR ═══ */}
      <nav className={`landing-nav ${scrolled ? 'landing-nav--scrolled' : ''}`}>
        <div className="landing-nav__inner">
          <Link to="/" className="landing-nav__brand">
            <SendrLogo size={28} />
            <span className="landing-nav__name">Sendr</span>
          </Link>
          <div className="landing-nav__links">
            <a href="#features" className="landing-nav__link">Features</a>
            <a href="#how-it-works" className="landing-nav__link">How it Works</a>
            <a href="#pricing" className="landing-nav__link">Pricing</a>
          </div>
          <div className="landing-nav__actions">
            <Link to="/login" className="landing-nav__signin">Sign in</Link>
            <Link to="/login" className="landing-nav__cta">
              Get Started <ArrowRight size={14} />
            </Link>
          </div>
        </div>
      </nav>

      {/* ═══ HERO ═══ */}
      <section className="hero">
        <div className="hero__content">
          <Reveal>
            <div className="hero__badge">
              <span className="hero__badge-dot" />
              Built for developers, by developers
            </div>
          </Reveal>
          <Reveal delay={100}>
            <h1 className="hero__title">
              Transactional emails,{' '}
              <span className="hero__title-accent">delivered reliably.</span>
            </h1>
          </Reveal>
          <Reveal delay={200}>
            <p className="hero__subtitle">
              A blazing-fast email API with queued delivery, automatic retries,
              and real-time status tracking. Ship emails from your app in minutes, not days.
            </p>
          </Reveal>
          <Reveal delay={300}>
            <div className="hero__actions">
              <Link to="/login" className="btn btn--primary btn--lg">
                Start Sending Free <ArrowRight size={16} />
              </Link>
              <a href="#features" className="btn btn--ghost btn--lg">
                Explore Features
              </a>
            </div>
          </Reveal>
          <Reveal delay={400}>
            <div className="hero__stats">
              <div className="hero__stat">
                <span className="hero__stat-value">Unlimited</span>
                <span className="hero__stat-label">Emails on Max</span>
              </div>
              <div className="hero__stat-divider" />
              <div className="hero__stat">
                <span className="hero__stat-value"><AnimatedNumber target={99} suffix="%" duration={1500} /></span>
                <span className="hero__stat-label">Delivery Rate</span>
              </div>
              <div className="hero__stat-divider" />
              <div className="hero__stat">
                <span className="hero__stat-value">&lt;2s</span>
                <span className="hero__stat-label">Queue Latency</span>
              </div>
            </div>
          </Reveal>
        </div>
        <Reveal delay={350} className="hero__code-wrap">
          <CodeBlock />
        </Reveal>
      </section>

      {/* ═══ LOGOS / TRUST ═══ */}
      <section className="trust">
        <p className="trust__label">Built with modern, battle-tested infrastructure</p>
        <div className="trust__logos">
          {['Go', 'PostgreSQL', 'Redis', 'SendGrid', 'Docker'].map((name) => (
            <div key={name} className="trust__logo mono">{name}</div>
          ))}
        </div>
      </section>

      {/* ═══ FEATURES ═══ */}
      <section id="features" className="features">
        <Reveal>
          <div className="section-header">
            <span className="section-tag">Features</span>
            <h2 className="section-title">Everything you need to send emails at scale</h2>
            <p className="section-subtitle">
              From API key management to dead letter queues — Sendr handles the complexity so you can focus on building.
            </p>
          </div>
        </Reveal>

        <div className="features__grid">
          <FeatureCard
            delay={0}
            icon={
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" stroke="var(--accent)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            }
            title="Queued Delivery"
            desc="Emails are queued and processed asynchronously. Get a 202 response instantly while we handle delivery in the background."
          />
          <FeatureCard
            delay={80}
            icon={
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                <path d="M1 4v6h6M23 20v-6h-6" stroke="var(--accent)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
                <path d="M20.49 9A9 9 0 005.64 5.64L1 10m22 4l-4.64 4.36A9 9 0 013.51 15" stroke="var(--accent)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            }
            title="Automatic Retries"
            desc="Failed deliveries are retried up to 3 times with exponential backoff. No emails lost to transient failures."
          />
          <FeatureCard
            delay={160}
            icon={
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                <rect x="3" y="11" width="18" height="11" rx="2" stroke="var(--accent)" strokeWidth="1.5" />
                <path d="M7 11V7a5 5 0 0110 0v4" stroke="var(--accent)" strokeWidth="1.5" strokeLinecap="round" />
              </svg>
            }
            title="Secure API Keys"
            desc="SHA-256 hashed keys with revocable access. Manage multiple keys per project with fine-grained control."
          />
          <FeatureCard
            delay={240}
            icon={
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                <circle cx="12" cy="12" r="10" stroke="var(--accent)" strokeWidth="1.5" />
                <path d="M12 6v6l4 2" stroke="var(--accent)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            }
            title="Real-time Status"
            desc="Track every email from pending to delivered. Poll job status with simple GET requests for full visibility."
          />
          <FeatureCard
            delay={320}
            icon={
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                <path d="M12 2L2 7l10 5 10-5-10-5z" stroke="var(--accent)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
                <path d="M2 17l10 5 10-5M2 12l10 5 10-5" stroke="var(--accent)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            }
            title="Rate Limiting"
            desc="Built-in Redis-powered rate limiting per plan. Clear headers tell your app exactly when limits reset."
          />
          <FeatureCard
            delay={400}
            icon={
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8l-6-6z" stroke="var(--accent)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
                <path d="M14 2v6h6M16 13H8M16 17H8M10 9H8" stroke="var(--accent)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            }
            title="Dead Letter Queue"
            desc="Permanently failed emails are captured in a DLQ for review. Never lose track of delivery issues."
          />
        </div>
      </section>

      {/* ═══ HOW IT WORKS ═══ */}
      <section id="how-it-works" className="how-it-works">
        <Reveal>
          <div className="section-header">
            <span className="section-tag">How it Works</span>
            <h2 className="section-title">Up and running in three steps</h2>
            <p className="section-subtitle">
              No complex configuration. No vendor lock-in. Just a clean REST API.
            </p>
          </div>
        </Reveal>

        <div className="steps__grid">
          <StepCard
            delay={0}
            number="01"
            title="Create an Account"
            desc="Sign in with Google OAuth. Your account and free plan are provisioned instantly — no credit card required."
          />
          <StepCard
            delay={120}
            number="02"
            title="Generate an API Key"
            desc="Create a named API key from your dashboard. Use the key in your Authorization header for all email requests."
          />
          <StepCard
            delay={240}
            number="03"
            title="Send Emails via API"
            desc="POST your email payload. We queue, process, retry, and deliver — you get a job ID to track the status."
          />
        </div>
      </section>

      {/* ═══ PRICING ═══ */}
      <section id="pricing" className="pricing">
        <Reveal>
          <div className="section-header">
            <span className="section-tag">Pricing</span>
            <h2 className="section-title">Simple, transparent pricing</h2>
            <p className="section-subtitle">
              Start free, scale as you grow. No hidden fees, no surprises.
            </p>
          </div>
        </Reveal>

        <div className="pricing__grid">
          <PricingCard
            delay={0}
            name="Free"
            price="₹0"
            period="forever"
            features={[
              '5 emails / day',
              '1 API key',
              '30s wait between sends',
              'Job status tracking',
            ]}
          />
          <PricingCard
            delay={120}
            name="Pro"
            price="₹299"
            period="month"
            highlighted
            features={[
              '10 emails / day',
              '3 API keys',
              '5s wait between sends',
              'Job status tracking',
              'Email support',
            ]}
          />
          <PricingCard
            delay={240}
            name="Max"
            price="₹999"
            period="month"
            features={[
              'Unlimited emails / day',
              'Unlimited API keys',
              'No wait between sends',
              'Priority support',
              'Dedicated throughput',
            ]}
          />
        </div>
      </section>

      {/* ═══ CTA ═══ */}
      <section className="cta">
        <div className="cta__glow" />
        <Reveal>
          <h2 className="cta__title">Ready to start sending?</h2>
          <p className="cta__desc">
            Join developers who trust Sendr for reliable, scalable email delivery. Free forever — no credit card needed.
          </p>
          <Link to="/login" className="btn btn--primary btn--lg cta__btn">
            Create Your Free Account <ArrowRight size={16} />
          </Link>
        </Reveal>
      </section>

      {/* ═══ FOOTER ═══ */}
      <footer className="landing-footer">
        <div className="landing-footer__inner">
          <div className="landing-footer__brand">
            <div className="landing-footer__logo">
              <SendrLogo size={24} />
              <span>Sendr</span>
            </div>
            <p className="landing-footer__tagline">
              Transactional email API for developers.
            </p>
          </div>

          <div className="landing-footer__links">
            <div className="landing-footer__col">
              <h4>Product</h4>
              <a href="#features">Features</a>
              <a href="#pricing">Pricing</a>
              <a href="#how-it-works">How it Works</a>
            </div>
            <div className="landing-footer__col">
              <h4>Developers</h4>
              <a href="https://github.com/muhammedfazall/Sendr" target="_blank" rel="noopener noreferrer">GitHub</a>
              <a href="#features">API Reference</a>
              <a href="#features">Status</a>
            </div>
            <div className="landing-footer__col">
              <h4>Company</h4>
              <a href="#features">About</a>
              <a href="#features">Privacy</a>
              <a href="#features">Terms</a>
            </div>
          </div>
        </div>
        <div className="landing-footer__bottom">
          <span>© {new Date().getFullYear()} Sendr. All rights reserved.</span>
        </div>
      </footer>
    </div>
  )
}
