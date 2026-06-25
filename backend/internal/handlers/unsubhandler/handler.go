package unsubhandler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"html/template"
	"net/http"
)

// repo defines the unsubscribe persistence the handler needs.
type repo interface {
	Unsubscribe(ctx context.Context, email string) error
	IsUnsubscribed(ctx context.Context, email string) (bool, error)
}

// Handler serves the unsubscribe confirmation page and processes opt-outs.
type Handler struct {
	repo   repo
	secret string
}

func New(repo repo, secret string) *Handler {
	return &Handler{repo: repo, secret: secret}
}

var confirmationPage = template.Must(template.New("unsubscribe").Parse(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><title>Unsubscribe</title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; max-width: 500px; margin: 80px auto; padding: 0 20px; text-align: center; color: #333; }
h1 { font-size: 1.5rem; }
.btn { display: inline-block; padding: 12px 28px; margin: 20px 0; background: #e53e3e; color: #fff; text-decoration: none; border-radius: 6px; font-size: 1rem; }
.btn:hover { background: #c53030; }
.success { color: #38a169; }
.error { color: #e53e3e; }
</style></head>
<body>
{{if .Success}}<h1 class="success">You have been unsubscribed</h1><p>You will no longer receive emails from us.</p>
{{else if .Already}}<h1>Already unsubscribed</h1><p>This email address is already opted out.</p>
{{else if .Invalid}}<h1 class="error">Invalid link</h1><p>This unsubscribe link is invalid or expired.</p>
{{else}}<h1>Unsubscribe</h1><p>Are you sure you want to stop receiving emails?</p>
<form method="post" action="/unsubscribe">
<input type="hidden" name="email" value="{{.Email}}">
<input type="hidden" name="token" value="{{.Token}}">
<button class="btn" type="submit">Unsubscribe</button>
</form>{{end}}
</body>
</html>`))

type pageData struct {
	Email   string
	Token   string
	Success bool
	Already bool
	Invalid bool
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	token := r.FormValue("token")

	if email == "" || token == "" {
		_ = confirmationPage.Execute(w, pageData{Invalid: true})
		return
	}

	if !validToken(email, token, h.secret) {
		_ = confirmationPage.Execute(w, pageData{Invalid: true})
		return
	}

	if r.Method == http.MethodPost {
		already, _ := h.repo.IsUnsubscribed(r.Context(), email)
		if already {
			_ = confirmationPage.Execute(w, pageData{Already: true})
			return
		}
		if err := h.repo.Unsubscribe(r.Context(), email); err != nil {
			_ = confirmationPage.Execute(w, pageData{Invalid: true})
			return
		}
		_ = confirmationPage.Execute(w, pageData{Success: true})
		return
	}

	_ = confirmationPage.Execute(w, pageData{Email: email, Token: token})
}

func validToken(email, token, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(email))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(token))
}
