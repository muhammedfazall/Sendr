-- Add monthly_limit column to plans
ALTER TABLE plans
  ADD COLUMN monthly_limit INT NOT NULL DEFAULT 0;

-- Update plans with new Free/Pro/Max limits
UPDATE plans SET daily_limit = 100,  monthly_limit = 3000,  max_api_keys = 1,  rate_wait_secs = 0,  price_paise = 0     WHERE name = 'free';
UPDATE plans SET daily_limit = -1,   monthly_limit = 50000, max_api_keys = 5,  rate_wait_secs = 0,  price_paise = 29900 WHERE name = 'pro';
UPDATE plans SET daily_limit = -1,   monthly_limit = 100000,max_api_keys = -1, rate_wait_secs = 0,  price_paise = 99900 WHERE name = 'max';
-- Note: -1 means unlimited
