package unsubhandler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
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
<html>
<head><meta charset="UTF-8"><title>Unsubscribe</title>
</head>
<body>
{{if .Success}}<h1>You have been unsubscribed</h1><p>You will no longer receive emails from us.</p>
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

// Serve handles GET (show confirmation) and POST (apply opt-out) for /unsubscribe.
func (h *Handler) Serve(c *gin.Context) {
	email := c.Request.FormValue("email")
	token := c.Request.FormValue("token")

	if email == "" || token == "" {
		_ = confirmationPage.Execute(c.Writer, pageData{Invalid: true})
		return
	}

	if !validToken(email, token, h.secret) {
		_ = confirmationPage.Execute(c.Writer, pageData{Invalid: true})
		return
	}

	if c.Request.Method == http.MethodPost {
		already, _ := h.repo.IsUnsubscribed(c.Request.Context(), email)
		if already {
			_ = confirmationPage.Execute(c.Writer, pageData{Already: true})
			return
		}
		if err := h.repo.Unsubscribe(c.Request.Context(), email); err != nil {
			_ = confirmationPage.Execute(c.Writer, pageData{Invalid: true})
			return
		}
		_ = confirmationPage.Execute(c.Writer, pageData{Success: true})
		return
	}

	_ = confirmationPage.Execute(c.Writer, pageData{Email: email, Token: token})
}

func validToken(email, token, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(email))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(token))
}
