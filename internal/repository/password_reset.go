package repository

import (
	"context"
	"edutalks/internal/logger"
	"edutalks/internal/models"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type PasswordResetRepository struct {
	db *pgxpool.Pool
}

func NewPasswordResetRepository(db *pgxpool.Pool) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

type PasswordResetRepo interface {
	Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error
	GetValidByHash(ctx context.Context, tokenHash string) (*models.PasswordResetToken, error)
	MarkUsed(ctx context.Context, id int64) error
	UpdateUserPassword(ctx context.Context, userID int64, passwordHash string) error
	FindUserIDByEmail(ctx context.Context, email string) (int64, error)
}

func (r *PasswordResetRepository) Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at) VALUES ($1,$2,$3)`,
		userID, tokenHash, expiresAt,
	)
	if err != nil {
		logger.Log.Error("Create reset token failed", zap.Error(err), zap.Int64("user_id", userID))
	}
	return err
}

func (r *PasswordResetRepository) GetValidByHash(ctx context.Context, tokenHash string) (*models.PasswordResetToken, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token_hash = $1
		  AND used_at IS NULL
		  AND expires_at > now()
	`, tokenHash)

	var t models.PasswordResetToken
	if err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *PasswordResetRepository) MarkUsed(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `UPDATE password_reset_tokens SET used_at = now() WHERE id = $1`, id)
	return err
}

func (r *PasswordResetRepository) UpdateUserPassword(ctx context.Context, userID int64, passwordHash string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, passwordHash, userID)
	return err
}

func (r *PasswordResetRepository) FindUserIDByEmail(ctx context.Context, email string) (int64, error) {
	var userID int64
	err := r.db.QueryRow(ctx, `SELECT id FROM users WHERE lower(email)=lower($1) LIMIT 1`, email).Scan(&userID)
	return userID, err
}
