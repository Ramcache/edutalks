package services

import (
	"context"
	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/repository"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type EmailTokenService struct {
	repo     *repository.EmailTokenRepository
	userRepo *repository.UserRepository
}

func NewEmailTokenService(repo *repository.EmailTokenRepository, userRepo *repository.UserRepository) *EmailTokenService {
	return &EmailTokenService{repo: repo, userRepo: userRepo}
}

var (
	ErrTokenInvalid = errors.New("неверный токен")
	ErrTokenExpired = errors.New("токен истёк")
)

func (s *EmailTokenService) GenerateToken(ctx context.Context, userID int) (*models.EmailVerificationToken, error) {
	token := uuid.New().String()
	expires := time.Now().Add(24 * time.Hour)

	t := &models.EmailVerificationToken{
		UserID:    userID,
		Token:     token,
		ExpiresAt: expires,
		CreatedAt: time.Now(),
	}
	if err := s.repo.SaveToken(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *EmailTokenService) ConfirmToken(ctx context.Context, token string) error {
	t, err := s.repo.VerifyToken(ctx, token)
	if err != nil {
		return ErrTokenInvalid
	}
	if t.ExpiresAt.Before(time.Now()) {
		return ErrTokenExpired
	}
	if t.Confirmed {
		return ErrTokenInvalid
	}
	if err := s.repo.MarkConfirmed(ctx, token); err != nil {
		return err
	}
	if err := s.userRepo.SetEmailVerified(ctx, t.UserID, true); err != nil {
		return err
	}
	return nil
}

type EmailJob struct {
	To      []string
	Subject string
	Body    string
	IsHTML  bool
}

var (
	EmailQueue = make(chan EmailJob, 100)
	closeOnce  sync.Once
)

// StartEmailWorker — неблокирующий почтовый воркер с ID для логов.
func StartEmailWorker(id int, emailService *EmailService) {
	go func(workerID int) {
		logger.Log.Info("Сервис: email-воркер запущен", zap.Int("worker_id", workerID))
		for job := range EmailQueue {
			var err error
			if job.IsHTML {
				err = emailService.SendHTML(job.To, job.Subject, job.Body)
			} else {
				err = emailService.Send(job.To, job.Subject, job.Body)
			}
			if err != nil {
				logger.Log.Error("Не удалось отправить письмо",
					zap.Int("worker_id", workerID),
					zap.Strings("to", job.To),
					zap.String("subject", job.Subject),
					zap.Error(err),
				)
				continue
			}
			logger.Log.Info("Письмо отправлено (SMTP accepted)",
				zap.Int("worker_id", workerID),
				zap.Strings("to", job.To),
				zap.String("subject", job.Subject),
			)
		}
		logger.Log.Info("Email-воркер остановлен", zap.Int("worker_id", workerID))
	}(id)
}

// StopEmailWorkers — корректно закрывает очередь (воркеры завершатся сами).
func StopEmailWorkers() {
	closeOnce.Do(func() {
		close(EmailQueue)
		logger.Log.Info("Email-очередь закрыта")
	})
}

// GetLastTokenByUserID — вернёт последний токен подтверждения e-mail для пользователя.
// Используется для антиспама при повторной отправке письма.
func (s *EmailTokenService) GetLastTokenByUserID(ctx context.Context, userID int) (*models.EmailVerificationToken, error) {
	return s.repo.GetLastTokenByUserID(ctx, userID)
}
