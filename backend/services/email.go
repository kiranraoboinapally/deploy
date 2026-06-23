package services

import (
	"fmt"
	"log"
	"net/smtp"

	"backend/config"
)

type EmailService struct {
	cfg *config.Config
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{cfg: cfg}
}

// SendDownloadEmail sends the secure PDF download link to the customer
func (s *EmailService) SendDownloadEmail(toEmail string, customerName string, productName string, downloadToken string, backendHost string) error {
	downloadURL := fmt.Sprintf("%s/api/download/%s", backendHost, downloadToken)

	subject := fmt.Sprintf("Your Download: %s", productName)
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Your Digital Download</title>
    <style>
        body { font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; color: #333; line-height: 1.6; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; border: 1px solid #e0e0e0; padding: 30px; border-radius: 8px; background-color: #fcfcfc; }
        h2 { color: #111; font-weight: 600; margin-top: 0; }
        .button { display: inline-block; padding: 12px 24px; margin: 20px 0; font-size: 16px; color: #fff; background-color: #1a1a1a; text-decoration: none; border-radius: 4px; font-weight: bold; }
        .footer { font-size: 12px; color: #666; margin-top: 30px; border-top: 1px solid #eee; padding-top: 15px; }
        .warning { color: #b91c1c; font-size: 13px; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <h2>Thank you for your purchase, %s!</h2>
        <p>Your payment for <strong>%s</strong> was completed successfully.</p>
        <p>You can download your PDF file by clicking the button below:</p>
        <p style="text-align: center;">
            <a href="%s" class="button" style="color: #ffffff;">Download PDF Now</a>
        </p>
        <p class="warning">Note: This download link is secure and will expire in 24 hours.</p>
        <div class="footer">
            <p>If you have any questions or did not authorize this purchase, please reply to this email.</p>
            <p>Order Reference: %s</p>
        </div>
    </div>
</body>
</html>`, customerName, productName, downloadURL, downloadToken)

	// If SMTP parameters are missing, fallback to mock console logging
	if s.cfg.SMTPHost == "" || s.cfg.SMTPPort == "" || s.cfg.SMTPUser == "" {
		log.Printf("\n--- [MOCK EMAIL IN DEVELOPMENT] ---\n"+
			"To: %s\n"+
			"Subject: %s\n"+
			"Download Link: %s\n"+
			"-----------------------------------\n", toEmail, subject, downloadURL)
		return nil
	}

	// Setup SMTP Auth
	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPass, s.cfg.SMTPHost)

	// Format Email Message headers
	msg := []byte("To: " + toEmail + "\r\n" +
		"From: " + s.cfg.SMTPFrom + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-version: 1.0;\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\";\r\n\r\n" +
		body + "\r\n")

	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, s.cfg.SMTPPort)
	err := smtp.SendMail(addr, auth, s.cfg.SMTPFrom, []string{toEmail}, msg)
	if err != nil {
		return fmt.Errorf("failed to send SMTP email: %w", err)
	}

	log.Printf("Digital delivery email successfully sent to %s", toEmail)
	return nil
}
