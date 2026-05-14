DROP TABLE IF EXISTS payments;
ALTER TABLE plans
  DROP COLUMN IF EXISTS max_api_keys,
  DROP COLUMN IF EXISTS rate_wait_secs,
  DROP COLUMN IF EXISTS price_paise;