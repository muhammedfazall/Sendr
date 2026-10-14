package templatehandler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/response"
)

type Handler struct {
	repo ports.TemplateRepository
}

func New(repo ports.TemplateRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Create(c *gin.Context) {
	userID := userFromClaims(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}

	var req struct {
		Name            string `json:"name"`
		SubjectTemplate string `json:"subject_template"`
		HTMLTemplate    string `json:"html_template"`
		TextTemplate    string `json:"text_template"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_body", "invalid JSON")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		response.Error(c, http.StatusBadRequest, "missing_fields", "name is required")
		return
	}
	tpl := &domain.Template{
		ID:              uuid.NewString(),
		UserID:          userID,
		Name:            req.Name,
		SubjectTemplate: req.SubjectTemplate,
		HTMLTemplate:    req.HTMLTemplate,
		TextTemplate:    req.TextTemplate,
	}
	if err := h.repo.Create(c.Request.Context(), tpl); err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", "failed to create template")
		return
	}
	response.JSON(c, http.StatusCreated, tpl)
}

func (h *Handler) List(c *gin.Context) {
	userID := userFromClaims(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	tpls, err := h.repo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", "failed to list templates")
		return
	}
	response.JSON(c, http.StatusOK, tpls)
}

func (h *Handler) Get(c *gin.Context) {
	userID := userFromClaims(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	id := c.Param("id")
	tpl, err := h.repo.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "not_found", "template not found")
		return
	}
	response.JSON(c, http.StatusOK, tpl)
}

func (h *Handler) Update(c *gin.Context) {
	userID := userFromClaims(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	id := c.Param("id")
	existing, err := h.repo.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "not_found", "template not found")
		return
	}
	var req struct {
		Name            *string `json:"name"`
		SubjectTemplate *string `json:"subject_template"`
		HTMLTemplate    *string `json:"html_template"`
		TextTemplate    *string `json:"text_template"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_body", "invalid JSON")
		return
	}
	if req.Name != nil {
		existing.Name = strings.TrimSpace(*req.Name)
	}
	if req.SubjectTemplate != nil {
		existing.SubjectTemplate = *req.SubjectTemplate
	}
	if req.HTMLTemplate != nil {
		existing.HTMLTemplate = *req.HTMLTemplate
	}
	if req.TextTemplate != nil {
		existing.TextTemplate = *req.TextTemplate
	}
	if err := h.repo.Update(c.Request.Context(), existing); err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", "failed to update template")
		return
	}
	response.JSON(c, http.StatusOK, existing)
}

func (h *Handler) Delete(c *gin.Context) {
	userID := userFromClaims(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	id := c.Param("id")
	if err := h.repo.Delete(c.Request.Context(), id, userID); err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", "failed to delete template")
		return
	}
	c.Status(http.StatusNoContent)
}

func userFromClaims(c *gin.Context) string {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		return ""
	}
	userID, _ := claims["user_id"].(string)
	return userID
}
