package vm

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// sendSMTPEmail sends an email using SMTP
func sendSMTPEmail(host, port, username, password, from, to, subject, body, htmlBody string) error {
	// Setup authentication
	auth := smtp.PlainAuth("", username, password, host)

	// Prepare email content
	contentType := "text/plain"
	content := body
	if htmlBody != "" {
		contentType = "text/html"
		content = htmlBody
	}

	// Build email message
	msg := []byte(fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: %s; charset=UTF-8\r\n"+
		"\r\n"+
		"%s\r\n", from, to, subject, contentType, content))

	// Determine server address
	addr := host + ":" + port

	// Try TLS first (for ports like 587)
	if port == "587" || port == "465" {
		return sendWithTLS(addr, auth, from, []string{to}, msg)
	}

	// Fallback to plain SMTP
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

// sendWithTLS sends email with TLS encryption
func sendWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// Extract host from addr
	host := strings.Split(addr, ":")[0]

	// Connect to server
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("dial failed: %v", err)
	}
	defer client.Close()

	// Start TLS
	tlsConfig := &tls.Config{
		ServerName: host,
	}
	if err = client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("StartTLS failed: %v", err)
	}

	// Authenticate
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("auth failed: %v", err)
	}

	// Set sender
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("Mail failed: %v", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("Rcpt failed: %v", err)
		}
	}

	// Send message
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("Data failed: %v", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("Write failed: %v", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("Close failed: %v", err)
	}

	return client.Quit()
}
