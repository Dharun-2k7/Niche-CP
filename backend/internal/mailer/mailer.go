package mailer

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
)

// SendEmail sends an email using SMTP or logs to console if SMTP is not configured
func SendEmail(to string, subject string, body string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")

	// If no SMTP host is configured, just log to console (great for local dev)
	if host == "" {
		log.Printf("\n========== MOCK EMAIL ==========\nTo: %s\nSubject: %s\n\n%s\n================================\n", to, subject, body)
		return nil
	}

	auth := smtp.PlainAuth("", user, pass, host)
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", to, subject, body))

	addr := fmt.Sprintf("%s:%s", host, port)
	err := smtp.SendMail(addr, auth, user, []string{to}, msg)
	if err != nil {
		log.Printf("Failed to send email to %s: %v", to, err)
		return err
	}

	return nil
}
