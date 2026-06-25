package templatehandler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
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

func (h *Handler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := userFromClaims(r)
		if userID == "" {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
			return
		}
		var req struct {
			Name           string `json:"name"`
			SubjectTemplate string `json:"subject_template"`
			HTMLTemplate   string `json:"html_template"`
			TextTemplate   string `json:"text_template"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid_body", "invalid JSON")
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			response.Error(w, http.StatusBadRequest, "missing_fields", "name is required")
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
		if err := h.repo.Create(r.Context(), tpl); err != nil {
			response.Error(w, http.StatusInternalServerError, "internal_error", "failed to create template")
			return
		}
		response.JSON(w, http.StatusCreated, tpl)
	}
}

func (h *Handler) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := userFromClaims(r)
		if userID == "" {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
			return
		}
		tpls, err := h.repo.ListByUser(r.Context(), userID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "internal_error", "failed to list templates")
			return
		}
		response.JSON(w, http.StatusOK, tpls)
	}
}

func (h *Handler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := userFromClaims(r)
		if userID == "" {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
			return
		}
		id := chi.URLParam(r, "id")
		tpl, err := h.repo.GetByID(r.Context(), id, userID)
		if err != nil {
			response.Error(w, http.StatusNotFound, "not_found", "template not found")
			return
		}
		response.JSON(w, http.StatusOK, tpl)
	}
}

func (h *Handler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := userFromClaims(r)
		if userID == "" {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
			return
		}
		id := chi.URLParam(r, "id")
		existing, err := h.repo.GetByID(r.Context(), id, userID)
		if err != nil {
			response.Error(w, http.StatusNotFound, "not_found", "template not found")
			return
		}
		var req struct {
			Name           *string `json:"name"`
			SubjectTemplate *string `json:"subject_template"`
			HTMLTemplate   *string `json:"html_template"`
			TextTemplate   *string `json:"text_template"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid_body", "invalid JSON")
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
		if err := h.repo.Update(r.Context(), existing); err != nil {
			response.Error(w, http.StatusInternalServerError, "internal_error", "failed to update template")
			return
		}
		response.JSON(w, http.StatusOK, existing)
	}
}

func (h *Handler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := userFromClaims(r)
		if userID == "" {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
			return
		}
		id := chi.URLParam(r, "id")
		if err := h.repo.Delete(r.Context(), id, userID); err != nil {
			response.Error(w, http.StatusInternalServerError, "internal_error", "failed to delete template")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func userFromClaims(r *http.Request) string {
	claims, ok := middleware.GetClaims(r)
	if !ok {
		return ""
	}
	userID, _ := claims["user_id"].(string)
	return userID
}
