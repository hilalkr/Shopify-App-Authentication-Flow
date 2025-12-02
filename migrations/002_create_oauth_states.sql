CREATE TABLE IF NOT EXISTS oauth_states (
  id BIGSERIAL PRIMARY KEY,
  shop_domain TEXT NOT NULL,
  nonce TEXT NOT NULL UNIQUE, 
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oauth_states_shop_domain ON oauth_states (shop_domain);