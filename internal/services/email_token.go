package services

import (
	"context"
	"edutalks/internal/config"
	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/repository"
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Параметры воркера — задаются из .env через ConfigureEmailWorkerFromEnv
var (
	emailSendInterval = 10 * time.Second // задержка между заданиями воркера
	emailMaxRetries   = 6                // кол-во ретраев для временных ошибок
	emailBaseBackoff  = 30 * time.Second // базовый backoff (экспонента + джиттер)
	emailBatchSize    = 25               // сколько адресатов в одном батче
)

// ConfigureEmailWorkerFromEnv — вызови один раз при старте (после LoadConfig)
func ConfigureEmailWorkerFromEnv(cfg *config.Config) {
	if d, err := time.ParseDuration(cfg.EmailSendInterval); err == nil && d >= 0 {
		emailSendInterval = d
	}
	if d, err := time.ParseDuration(cfg.EmailBaseBackoff); err == nil && d > 0 {
		emailBaseBackoff = d
	}
	if v, err := strconv.Atoi(cfg.EmailMaxRetries); err == nil && v >= 0 {
		emailMaxRetries = v
	}
	if v, err := strconv.Atoi(cfg.EmailBatchSize); err == nil && v > 0 {
		emailBatchSize = v
	}
	logger.Log.Info("Email-воркер: применены настройки из .env",
		zap.Duration("send_interval", emailSendInterval),
		zap.Int("max_retries", emailMaxRetries),
		zap.Duration("base_backoff", emailBaseBackoff),
		zap.Int("batch_size", emailBatchSize),
	)
}

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

// -------------------------------------------------
// Очередь и воркеры
// -------------------------------------------------

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

// StartEmailWorker — воркер с глобальным троттлингом, ретраями и автонарезкой по batch size.
func StartEmailWorker(id int, emailService *EmailService) {
	go func(workerID int) {
		logger.Log.Info("Сервис: email-воркер запущен", zap.Int("worker_id", workerID))

		ticker := time.NewTicker(emailSendInterval)
		defer ticker.Stop()

		for job := range EmailQueue {
			<-ticker.C // квота перед обработкой задания

			batches := ChunkEmails(job.To, emailBatchSize)
			for bi, batch := range batches {
				var err error
				for attempt := 0; attempt <= emailMaxRetries; attempt++ {
					if job.IsHTML {
						err = emailService.SendHTML(batch, job.Subject, job.Body)
					} else {
						err = emailService.Send(batch, job.Subject, job.Body)
					}
					if err == nil {
						logger.Log.Info("Письмо отправлено (SMTP accepted)",
							zap.Int("worker_id", workerID),
							zap.Int("batch_index", bi),
							zap.Int("batch_size", len(batch)),
							zap.String("subject", job.Subject),
						)
						break
					}
					if !isTempSMTPError(err) || attempt == emailMaxRetries {
						logger.Log.Error("Не удалось отправить письмо",
							zap.Int("worker_id", workerID),
							zap.Int("batch_index", bi),
							zap.Int("batch_size", len(batch)),
							zap.String("subject", job.Subject),
							zap.Int("attempt", attempt),
							zap.Error(err),
						)
						break
					}
					// backoff + джиттер
					sleep := emailBaseBackoff * time.Duration(1<<attempt)
					jitter := time.Duration(rand.Int63n(int64(emailBaseBackoff / 2)))
					time.Sleep(sleep + jitter)
				}

				// Пауза между батчами (кроме последнего), чтобы сгладить поток
				if bi < len(batches)-1 && emailSendInterval > 0 {
					time.Sleep(emailSendInterval)
				}
			}
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

// Heuristic: временная SMTP-ошибка (чаще всего 451/4xx/4.7.x)
func isTempSMTPError(err error) bool {
	if err == nil {
		return false
	}
	es := strings.ToLower(err.Error())
	return strings.Contains(es, " 4") || strings.Contains(es, "451") || strings.Contains(es, "4.7")
}

// Вспомогательная нарезка адресов на батчи
func ChunkEmails(emails []string, size int) [][]string {
	if size <= 0 {
		size = 50
	}
	var res [][]string
	for i := 0; i < len(emails); i += size {
		end := i + size
		if end > len(emails) {
			end = len(emails)
		}
		res = append(res, emails[i:end])
	}
	return res
}
