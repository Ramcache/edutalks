package services

import (
	"context"
	"edutalks/internal/models"
	"edutalks/internal/repository"
	"errors"
	"time"

	"github.com/google/uuid"
)

type EmailTokenService struct {
	repo *repository.EmailTokenRepository
}

func NewEmailTokenService(repo *repository.EmailTokenRepository) *EmailTokenService {
	return &EmailTokenService{repo: repo}
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
	return s.repo.MarkConfirmed(ctx, token)
}
