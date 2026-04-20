package constants

import "time"

const (
  JWTExpiry         = 24 * time.Hour
  OAuthStateCookieTTL = 5 * time.Minute  // MaxAge on the oauth_state cookie
  OAuthStateCookieName = "oauth_state"
)