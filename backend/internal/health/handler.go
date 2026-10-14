package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/muhammedfazall/Sendr/pkg/response"
)

// Handler handles HTTP requests for health checks.
type Handler struct {
	service *Service
}

// NewHandler creates a new health Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Check handles GET /health — returns the health status of all dependencies.
func (h *Handler) Check(c *gin.Context) {
	result := h.service.Check(c.Request.Context())

	status := http.StatusOK
	if result.DB != "ok" || result.Redis != "ok" {
		status = http.StatusServiceUnavailable
	}

	response.JSON(c, status, result)
}
