package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
)

type Config struct {
	// clamsmtpd
	Email      string
	Recipients []string
	Sender     string
	Virus      string

	// smtp
	SMTPHost     string
	SMTPPort     string
	EmailAdmin   string
	EmailService string

	// app
	QuarantineFolder string
	QuarantineFile   string
	Debug            bool
}

type Header struct {
	Key   string
	Value string
}

const (
	SubjectQuarantined string = "[QUARANTINED] Potentially Infected Email"
	AdminEmailTemplate string = `* * * * * * * * * * * * * Virus ALERT * * * * * * * * * *

A potentially infected email send to one or more of your users was detected.

Sender: %s
Virus: %s
Recipients: %s

----- Forwarded headers from %s -----

%s
`
	RecipientEmailTemplate string = `* * * * * * * * * * * * * Virus ALERT * * * * * * * * * *

A potentially infected email was sent to you. The email has been quarantined for your safety.

Contact your admin %s if you need assistance.

Sender: %s
Virus: %s
Quarantine ID: %s

Mail-Info:
--8<--

%s
--8<--
`
)

var debugEnabled bool

func main() {
	log.SetOutput(os.Stdout)

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	debugEnabled = config.Debug

	if err := quarantineVirus(config); err != nil {
		log.Fatalf("Failed to quarantine virus: %v", err)
	}

	emlFile, err := os.ReadFile(config.QuarantineFile)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	headers, err := parseHeaders(emlFile)
	if err != nil {
		log.Fatalf("Failed to parse headers: %v", err)
	}

	if err := notifyAdmin(config, emlFile, headers); err != nil {
		log.Fatalf("Failed to notify admin: %v", err)
	}

	for _, recipient := range config.Recipients {
		if recipient != "" {
			if err := notifyRecipient(config, recipient, headers); err != nil {
				log.Fatalf("Failed to notify recipient: %v", err)
			}
		}
	}

}

func getEnvVar(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("missing required environment variable: %s", key)
	}
	return value, nil
}

func loadConfig() (*Config, error) {
	config := &Config{
		SMTPHost: "localhost",
		SMTPPort: "25",
		Debug:    false,
	}

	var err error

	config.Email, err = getEnvVar("EMAIL")
	if err != nil {
		return nil, err
	}

	config.Virus, err = getEnvVar("VIRUS")
	if err != nil {
		return nil, err
	}

	recipients, err := getEnvVar("RECIPIENTS")
	if err != nil {
		return nil, err
	}
	config.Recipients = strings.Split(recipients, "\n")

	config.Sender, err = getEnvVar("SENDER")
	if err != nil {
		return nil, err
	}

	if smtpHost := os.Getenv("SMTP_HOST"); smtpHost != "" {
		config.SMTPHost = smtpHost
	}

	if smtpPort := os.Getenv("SMTP_PORT"); smtpPort != "" {
		config.SMTPPort = smtpPort
	}

	config.EmailAdmin, err = getEnvVar("EMAIL_ADMIN")
	if err != nil {
		return nil, err
	}

	config.EmailService, err = getEnvVar("EMAIL_SERVICE")
	if err != nil {
		return nil, err
	}

	config.QuarantineFolder, err = getEnvVar("QUARANTINE_FOLDER")
	if err != nil {
		return nil, err
	}
	config.QuarantineFolder = strings.TrimRight(config.QuarantineFolder, "/")

	config.QuarantineFile = filepath.Join(config.QuarantineFolder, filepath.Base(config.Email))

	config.Debug = os.Getenv("DEBUG") == "true"

	return config, nil
}

func quarantineVirus(config *Config) error {
	srcFile, err := os.Open(config.Email)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(config.QuarantineFile)
	if err != nil {
		return fmt.Errorf("failed to create quarantine file: %v", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file to quarantine: %v", err)
	}

	err = destFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync quarantine file: %v", err)
	}

	err = os.Remove(config.Email)
	if err != nil {
		return fmt.Errorf("failed to delete original file after quarantine: %v", err)
	}

	return nil
}

func notifyAdmin(config *Config, emlFile []byte, headers []Header) error {
	formattedHeaders := formatHeaders(headers)

	adminContent := fmt.Sprintf(
		AdminEmailTemplate,
		config.Sender,
		config.Virus,
		strings.Join(config.Recipients, ", "),
		config.Sender,
		formattedHeaders,
	)

	return prepareEmail(config.EmailService, config.EmailAdmin, adminContent, config.SMTPHost, config.SMTPPort, emlFile, filepath.Base(config.QuarantineFile))
}

