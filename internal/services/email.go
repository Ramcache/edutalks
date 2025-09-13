package services

import (
	"context"
	"edutalks/internal/config"
	"edutalks/internal/logger"
	"edutalks/internal/utils/helpers"
	"fmt"
	"net/smtp"
	"time"

	"go.uber.org/zap"
)

// Пауза между адресатами; настраивается из .env через NewEmailService.
var emailPerRecipientDelay = 2 * time.Second

type EmailService struct {
	auth smtp.Auth
	from string
	host string
	port string
}

func NewEmailService(cfg *config.Config) *EmailService {
	// Применяем настройку задержки между адресатами из .env
	if d, err := time.ParseDuration(cfg.EmailPerRecipientDelay); err == nil && d >= 0 {
		emailPerRecipientDelay = d
	}

	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPHost)
	s := &EmailService{
		auth: auth,
		from: cfg.SMTPUser,
		host: cfg.SMTPHost,
		port: cfg.SMTPPort,
	}
	logger.Log.Info("Сервис: инициализация EmailService",
		zap.String("smtp_host", s.host),
		zap.String("smtp_port", s.port),
		zap.String("from", s.from),
		zap.Duration("per_recipient_delay", emailPerRecipientDelay),
	)
	return s
}

func (s *EmailService) smtpAddr() string {
	return fmt.Sprintf("%s:%s", s.host, s.port)
}

// Send — текстовое письмо; отправляем по одному получателю с небольшой паузой
func (s *EmailService) Send(to []string, subject, body string) error {
	addr := s.smtpAddr()

	for i, recipient := range to {
		logger.Log.Info("Сервис: отправка письма (plain)",
			zap.String("to", recipient),
			zap.String("subject", subject),
		)

		msg := []byte(
			"From: Edutalks <" + s.from + ">\r\n" +
				"To: " + recipient + "\r\n" +
				"Subject: " + subject + "\r\n" +
				"List-Unsubscribe: <mailto:unsubscribe@edutalks.ru?subject=unsubscribe>, <https://edutalks.ru/unsubscribe>\r\n" +
				"List-Unsubscribe-Post: List-Unsubscribe=One-Click\r\n" +
				"Precedence: bulk\r\n" +
				"Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n" +
				body,
		)

		if err := smtp.SendMail(addr, s.auth, s.from, []string{recipient}, msg); err != nil {
			logger.Log.Error("Сервис: ошибка отправки письма (plain)",
				zap.String("to", recipient),
				zap.String("subject", subject),
				zap.Error(err),
			)
			return err
		}

		logger.Log.Info("Сервис: письмо отправлено (plain)",
			zap.String("to", recipient),
			zap.String("subject", subject),
		)

		// Пауза между адресатами, чтобы сгладить спайки
		if i < len(to)-1 && emailPerRecipientDelay > 0 {
			time.Sleep(emailPerRecipientDelay)
		}
	}
	return nil
}

// SendHTML — HTML-письмо; отправляем по одному получателю с небольшой паузой
func (s *EmailService) SendHTML(to []string, subject, htmlBody string) error {
	addr := s.smtpAddr()

	for i, recipient := range to {
		logger.Log.Info("Сервис: отправка письма (html)",
			zap.String("to", recipient),
			zap.String("subject", subject),
		)

		msg := []byte(
			"From: Edutalks <" + s.from + ">\r\n" +
				"To: " + recipient + "\r\n" +
				"Subject: " + subject + "\r\n" +
				"MIME-Version: 1.0\r\n" +
				"List-Unsubscribe: <mailto:unsubscribe@edutalks.ru?subject=unsubscribe>, <https://edutalks.ru/unsubscribe>\r\n" +
				"List-Unsubscribe-Post: List-Unsubscribe=One-Click\r\n" +
				"Precedence: bulk\r\n" +
				"Content-Type: text/html; charset=\"utf-8\"\r\n\r\n" +
				htmlBody,
		)

		if err := smtp.SendMail(addr, s.auth, s.from, []string{recipient}, msg); err != nil {
			logger.Log.Error("Сервис: ошибка отправки письма (html)",
				zap.String("to", recipient),
				zap.String("subject", subject),
				zap.Error(err),
			)
			return err
		}

		logger.Log.Info("Сервис: письмо отправлено (html)",
			zap.String("to", recipient),
			zap.String("subject", subject),
		)

		// Пауза между адресатами, чтобы сгладить спайки
		if i < len(to)-1 && emailPerRecipientDelay > 0 {
			time.Sleep(emailPerRecipientDelay)
		}
	}
	return nil
}

func (s *EmailService) SendPasswordReset(ctx context.Context, to, resetLink string) error {
	subject := "Восстановление пароля"
	htmlBody := helpers.BuildPasswordResetHTML(resetLink)

	logger.Log.Info("Сервис: формирование письма для восстановления пароля",
		zap.String("to", to),
	)

	if err := s.SendHTML([]string{to}, subject, htmlBody); err != nil {
		logger.Log.Error("Сервис: ошибка отправки письма восстановления",
			zap.String("to", to),
			zap.Error(err),
		)
		return err
	}

	logger.Log.Info("Сервис: письмо восстановления отправлено", zap.String("to", to))
	return nil
}

func (s *EmailService) SendSubscriptionGranted(ctx context.Context, to, name, planLabel string, expiresAt time.Time) error {
	subject := "Подписка активирована"
	body := helpers.BuildSubscriptionGrantedHTML(name, planLabel, expiresAt.Format("02.01.2006 15:04"))

	logger.Log.Info("Сервис: формирование письма об активации подписки",
		zap.String("to", to),
		zap.String("plan", planLabel),
		zap.Time("expires_at", expiresAt),
	)

	if err := s.SendHTML([]string{to}, subject, body); err != nil {
		logger.Log.Error("Сервис: ошибка отправки письма об активации подписки",
			zap.String("to", to),
			zap.Error(err),
		)
		return err
	}

	logger.Log.Info("Сервис: письмо об активации подписки отправлено",
		zap.String("to", to),
		zap.String("plan", planLabel),
	)
	return nil
}

func (s *EmailService) SendSubscriptionRevoked(ctx context.Context, to, name string, revokedAt time.Time, prevExpiresAt *time.Time) error {
	subject := "Подписка отключена"
	body := helpers.BuildSubscriptionRevokedHTML(name, revokedAt, prevExpiresAt)

	logger.Log.Info("Сервис: формирование письма об отключении подписки",
		zap.String("to", to),
		zap.Time("revoked_at", revokedAt),
		zap.Bool("had_prev_expiry", prevExpiresAt != nil),
	)

	if err := s.SendHTML([]string{to}, subject, body); err != nil {
		logger.Log.Error("Сервис: ошибка отправки письма об отключении подписки",
			zap.String("to", to),
			zap.Error(err),
		)
		return err
	}

	logger.Log.Info("Сервис: письмо об отключении подписки отправлено", zap.String("to", to))
	return nil
}
