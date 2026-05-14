package paymentrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/pkg/constants"
)

type PostgresPaymentRepository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{db: db}
}

func (r *PostgresPaymentRepository) Create(ctx context.Context, p *domain.Payment) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO payments (user_id, razorpay_order_id, plan_name, amount_paise, currency, status)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		p.UserID, p.RazorpayOrderID, p.PlanName, p.AmountPaise, p.Currency, p.Status,
	)
	if err != nil {
		return fmt.Errorf("create payment: %w", err)
	}
	return nil
}

func (r *PostgresPaymentRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	var p domain.Payment
	var paymentID sql.NullString
	var signature sql.NullString

	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, razorpay_order_id, razorpay_payment_id, razorpay_signature,
		        plan_name, amount_paise, currency, status, created_at, updated_at
		 FROM payments WHERE razorpay_order_id = $1`, orderID,
	).Scan(&p.ID, &p.UserID, &p.RazorpayOrderID, &paymentID, &signature,
		&p.PlanName, &p.AmountPaise, &p.Currency, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, constants.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("find payment by order: %w", err)
	}

	p.RazorpayPaymentID = stringPtrFromNull(paymentID)
	p.RazorpaySignature = stringPtrFromNull(signature)

	return &p, nil
}

func (r *PostgresPaymentRepository) MarkPaid(ctx context.Context, orderID, paymentID, signature string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE payments
		 SET razorpay_payment_id = $1, razorpay_signature = $2,
		     status = 'paid', updated_at = now()
		 WHERE razorpay_order_id = $3`,
		paymentID, signature, orderID,
	)
	if err != nil {
		return fmt.Errorf("mark payment paid: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return constants.ErrPaymentNotFound
	}
	return nil
}

func (r *PostgresPaymentRepository) MarkFailed(ctx context.Context, orderID string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE payments SET status = 'failed', updated_at = now()
		 WHERE razorpay_order_id = $1`, orderID,
	)
	if err != nil {
		return fmt.Errorf("mark payment failed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return constants.ErrPaymentNotFound
	}
	return nil
}

func stringPtrFromNull(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	s := value.String
	return &s
}
