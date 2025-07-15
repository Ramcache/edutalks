package services

import (
	"context"
	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/repository"
	"errors"
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
	}
	err := s.repo.SaveToken(ctx, t)
	return t, err
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

var EmailQueue = make(chan EmailJob, 100) // глобальная очередь на 100 писем

func StartEmailWorker(emailService *EmailService) {
	go func() {
		for job := range EmailQueue {
			var err error
			if job.IsHTML {
				err = emailService.SendHTML(job.To, job.Subject, job.Body)
			} else {
				err = emailService.Send(job.To, job.Subject, job.Body)
			}
			if err != nil {
				// Используй свой логгер!
				logger.Log.Error("Не удалось отправить письмо", zap.Error(err))
			}
		}
	}()
}
