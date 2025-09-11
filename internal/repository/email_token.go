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
	_, err := r.db.Exec(ctx, `DELETE FROM email_verification_tokens WHERE user_id = $1`, token.UserID)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO email_verification_tokens (user_id, token, expires_at, confirmed, created_at)
		VALUES ($1, $2, $3, false, NOW())
	`, token.UserID, token.Token, token.ExpiresAt)

	return err
}

func (r *EmailTokenRepository) VerifyToken(ctx context.Context, token string) (*models.EmailVerificationToken, error) {
	row := r.db.QueryRow(ctx, `SELECT user_id, token, expires_at, confirmed, created_at FROM email_verification_tokens WHERE token = $1`, token)
	var t models.EmailVerificationToken
	if err := row.Scan(&t.UserID, &t.Token, &t.ExpiresAt, &t.Confirmed, &t.CreatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *EmailTokenRepository) MarkConfirmed(ctx context.Context, token string) error {
	_, err := r.db.Exec(ctx, `UPDATE email_verification_tokens SET confirmed = true WHERE token = $1`, token)
	return err
}

func (r *EmailTokenRepository) GetLastTokenByUserID(ctx context.Context, userID int) (*models.EmailVerificationToken, error) {
	row := r.db.QueryRow(ctx, `
		SELECT user_id, token, expires_at, confirmed, created_at
		FROM email_verification_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, userID)

	var t models.EmailVerificationToken
	if err := row.Scan(&t.UserID, &t.Token, &t.ExpiresAt, &t.Confirmed, &t.CreatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}