func notifyRecipient(config *Config, recipient string, headers []Header) error {
	filename := filepath.Base(config.QuarantineFile)
	quarantineId := filename[strings.LastIndex(filename, ".")+1:]
	sanitizedSender := strings.ReplaceAll(config.Sender, "@", "[at]")
	sanitizedSender = strings.ReplaceAll(sanitizedSender, ".", "[dot]")

	formattedHeaders := formatSelectedHeaders(headers, []string{"Message-Id", "Sender", "From", "To", "Date", "Subject"})

	emailContent := fmt.Sprintf(
		RecipientEmailTemplate,
		config.EmailAdmin,
		sanitizedSender,
		config.Virus,
		quarantineId,
		formattedHeaders,
	)

	return prepareEmail(config.EmailService, recipient, emailContent, config.SMTPHost, config.SMTPPort, nil, "")
}

func parseHeaders(emlContent []byte) ([]Header, error) {
	msg, err := message.Read(bytes.NewReader(emlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %v", err)
	}

	var headers []Header
	fields := msg.Header.Fields()
	for fields.Next() {
		key := fields.Key()
		rawValue := fields.Value()

		decodedValue, err := (&mime.WordDecoder{}).DecodeHeader(rawValue)
		if err != nil {
			decodedValue = rawValue // Use raw value as a fallback
		}

		headers = append(headers, Header{Key: key, Value: decodedValue})
	}

	return headers, nil
}

func formatHeaders(headers []Header) string {
	var headerBuilder strings.Builder
	for _, header := range headers {
		headerLine := fmt.Sprintf("%s: %s", header.Key, header.Value)
		lines := splitLongLines(headerLine, 76)

		for i, ln := range lines {
			if i == 0 {
				headerBuilder.WriteString(ln + "\n")
			} else {
				headerBuilder.WriteString("\t" + ln + "\n")
			}
		}
	}
	return headerBuilder.String()
}

func formatSelectedHeaders(headers []Header, targetHeaders []string) string {

	targetHeadersMap := make(map[string]struct{})
	for _, key := range targetHeaders {
		targetHeadersMap[key] = struct{}{}
	}

	var headerBuilder strings.Builder

	for _, header := range headers {
		if _, wanted := targetHeadersMap[header.Key]; wanted {
			headerBuilder.WriteString(fmt.Sprintf("%s: %s\n", header.Key, header.Value))
		}
	}

	return headerBuilder.String()
}

func splitLongLines(s string, maxLength int) []string {

	var lines []string
	for len(s) > maxLength {
		idx := strings.LastIndexAny(s[:maxLength], " \t")
		if idx == -1 || idx == 0 {
			// If no space/tab found or it's at the beginning,
			// force a split at maxLength
			idx = maxLength
		}
		lines = append(lines, s[:idx])
		s = strings.TrimSpace(s[idx:])
	}
	if len(s) > 0 {
		lines = append(lines, s)
	}
	return lines
}

func prepareEmail(sender, recipient, content, smtpHost, smtpPort string, attachment []byte, filename string) error {

	var buf bytes.Buffer
	var contentType string
	var writer *multipart.Writer

	hasAttachment := attachment != nil
	if hasAttachment {
		writer = multipart.NewWriter(&buf)
		contentType = fmt.Sprintf("multipart/mixed; boundary=%s", writer.Boundary())
	} else {
		contentType = "text/plain; charset=UTF-8"
	}

	headers := textproto.MIMEHeader{
		"From":         {sender},
		"To":           {recipient},
		"Subject":      {SubjectQuarantined},
		"MIME-Version": {"1.0"},
		"Content-Type": {contentType},
	}

	for k, v := range headers {
		_, _ = fmt.Fprintf(&buf, "%s: %s\r\n", k, strings.Join(v, " "))
	}

	buf.WriteString("\r\n")

	if hasAttachment {
		partWriter, _ := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain; charset=UTF-8"}})
		partWriter.Write([]byte(content))

		if filename == "" {
			filename = "attachment.eml"
		}
		attHeader := textproto.MIMEHeader{
			"Content-Type":        {"message/rfc822"},
			"Content-Disposition": {fmt.Sprintf("attachment; filename=\"%s.eml\"", filename)},
		}
		partWriter, _ = writer.CreatePart(attHeader)
		partWriter.Write(attachment)

		if err := writer.Close(); err != nil {
			return fmt.Errorf("failed to close multipart writer: %w", err)
		}
	} else {
		buf.WriteString(content)
	}

	return sendSMTP(smtpHost, smtpPort, sender, recipient, buf.Bytes())
}

func sendSMTP(smtpHost, smtpPort, sender, receiver string, msg []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
