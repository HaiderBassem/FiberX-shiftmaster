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

type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, fmt.Errorf("unknown fromServer: %s", string(fromServer))
		}
	}
	return nil, nil
}

func (s *EmailService) sendEmail(to []string, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	var auth smtp.Auth
	// Use LoginAuth for Office365/Outlook, otherwise PlainAuth
	if s.cfg.User != "" && s.cfg.Password != "" {
		hostLower := strings.ToLower(s.cfg.Host)
		if strings.Contains(hostLower, "office365") || strings.Contains(hostLower, "outlook") || strings.Contains(hostLower, "hotmail") {
			auth = LoginAuth(s.cfg.User, s.cfg.Password)
		} else {
			auth = smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)
		}
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
