package services

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"github.com/muhammedfazall/Sendr/internal/core/ports"
)

// TemplateService renders user-defined email templates.
type TemplateService struct {
	repo ports.TemplateRepository
}

func NewTemplateService(repo ports.TemplateRepository) *TemplateService {
	return &TemplateService{repo: repo}
}

// Render loads a template and renders it with the given data, returning
// the subject, HTML body, and text body.
func (s *TemplateService) Render(ctx context.Context, templateID, userID string, data map[string]any) (subject, html, text string, err error) {
	tpl, err := s.repo.GetByID(ctx, templateID, userID)
	if err != nil {
		return "", "", "", fmt.Errorf("load template: %w", err)
	}

	if tpl.SubjectTemplate != "" {
		subject, err = executeTemplate("subject", tpl.SubjectTemplate, data)
		if err != nil {
			return "", "", "", fmt.Errorf("render subject: %w", err)
		}
	}
	if tpl.HTMLTemplate != "" {
		html, err = executeTemplate("html", tpl.HTMLTemplate, data)
		if err != nil {
			return "", "", "", fmt.Errorf("render html: %w", err)
		}
	}
	if tpl.TextTemplate != "" {
		text, err = executeTemplate("text", tpl.TextTemplate, data)
		if err != nil {
			return "", "", "", fmt.Errorf("render text: %w", err)
		}
	}
	return subject, html, text, nil
}

func executeTemplate(name, src string, data map[string]any) (string, error) {
	t, err := template.New(name).Parse(src)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// compile-time check
var _ ports.TemplateService = (*TemplateService)(nil)
