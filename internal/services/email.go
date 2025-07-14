package services

import (
	"edutalks/internal/config"
	"fmt"
	"net/smtp"
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
	msg := []byte("Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n" +
		body)

	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	return smtp.SendMail(addr, s.auth, s.from, to, msg)
}
