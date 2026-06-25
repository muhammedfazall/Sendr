package templaterepo

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
)

type PostgresTemplateRepository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *PostgresTemplateRepository {
	return &PostgresTemplateRepository{db: db}
}

func (r *PostgresTemplateRepository) Create(ctx context.Context, tpl *domain.Template) error {
	const q = `INSERT INTO templates (id, user_id, name, subject_template, html_template, text_template, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	now := time.Now()
	tpl.CreatedAt = now
	tpl.UpdatedAt = now
	_, err := r.db.Exec(ctx, q, tpl.ID, tpl.UserID, tpl.Name, tpl.SubjectTemplate, tpl.HTMLTemplate, tpl.TextTemplate, tpl.CreatedAt, tpl.UpdatedAt)
	return err
}

func (r *PostgresTemplateRepository) GetByID(ctx context.Context, id, userID string) (*domain.Template, error) {
	const q = `SELECT id, user_id, name, subject_template, html_template, text_template, created_at, updated_at FROM templates WHERE id = $1 AND user_id = $2`
	var t domain.Template
	err := r.db.QueryRow(ctx, q, id, userID).Scan(&t.ID, &t.UserID, &t.Name, &t.SubjectTemplate, &t.HTMLTemplate, &t.TextTemplate, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}
	return &t, nil
}

func (r *PostgresTemplateRepository) ListByUser(ctx context.Context, userID string) ([]domain.Template, error) {
	const q = `SELECT id, user_id, name, subject_template, html_template, text_template, created_at, updated_at FROM templates WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()
	var out []domain.Template
	for rows.Next() {
		var t domain.Template
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.SubjectTemplate, &t.HTMLTemplate, &t.TextTemplate, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *PostgresTemplateRepository) Update(ctx context.Context, tpl *domain.Template) error {
	const q = `UPDATE templates SET name=$1, subject_template=$2, html_template=$3, text_template=$4, updated_at=$5 WHERE id=$6 AND user_id=$7`
	tpl.UpdatedAt = time.Now()
	_, err := r.db.Exec(ctx, q, tpl.Name, tpl.SubjectTemplate, tpl.HTMLTemplate, tpl.TextTemplate, tpl.UpdatedAt, tpl.ID, tpl.UserID)
	return err
}

func (r *PostgresTemplateRepository) Delete(ctx context.Context, id, userID string) error {
	const q = `DELETE FROM templates WHERE id = $1 AND user_id = $2`
	_, err := r.db.Exec(ctx, q, id, userID)
	return err
}
