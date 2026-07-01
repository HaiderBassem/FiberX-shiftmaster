package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"shiftmaster-backend/internal/config"
)

// EmailService handles sending emails via Microsoft Graph API.
type EmailService struct {
	cfg config.GraphAPIConfig
}

// NewEmailService creates a new EmailService using Microsoft Graph API config.
func NewEmailService(cfg config.GraphAPIConfig) *EmailService {
	return &EmailService{cfg: cfg}
}

// SendEmailAsync sends an email in a background goroutine using Microsoft Graph API.
func (s *EmailService) SendEmailAsync(to []string, subject, body string) {
	if s.cfg.ClientID == "" || s.cfg.TenantID == "" || s.cfg.ClientSecret == "" {
		fmt.Println("[EMAIL] Graph API credentials not fully configured. Skipping email to:", to)
		return
	}

	go func() {
		err := s.sendEmailGraph(to, subject, body)
		if err != nil {
			fmt.Printf("[EMAIL] Failed to send email to %v via Graph API: %v\n", to, err)
		} else {
			fmt.Printf("[EMAIL] Successfully sent email to %v via Graph API\n", to)
		}
	}()
}

// getAccessToken fetches an OAuth2 access token from Azure AD using client credentials.
func (s *EmailService) getAccessToken() (string, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", s.cfg.TenantID)

	data := url.Values{}
	data.Set("client_id", s.cfg.ClientID)
	data.Set("client_secret", s.cfg.ClientSecret)
	data.Set("scope", "https://graph.microsoft.com/.default")
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get token: %s", string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	token, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("access_token not found in response")
	}
	return token, nil
}

// sendEmailGraph sends an email using the Microsoft Graph API /sendMail endpoint.
func (s *EmailService) sendEmailGraph(to []string, subject, body string) error {
	return s.sendEmailGraphWithCC(to, nil, subject, body)
}

// SendEmailWithCCAsync sends an email in a background goroutine including CC recipients.
func (s *EmailService) SendEmailWithCCAsync(to, cc []string, subject, body string) {
	if s.cfg.ClientID == "" || s.cfg.TenantID == "" || s.cfg.ClientSecret == "" {
		fmt.Println("[EMAIL] Graph API credentials not fully configured. Skipping email to:", to)
		return
	}

	go func() {
		err := s.sendEmailGraphWithCC(to, cc, subject, body)
		if err != nil {
			fmt.Printf("[EMAIL] Failed to send email to %v via Graph API: %v\n", to, err)
		} else {
			fmt.Printf("[EMAIL] Successfully sent email to %v via Graph API\n", to)
		}
	}()
}

func (s *EmailService) sendEmailGraphWithCC(to, cc []string, subject, body string) error {
	token, err := s.getAccessToken()
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	graphURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/sendMail", s.cfg.SenderEmail)

	// Construct the Graph API payload
	toRecipients := make([]map[string]interface{}, 0, len(to))
	for _, recipient := range to {
		toRecipients = append(toRecipients, map[string]interface{}{
			"emailAddress": map[string]string{
				"address": strings.TrimSpace(recipient),
			},
		})
	}

	ccRecipients := make([]map[string]interface{}, 0, len(cc))
	for _, recipient := range cc {
		ccRecipients = append(ccRecipients, map[string]interface{}{
			"emailAddress": map[string]string{
				"address": strings.TrimSpace(recipient),
			},
		})
	}

	messageBody := map[string]interface{}{
		"subject": subject,
		"body": map[string]string{
			"contentType": "Text",
			"content":     body,
		},
		"toRecipients": toRecipients,
	}

	if len(ccRecipients) > 0 {
		messageBody["ccRecipients"] = ccRecipients
	}

	message := map[string]interface{}{
		"message": messageBody,
		"saveToSentItems": "false",
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", graphURL, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to send email. status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
