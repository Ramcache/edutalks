package repository

import (
	"context"
	"edutalks/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailTokenRepository struct {
	db *pgxpool.Pool
}

func NewEmailTokenRepository(db *pgxpool.Pool) *EmailTokenRepository {
	return &EmailTokenRepository{db: db}
}

func (r *EmailTokenRepository) SaveToken(ctx context.Context, token *models.EmailVerificationToken) error {
	_, err := r.db.Exec(ctx, `INSERT INTO email_verification_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`,
		token.UserID, token.Token, token.ExpiresAt)
	return err
}

func (r *EmailTokenRepository) VerifyToken(ctx context.Context, token string) (*models.EmailVerificationToken, error) {
	row := r.db.QueryRow(ctx, `SELECT user_id, token, expires_at, confirmed FROM email_verification_tokens WHERE token = $1`, token)
	var t models.EmailVerificationToken
	if err := row.Scan(&t.UserID, &t.Token, &t.ExpiresAt, &t.Confirmed); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *EmailTokenRepository) MarkConfirmed(ctx context.Context, token string) error {
	_, err := r.db.Exec(ctx, `UPDATE email_verification_tokens SET confirmed = true WHERE token = $1`, token)
	return err
}
