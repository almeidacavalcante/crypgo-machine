package notification

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strings"
)

type EmailService struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	fromEmail    string
}

type EmailData struct {
	To      string
	Subject string
	Body    string
}

func NewEmailService() *EmailService {
	return &EmailService{
		smtpHost:     getEnvOrDefault("SMTP_HOST", "smtp.gmail.com"),
		smtpPort:     getEnvOrDefault("SMTP_PORT", "587"),
		smtpUsername: getEnvOrDefault("SMTP_USERNAME", ""),
		smtpPassword: getEnvOrDefault("SMTP_PASSWORD", ""),
		fromEmail:    getEnvOrDefault("FROM_EMAIL", ""),
	}
}

func (e *EmailService) SendEmail(emailData EmailData) error {
	if e.smtpUsername == "" || e.smtpPassword == "" {
		fmt.Printf("⚠️ SMTP credentials not configured, simulating email send:\n")
		fmt.Printf("📧 To: %s\n", emailData.To)
		fmt.Printf("📧 Subject: %s\n", emailData.Subject)
		fmt.Printf("📧 Body:\n%s\n", emailData.Body)
		fmt.Printf("---\n")
		return nil
	}

	// Prepare message
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		e.fromEmail, emailData.To, emailData.Subject, emailData.Body)

	// Send email with TLS support
	err := e.sendMailTLS(emailData.To, []byte(msg))

	if err != nil {
		fmt.Printf("❌ Error sending email: %v\n", err)
		return err
	}

	fmt.Printf("✅ Email sent successfully to %s\n", emailData.To)
	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return strings.TrimSpace(value)
	}
	return defaultValue
}

func (e *EmailService) sendMailTLS(to string, msg []byte) error {
	// Connect to server
	serverName := e.smtpHost + ":" + e.smtpPort
	
	// TLS config
	tlsConfig := &tls.Config{
		ServerName:         e.smtpHost,
		InsecureSkipVerify: false,
	}

	// Different approach for different ports
	if e.smtpPort == "465" {
		// SSL/TLS from the start (port 465)
		conn, err := tls.Dial("tcp", serverName, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect via SSL: %v", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, e.smtpHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %v", err)
		}
		defer client.Quit()

		return e.authenticateAndSend(client, to, msg)
	} else {
		// STARTTLS (port 587)
		conn, err := net.Dial("tcp", serverName)
		if err != nil {
			return fmt.Errorf("failed to connect: %v", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, e.smtpHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %v", err)
		}
		defer client.Quit()

		// Start TLS
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %v", err)
		}

		return e.authenticateAndSend(client, to, msg)
	}
}

func (e *EmailService) authenticateAndSend(client *smtp.Client, to string, msg []byte) error {
	// Authenticate
	auth := smtp.PlainAuth("", e.smtpUsername, e.smtpPassword, e.smtpHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}

	// Set sender
	if err := client.Mail(e.fromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %v", err)
	}

	// Set recipient
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %v", err)
	}

	// Send message
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %v", err)
	}
	defer writer.Close()

	if _, err := writer.Write(msg); err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}

	return nil
}