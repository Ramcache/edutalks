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
