package service

import (
	"fmt"
	"net/smtp"
	"strings"

	"shiftmaster-backend/internal/config"
)

// EmailService handles sending emails.
type EmailService struct {
	cfg config.SMTPConfig
}

// NewEmailService creates a new EmailService.
func NewEmailService(cfg config.SMTPConfig) *EmailService {
	return &EmailService{cfg: cfg}
}

// SendEmailAsync sends an email in a background goroutine.
func (s *EmailService) SendEmailAsync(to []string, subject, body string) {
	if s.cfg.Host == "" {
		fmt.Println("[EMAIL] SMTP not configured. Skipping email to:", to)
		return
	}

	go func() {
		err := s.sendEmail(to, subject, body)
		if err != nil {
			fmt.Printf("[EMAIL] Failed to send email to %v: %v\n", to, err)
		} else {
			fmt.Printf("[EMAIL] Successfully sent email to %v\n", to)
		}
	}()
}

func (s *EmailService) sendEmail(to []string, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	var auth smtp.Auth
	// Only use authentication if user/password are provided
	if s.cfg.User != "" && s.cfg.Password != "" {
		auth = smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)
	}

	msg := []byte("To: " + strings.Join(to, ",") + "\r\n" +
		"From: " + s.cfg.From + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-version: 1.0;\r\n" +
		"Content-Type: text/plain; charset=\"UTF-8\";\r\n\r\n" +
		body + "\r\n")

	err := smtp.SendMail(addr, auth, s.cfg.From, to, msg)
	return err
}
