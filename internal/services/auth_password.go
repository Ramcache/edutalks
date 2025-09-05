package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"edutalks/internal/logger"
	"edutalks/internal/repository"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type PasswordService struct {
	repo        repository.PasswordResetRepo
	emailSender EmailSender // интерфейс из твоего email-сервиса
	appURL      string      // фронтовый URL для ссылки вида /reset?token=...
	tokenTTL    time.Duration
}

type EmailSender interface {
	// Примени к твоему email сервису (у тебя есть handlers/email.go и репозиторий email_token.go — тут только интерфейс)
	SendPasswordReset(ctx context.Context, to, resetLink string) error
}

func NewPasswordService(repo repository.PasswordResetRepo, emailSender EmailSender, appURL string) *PasswordService {
	return &PasswordService{
		repo:        repo,
		emailSender: emailSender,
		appURL:      appURL,
		tokenTTL:    30 * time.Minute,
	}
}

// генерим токен и шлём письмо. Возвращаем НИЧЕГО, чтобы не палить, существует ли юзер.
func (s *PasswordService) RequestReset(ctx context.Context, email string) error {
	userID, err := s.repo.FindUserIDByEmail(ctx, email)
	if err != nil {
		// Не раскрываем, существует ли почта. Просто логируем.
		logger.Log.Info("Password reset requested for unknown email (safe)", zap.String("email", email))
		return nil
	}

	// token (raw)
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)

	// store only hash
	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])

	expires := time.Now().Add(s.tokenTTL)
	if err := s.repo.Create(ctx, userID, tokenHash, expires); err != nil {
		return err
	}

	resetLink := fmt.Sprintf("%s/reset?token=%s", s.appURL, token)
	if err := s.emailSender.SendPasswordReset(ctx, email, resetLink); err != nil {
		logger.Log.Error("Send password reset email failed", zap.Error(err))
		// не фейлим намеренно, чтобы злоумышленник не мог брутить наличие почты
	}
	return nil
}

// подтверждение по токену
func (s *PasswordService) ResetPassword(ctx context.Context, token, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("password too short")
	}

	// hash token to lookup
	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])

	rec, err := s.repo.GetValidByHash(ctx, tokenHash)
	if err != nil {
		return errors.New("invalid or expired token")
	}

	pwHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}

	if err := s.repo.UpdateUserPassword(ctx, rec.UserID, string(pwHash)); err != nil {
		return err
	}
	if err := s.repo.MarkUsed(ctx, rec.ID); err != nil {
		logger.Log.Warn("Failed to mark reset token used", zap.Error(err), zap.Int64("token_id", rec.ID))
	}
	return nil
}

// смена пароля при активной сессии
func (s *PasswordService) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword, currentHash string) (string, error) {
	if len(newPassword) < 8 {
		return "", errors.New("password too short")
	}
	// проверяем старый
	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(oldPassword)); err != nil {
		return "", errors.New("old password incorrect")
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return "", err
	}
	if err := s.repo.UpdateUserPassword(ctx, userID, string(newHash)); err != nil {
		return "", err
	}
	return string(newHash), nil
}
