# Sendr
A email service enabling developers to authenticate, generate API keys and send emails



Salman Faris
18:52
// Package constant holds status codes and canonical messages used across the
// backend. Keeping these in one place means handlers/services/tests never
// re-invent HTTP strings or reach for literal error phrases.
package constants

import "net/http"

// HTTP status aliases — handlers read from constants.StatusXxx instead of
// pulling in net/http all over the place, so a future swap (e.g. 409 →
// custom app code) only touches this file.
const (
	StatusOK                  = http.StatusOK
	StatusCreated             = http.StatusCreated
	StatusBadRequest          = http.StatusBadRequest
	StatusUnauthorized        = http.StatusUnauthorized
	StatusForbidden           = http.StatusForbidden
	StatusNotFound            = http.StatusNotFound
	StatusConflict            = http.StatusConflict
	StatusInternalServerError = http.StatusInternalServerError
	StatusServiceUnavailable  = http.StatusServiceUnavailable
)

// Canonical messages. Handlers should prefer these over ad-hoc strings so
// the API surface stays consistent.
const (
	MsgOK                 = "OK"
	MsgCreated            = "Created"
	MsgInvalidBody        = "Invalid request body"
	MsgMissingFields      = "Required fields are missing"
	MsgUnauthorized       = "Authentication required"
	MsgInvalidCredentials = "Invalid credentials"
	MsgForbidden          = "You do not have permission to access this resource"
	MsgNotFound           = "Resource not found"
	MsgConflict           = "Resource already exists"
	MsgDBError            = "Database error"
	MsgInternal           = "Internal server error"
	MsgOrgInactive        = "Organisation is inactive — contact your administrator"
	MsgSubscriptionEnded  = "Subscription has ended — please renew to regain access"
	MsgTenantUnavailable  = "Tenant database is unavailable"
	MsgTokenIssueFailed   = "Failed to issue token"
	MsgSuperAdminOnly     = "Super admin access only"
)
evw-pxeu-yfe