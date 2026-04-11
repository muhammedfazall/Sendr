package auth

// GoogleUser represents the user info returned by Google's OAuth API.
type GoogleUser struct {
	ID    string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// AuthTokens holds the JWT token(s) returned after successful authentication.
type AuthTokens struct {
	Token string `json:"token"`
}
