package main

import (
	"fmt"
	"log"
	"time"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	emailSvc := service.NewEmailService(cfg.GraphAPI)
	
	to := []string{"support@fiberx.iq"} // Send to ourselves as a test
	subject := "Test Email from ShiftMaster Graph API"
	body := "Hello! This is a test email sent using the new Microsoft Graph API integration in Go. If you receive this, the integration is successful!"

	fmt.Println("Sending test email...")
	emailSvc.SendEmailAsync(to, subject, body)
	
	time.Sleep(5 * time.Second)
	fmt.Println("Done waiting.")
}
