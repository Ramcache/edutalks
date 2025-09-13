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
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type PasswordService struct {
	repo        repository.PasswordResetRepo
	emailSender EmailSender // интерфейс отправки писем
	appURL      string      // фронтовый URL: https://example.com  (ссылка вида /reset?token=...)
	tokenTTL    time.Duration
}

type EmailSender interface {
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

// RequestReset генерирует одноразовый токен и отправляет письмо со ссылкой.
// Возвращает nil всегда (не раскрываем существует ли такой e-mail).
func (s *PasswordService) RequestReset(ctx context.Context, email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	logger.Log.Info("Запрос на сброс пароля", zap.String("email", email))

	userID, err := s.repo.FindUserIDByEmail(ctx, email)
	if err != nil {
		// Не раскрываем наличие почты пользователю, но логируем для нас:
		logger.Log.Warn("Не удалось найти пользователя по email при запросе сброса",
			zap.String("email", email),
			zap.Error(err),
		)
		return nil
	}

	// Сгенерировать криптостойкий токен
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		logger.Log.Error("Ошибка генерации токена для сброса", zap.Error(err), zap.Int64("user_id", userID))
		// Также не раскрываем детали клиенту
		return nil
	}
	token := base64.RawURLEncoding.EncodeToString(raw)

	// В базе храним только хеш
	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])

	expires := time.Now().Add(s.tokenTTL)
	if err := s.repo.Create(ctx, userID, tokenHash, expires); err != nil {
		logger.Log.Error("Ошибка сохранения токена сброса пароля",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return nil
	}

	resetLink := fmt.Sprintf("%s/reset?token=%s", s.appURL, token)
	if err := s.emailSender.SendPasswordReset(ctx, email, resetLink); err != nil {
		logger.Log.Error("Ошибка отправки письма для сброса пароля",
			zap.Int64("user_id", userID),
			zap.String("email", email),
			zap.Error(err),
		)
		// Не фейлим намеренно — чтобы нельзя было брутить наличие e-mail
	}

	logger.Log.Info("Письмо со ссылкой на сброс пароля поставлено на отправку",
		zap.Int64("user_id", userID),
		zap.String("email", email),
		zap.Time("expires_at", expires),
	)
	return nil
}

// ResetPassword подтверждает токен и устанавливает новый пароль.
func (s *PasswordService) ResetPassword(ctx context.Context, token, newPassword string) error {
	logger.Log.Info("Попытка сброса пароля по токену")

	if len(newPassword) < 8 {
		logger.Log.Warn("Слишком короткий новый пароль")
		return errors.New("password too short")
	}

	// Ищем по хешу токена
	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])

	rec, err := s.repo.GetValidByHash(ctx, tokenHash)
	if err != nil {
		logger.Log.Warn("Неверный или просроченный токен при сбросе пароля", zap.Error(err))
		return errors.New("invalid or expired token")
	}

	pwHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		logger.Log.Error("Ошибка генерации хеша пароля", zap.Error(err), zap.Int64("user_id", rec.UserID))
		return err
	}

	if err := s.repo.UpdateUserPassword(ctx, rec.UserID, string(pwHash)); err != nil {
		logger.Log.Error("Ошибка обновления пароля пользователя",
			zap.Int64("user_id", rec.UserID),
			zap.Error(err),
		)
		return err
	}

	if err := s.repo.MarkUsed(ctx, rec.ID); err != nil {
		logger.Log.Warn("Не удалось пометить токен сброса как использованный",
			zap.Error(err),
			zap.Int64("token_id", rec.ID),
			zap.Int64("user_id", rec.UserID),
		)
	}

	logger.Log.Info("Пароль успешно сброшен", zap.Int64("user_id", rec.UserID))
	return nil
}

// ChangePassword меняет пароль для авторизованного пользователя по старому паролю.
func (s *PasswordService) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword, currentHash string) (string, error) {
	logger.Log.Info("Смена пароля (авторизованный пользователь)", zap.Int64("user_id", userID))

	if len(newPassword) < 8 {
		logger.Log.Warn("Слишком короткий новый пароль", zap.Int64("user_id", userID))
		return "", errors.New("password too short")
	}

	// Проверяем старый пароль
	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(oldPassword)); err != nil {
		logger.Log.Warn("Старый пароль не совпадает", zap.Int64("user_id", userID))
		return "", errors.New("old password incorrect")
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		logger.Log.Error("Ошибка генерации нового хеша пароля", zap.Error(err), zap.Int64("user_id", userID))
		return "", err
	}

	if err := s.repo.UpdateUserPassword(ctx, userID, string(newHash)); err != nil {
		logger.Log.Error("Ошибка обновления пароля пользователя",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return "", err
	}

	logger.Log.Info("Пароль успешно изменён", zap.Int64("user_id", userID))
	return string(newHash), nil
}
