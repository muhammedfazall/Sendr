-- 1. Add new columns to plans table for the new limits
ALTER TABLE plans
  ADD COLUMN max_api_keys    INT NOT NULL DEFAULT 1,
  ADD COLUMN rate_wait_secs  INT NOT NULL DEFAULT 30,
  ADD COLUMN price_paise     INT NOT NULL DEFAULT 0;

-- 2. Update existing plan rows with new values
UPDATE plans SET daily_limit = 5,   max_api_keys = 1,  rate_wait_secs = 30, price_paise = 0     WHERE name = 'free';
UPDATE plans SET daily_limit = 10,  max_api_keys = 3,  rate_wait_secs = 5,  price_paise = 29900 WHERE name = 'pro';
UPDATE plans SET daily_limit = -1,  max_api_keys = -1, rate_wait_secs = 0,  price_paise = 99900 WHERE name = 'max';
-- NOTE: daily_limit = -1 means unlimited, max_api_keys = -1 means unlimited

-- 3. Create payments table to track all Razorpay transactions
CREATE TABLE payments (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  razorpay_order_id   TEXT UNIQUE NOT NULL,
  razorpay_payment_id TEXT,
  razorpay_signature  TEXT,
  plan_name       TEXT NOT NULL,
  amount_paise    INT NOT NULL,
  currency        TEXT NOT NULL DEFAULT 'INR',
  status          TEXT NOT NULL DEFAULT 'created',  -- created | paid | failed
  created_at      TIMESTAMPTZ DEFAULT now(),
  updated_at      TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_payments_user_id ON payments(user_id);
CREATE INDEX idx_payments_order_id ON payments(razorpay_order_id);