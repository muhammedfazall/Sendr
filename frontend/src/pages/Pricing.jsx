import { useEffect, useState } from 'react'
import { api } from '../lib/api'
import Layout from '../components/Layout'

const PLAN_FEATURES = {
  free: {
    color: '#666',
    icon: '📧',
    features: ['5 emails/day', '30 sec wait between sends', '1 API key'],
  },
  pro: {
    color: '#00d084',
    icon: '⚡',
    features: ['10 emails/day', '5 sec wait between sends', '3 API keys'],
  },
  max: {
    color: '#a78bfa',
    icon: '🚀',
    features: ['Unlimited emails', 'No wait limit', 'Unlimited API keys'],
  },
}

export default function Pricing() {
  const [plans, setPlans] = useState([])
  const [profile, setProfile] = useState(null)
  const [loading, setLoading] = useState(true)
  const [paying, setPaying] = useState(null)  // plan name being paid for
  const [error, setError] = useState(null)
  const [success, setSuccess] = useState(null)

  useEffect(() => {
    Promise.all([api.listPlans(), api.me()])
      .then(([p, me]) => {
        setPlans(p ?? [])
        setProfile(me)
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  // Load Razorpay script once
  useEffect(() => {
    if (document.getElementById('razorpay-script')) return
    const script = document.createElement('script')
    script.id = 'razorpay-script'
    script.src = 'https://checkout.razorpay.com/v1/checkout.js'
    script.async = true
    document.body.appendChild(script)
  }, [])

  async function handleUpgrade(planName) {
    setError(null)
    setSuccess(null)
    setPaying(planName)

    try {
      // 1. Create order on backend
      const order = await api.createPaymentOrder(planName)

      // 2. Open Razorpay Checkout
      const options = {
        key: order.key_id,
        amount: order.amount,
        currency: order.currency,
        name: 'Sendr',
        description: `Upgrade to ${planName} plan`,
        order_id: order.order_id,
        handler: async function (response) {
          // 3. Verify payment on backend
          try {
            await api.verifyPayment({
              razorpay_order_id: response.razorpay_order_id,
              razorpay_payment_id: response.razorpay_payment_id,
              razorpay_signature: response.razorpay_signature,
            })
            setSuccess(`Successfully upgraded to ${planName}!`)
            // Refresh profile to show new plan
            const me = await api.me()
            setProfile(me)
          } catch (err) {
            setError(`Payment verification failed: ${err.message}`)
          } finally {
            setPaying(null)
          }
        },
        prefill: {
          name: profile?.name || '',
          email: profile?.email || '',
        },
        theme: {
          color: '#a78bfa',
        },
        modal: {
          ondismiss: () => setPaying(null),
        },
      }

      const rzp = new window.Razorpay(options)
      rzp.on('payment.failed', (response) => {
        setError(response.error.description || 'Payment failed')
        setPaying(null)
      })
      rzp.open()
    } catch (err) {
      setError(err.message)
      setPaying(null)
    }
  }

  if (loading) {
    return (
      <Layout>
        <div className="p-8">
          <div className="max-w-4xl animate-pulse space-y-3">
            <div className="h-5 w-32 rounded" style={{ background: 'var(--border)' }} />
            <div className="h-64 rounded-xl" style={{ background: 'var(--surface)' }} />
          </div>
        </div>
      </Layout>
    )
  }

  return (
    <Layout>
      <div className="p-8 max-w-4xl">
        <h1 className="text-lg font-semibold mb-1" style={{ color: 'var(--text)' }}>
          Plans & Pricing
        </h1>
        <p className="text-sm mb-8" style={{ color: 'var(--muted)' }}>
          Choose the plan that fits your needs.
        </p>

        {error && (
          <div
            role="alert"
            className="text-xs mb-4 px-3 py-2 rounded-lg"
            style={{
              background: 'var(--danger-dim)',
              color: 'var(--danger)',
              border: '1px solid rgba(255,77,77,0.2)',
            }}
          >
            {error}
          </div>
        )}

        {success && (
          <div
            role="status"
            className="text-xs mb-4 px-3 py-2 rounded-lg"
            style={{
              background: 'var(--accent-dim)',
              color: 'var(--accent)',
              border: '1px solid var(--accent-border)',
            }}
          >
            {success}
          </div>
        )}

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {plans.map((plan) => {
            const meta = PLAN_FEATURES[plan.name] || PLAN_FEATURES.free
            const isCurrent = profile?.plan === plan.name
            const isFree = plan.price_paise === 0

            return (
              <div
                key={plan.id}
                className="rounded-xl border p-5 flex flex-col"
                style={{
                  background: 'var(--surface)',
                  borderColor: isCurrent ? meta.color : 'var(--border)',
                  borderWidth: isCurrent ? '2px' : '1px',
                }}
              >
                <div className="flex items-center gap-2 mb-3">
                  <span className="text-xl">{meta.icon}</span>
                  <span
                    className="text-sm font-semibold uppercase"
                    style={{ color: meta.color }}
                  >
                    {plan.name}
                  </span>
                  {isCurrent && (
                    <span
                      className="text-xs px-2 py-0.5 rounded-full"
                      style={{
                        background: `${meta.color}18`,
                        color: meta.color,
                        border: `1px solid ${meta.color}30`,
                      }}
                    >
                      Current
                    </span>
                  )}
                </div>

                <div className="text-2xl font-bold mb-1 mono" style={{ color: 'var(--text)' }}>
                  {isFree ? 'Free' : `₹${(plan.price_paise / 100).toLocaleString()}`}
                  {!isFree && (
                    <span className="text-xs font-normal" style={{ color: 'var(--muted)' }}>
                      /month
                    </span>
                  )}
                </div>

                <ul className="flex-1 mt-3 mb-5 space-y-2">
                  {meta.features.map((f, i) => (
                    <li key={i} className="text-xs flex items-center gap-2" style={{ color: 'var(--muted)' }}>
                      <span style={{ color: meta.color }}>✓</span> {f}
                    </li>
                  ))}
                </ul>

                {!isCurrent && !isFree && (
                  <button
                    type="button"
                    onClick={() => handleUpgrade(plan.name)}
                    disabled={paying !== null}
                    className="w-full py-2.5 rounded-lg text-sm font-medium transition-opacity disabled:opacity-40"
                    style={{ background: meta.color, color: '#000' }}
                  >
                    {paying === plan.name ? 'Processing...' : `Upgrade to ${plan.name}`}
                  </button>
                )}
                {isCurrent && (
                  <div
                    className="w-full py-2.5 rounded-lg text-sm font-medium text-center"
                    style={{ background: 'var(--border)', color: 'var(--muted)' }}
                  >
                    Current plan
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>
    </Layout>
  )
}