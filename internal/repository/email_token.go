package repository

import (
	"context"

	"edutalks/internal/logger"
	"edutalks/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type EmailTokenRepository struct {
	db *pgxpool.Pool
}

func NewEmailTokenRepository(db *pgxpool.Pool) *EmailTokenRepository {
	return &EmailTokenRepository{db: db}
}

// SaveToken — сохраняет (или заменяет) токен подтверждения email для пользователя.
func (r *EmailTokenRepository) SaveToken(ctx context.Context, token *models.EmailVerificationToken) error {
	log := logger.WithCtx(ctx)

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		log.Error("email token repo: begin tx failed", zap.Error(err))
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`DELETE FROM email_verification_tokens WHERE user_id = $1`,
		token.UserID,
	); err != nil {
		log.Error("email token repo: delete old tokens failed", zap.Error(err), zap.Int("user_id", token.UserID))
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO email_verification_tokens (user_id, token, expires_at, confirmed, created_at)
		VALUES ($1, $2, $3, false, NOW() AT TIME ZONE 'UTC')
	`, token.UserID, token.Token, token.ExpiresAt); err != nil {
		log.Error("email token repo: insert token failed",
			zap.Error(err), zap.Int("user_id", token.UserID))
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("email token repo: commit tx failed", zap.Error(err))
		return err
	}

	log.Info("email token repo: token saved",
		zap.Int("user_id", token.UserID),
		zap.String("token", token.Token),
		zap.Time("expires_at", token.ExpiresAt),
	)
	return nil
}

// VerifyToken — возвращает запись токена по значению.
func (r *EmailTokenRepository) VerifyToken(ctx context.Context, token string) (*models.EmailVerificationToken, error) {
	log := logger.WithCtx(ctx)

	row := r.db.QueryRow(ctx, `
		SELECT user_id, token, expires_at, confirmed, created_at
		FROM email_verification_tokens
		WHERE token = $1
	`, token)

	var t models.EmailVerificationToken
	if err := row.Scan(&t.UserID, &t.Token, &t.ExpiresAt, &t.Confirmed, &t.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			log.Warn("email token repo: token not found", zap.String("token", token))
		} else {
			log.Error("email token repo: select token failed", zap.Error(err))
		}
		return nil, err
	}

	log.Debug("email token repo: token loaded",
		zap.Int("user_id", t.UserID),
		zap.Bool("confirmed", t.Confirmed),
	)
	return &t, nil
}

// MarkConfirmed — помечает токен подтверждённым.
func (r *EmailTokenRepository) MarkConfirmed(ctx context.Context, token string) error {
	log := logger.WithCtx(ctx)

	if _, err := r.db.Exec(ctx,
		`UPDATE email_verification_tokens SET confirmed = true WHERE token = $1`,
		token,
	); err != nil {
		log.Error("email token repo: mark confirmed failed", zap.Error(err))
		return err
	}

	log.Info("email token repo: token confirmed", zap.String("token", token))
	return nil
}

// GetLastTokenByUserID — возвращает последний токен пользователя по времени создания.
func (r *EmailTokenRepository) GetLastTokenByUserID(ctx context.Context, userID int) (*models.EmailVerificationToken, error) {
	log := logger.WithCtx(ctx)

	row := r.db.QueryRow(ctx, `
		SELECT user_id, token, expires_at, confirmed, created_at
		FROM email_verification_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, userID)

	var t models.EmailVerificationToken
	if err := row.Scan(&t.UserID, &t.Token, &t.ExpiresAt, &t.Confirmed, &t.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			log.Warn("email token repo: last token not found", zap.Int("user_id", userID))
		} else {
			log.Error("email token repo: select last token failed", zap.Error(err), zap.Int("user_id", userID))
		}
		return nil, err
	}

	log.Debug("email token repo: last token fetched",
		zap.Int("user_id", t.UserID),
		zap.Time("created_at", t.CreatedAt),
		zap.Time("expires_at", t.ExpiresAt),
	)
	return &t, nil
}
