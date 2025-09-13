package repository

import (
	"context"
	"time"

	"edutalks/internal/logger"
	"edutalks/internal/models"

	"github.com/jackc/pgx/v5"
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

// Create — сохраняет запись для сброса пароля.
func (r *PasswordResetRepository) Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	log := logger.WithCtx(ctx)

	const q = `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1,$2,$3)
	`
	if _, err := r.db.Exec(ctx, q, userID, tokenHash, expiresAt); err != nil {
		log.Error("password reset repo: create token failed", zap.Error(err), zap.Int64("user_id", userID))
		return err
	}

	log.Info("password reset repo: token created", zap.Int64("user_id", userID), zap.Time("expires_at", expiresAt))
	return nil
}

// GetValidByHash — вернуть валидный (не использованный и не истёкший) токен по хэшу.
func (r *PasswordResetRepository) GetValidByHash(ctx context.Context, tokenHash string) (*models.PasswordResetToken, error) {
	log := logger.WithCtx(ctx)

	const q = `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token_hash = $1
		  AND used_at IS NULL
		  AND expires_at > now()
	`
	var t models.PasswordResetToken
	if err := r.db.QueryRow(ctx, q, tokenHash).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			log.Warn("password reset repo: valid token not found")
		} else {
			log.Error("password reset repo: select valid token failed", zap.Error(err))
		}
		return nil, err
	}

	log.Debug("password reset repo: valid token loaded", zap.Int64("user_id", t.UserID), zap.Time("expires_at", t.ExpiresAt))
	return &t, nil
}

// MarkUsed — пометить токен использованным.
func (r *PasswordResetRepository) MarkUsed(ctx context.Context, id int64) error {
	log := logger.WithCtx(ctx)

	const q = `UPDATE password_reset_tokens SET used_at = now() WHERE id = $1`
	if _, err := r.db.Exec(ctx, q, id); err != nil {
		log.Error("password reset repo: mark used failed", zap.Error(err), zap.Int64("id", id))
		return err
	}

	log.Info("password reset repo: token marked used", zap.Int64("id", id))
	return nil
}

// UpdateUserPassword — обновить пароль пользователя.
func (r *PasswordResetRepository) UpdateUserPassword(ctx context.Context, userID int64, passwordHash string) error {
	log := logger.WithCtx(ctx)

	const q = `UPDATE users SET password_hash = $1 WHERE id = $2`
	if _, err := r.db.Exec(ctx, q, passwordHash, userID); err != nil {
		log.Error("password reset repo: update user password failed", zap.Error(err), zap.Int64("user_id", userID))
		return err
	}

	log.Info("password reset repo: user password updated", zap.Int64("user_id", userID))
	return nil
}

// FindUserIDByEmail — получить ID пользователя по email.
func (r *PasswordResetRepository) FindUserIDByEmail(ctx context.Context, email string) (int64, error) {
	log := logger.WithCtx(ctx)

	const q = `SELECT id FROM users WHERE lower(email)=lower($1) LIMIT 1`

	var userID int64
	if err := r.db.QueryRow(ctx, q, email).Scan(&userID); err != nil {
		if err == pgx.ErrNoRows {
			log.Warn("password reset repo: user not found by email")
		} else {
			log.Error("password reset repo: select user by email failed", zap.Error(err))
		}
		return 0, err
	}

	log.Debug("password reset repo: user found by email", zap.Int64("user_id", userID))
	return userID, nil
}
