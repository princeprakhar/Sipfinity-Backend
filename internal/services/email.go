package services

import (
	"crypto/tls"
	"fmt"

	"github.com/princeprakhar/ecommerce-backend/internal/config"
	"gopkg.in/gomail.v2"
)

type EmailService struct {
	config *config.Config
}

func NewEmailService(config *config.Config) *EmailService {
	return &EmailService{config: config}
}

func (s *EmailService) SendEmail(to, subject, body string, attachmentPath ...string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.config.FromEmail)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	// Add attachment if provided
	for _, path := range attachmentPath {
		if path != "" {
			m.Attach(path)
		}
	}

	d := gomail.NewDialer(s.config.SMTPHost, s.config.SMTPPort, s.config.SMTPUsername, s.config.SMTPPassword)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	return d.DialAndSend(m)
}

func (s *EmailService) SendProductUploadNotification(adminEmail, filePath string, productCount int) error {
	subject := "Product Upload Completed"
	body := fmt.Sprintf(`
		<h2>Product Upload Notification</h2>
		<p>Your product upload has been processed successfully.</p>
		<p><strong>Total Products Processed:</strong> %d</p>
		<p>Please find the processed Excel file attached.</p>
		<p>Best regards,<br>Your E-commerce Team</p>
	`, productCount)

	return s.SendEmail(adminEmail, subject, body, filePath)
}

func (s *EmailService) SendPasswordResetEmail(email, resetToken, baseURL string) error {
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", baseURL, resetToken)

	subject := "Password Reset Request"
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .button { 
            display: inline-block; 
            padding: 12px 24px; 
            background-color: #4CAF50; 
            color: white; 
            text-decoration: none; 
            border-radius: 4px; 
            margin: 20px 0;
        }
        .footer { padding: 20px; text-align: center; font-size: 12px; color: #666; }
        .warning { background-color: #fff3cd; border-left: 4px solid #ffc107; padding: 10px; margin: 15px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Password Reset Request</h1>
        </div>
        <div class="content">
            <p>Hello,</p>
            <p>We received a request to reset your password for your account associated with <strong>%s</strong>.</p>
            <p>Click the button below to reset your password:</p>
            <p style="text-align: center;">
                <a href="%s" class="button">Reset Password</a>
            </p>
            <p>Or copy and paste this link in your browser:</p>
            <p style="word-break: break-all; background-color: #f0f0f0; padding: 10px; border-radius: 4px;">%s</p>
            
            <div class="warning">
                <strong>Security Notice:</strong>
                <ul>
                    <li>This link will expire in 1 hour for security reasons</li>
                    <li>If you didn't request this password reset, please ignore this email</li>
                    <li>Never share this link with anyone</li>
                </ul>
            </div>
        </div>
        <div class="footer">
            <p>This is an automated message, please do not reply to this email.</p>
            <p>&copy; 2025 Your Company Name. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`, email, resetLink, resetLink)

	return s.SendEmail(email, subject, body)
}
