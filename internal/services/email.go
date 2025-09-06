package services

import (
	"context"
	"edutalks/internal/config"
	"edutalks/internal/utils/helpers"
	"fmt"
	"net/smtp"
	"time"
)

type EmailService struct {
	auth smtp.Auth
	from string
	host string
	port string
}

func NewEmailService(cfg *config.Config) *EmailService {
	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPHost)
	return &EmailService{
		auth: auth,
		from: cfg.SMTPUser,
		host: cfg.SMTPHost,
		port: cfg.SMTPPort,
	}
}

func (s *EmailService) Send(to []string, subject, body string) error {
	for _, recipient := range to {
		msg := []byte(
			"From: Edutalks <" + s.from + ">\r\n" +
				"To: " + recipient + "\r\n" +
				"Subject: " + subject + "\r\n" +
				"Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n" +
				body,
		)
		addr := fmt.Sprintf("%s:%s", s.host, s.port)
		err := smtp.SendMail(addr, s.auth, s.from, []string{recipient}, msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *EmailService) SendHTML(to []string, subject, htmlBody string) error {
	for _, recipient := range to {
		msg := []byte(
			"From: Edutalks <" + s.from + ">\r\n" +
				"To: " + recipient + "\r\n" +
				"Subject: " + subject + "\r\n" +
				"MIME-Version: 1.0\r\n" +
				"Content-Type: text/html; charset=\"utf-8\"\r\n\r\n" +
				htmlBody,
		)
		addr := fmt.Sprintf("%s:%s", s.host, s.port)
		err := smtp.SendMail(addr, s.auth, s.from, []string{recipient}, msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *EmailService) SendPasswordReset(ctx context.Context, to, resetLink string) error {
	subject := "Восстановление пароля"
	htmlBody := helpers.BuildPasswordResetHTML(resetLink)
	// отправляем как HTML
	return s.SendHTML([]string{to}, subject, htmlBody)
}

func (s *EmailService) SendSubscriptionGranted(ctx context.Context, to, name, planLabel string, expiresAt time.Time) error {
	subject := "Подписка активирована"
	body := helpers.BuildSubscriptionGrantedHTML(name, planLabel, expiresAt.Format("02.01.2006 15:04"))
	return s.SendHTML([]string{to}, subject, body)
}

func (s *EmailService) SendSubscriptionRevoked(ctx context.Context, to, name string, revokedAt time.Time, prevExpiresAt *time.Time) error {
	subject := "Подписка отключена"
	body := helpers.BuildSubscriptionRevokedHTML(name, revokedAt, prevExpiresAt)
	return s.SendHTML([]string{to}, subject, body)
}
