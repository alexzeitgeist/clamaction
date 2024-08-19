package smtp

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"time"
)

func Send(smtpHost, smtpPort, sender, receiver string, msg []byte) error {
	const duration = 10 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%s", smtpHost, smtpPort))
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	client, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer func(client *smtp.Client) {
		err := client.Quit()
		if err != nil {
		}
	}(client)

	if err := client.Mail(sender); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}
	if err := client.Rcpt(receiver); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	if _, err := writer.Write(msg); err != nil {
		return fmt.Errorf("failed to write email data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return nil
}
