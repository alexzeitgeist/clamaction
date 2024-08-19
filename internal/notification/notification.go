package notification

import (
	"bytes"
	"fmt"
	"github.com/alexzeitgeist/clamaction/internal/config"
	"github.com/alexzeitgeist/clamaction/internal/headers"
	"github.com/alexzeitgeist/clamaction/internal/smtp"
	"mime/multipart"
	"net/textproto"
	"path/filepath"
	"strings"
)

// Email templates
const (
	SubjectQuarantined string = "[QUARANTINED] Potentially Infected Email"
	AdminEmailTemplate string = `* * * * * * * * * * * * * Virus ALERT * * * * * * * * * *

A potentially infected email sent to one or more of your users was detected.

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

func EmailAdmin(config *config.Config, emlFile []byte, hdrs []headers.Header) error {
	formattedHdrs := headers.Format(hdrs)

	adminContent := fmt.Sprintf(
		AdminEmailTemplate,
		config.Sender,
		config.Virus,
		strings.Join(config.Recipients, ", "),
		config.Sender,
		formattedHdrs,
	)

	return prepareEmail(config.EmailService, config.EmailAdmin, adminContent, config.SMTPHost, config.SMTPPort, emlFile, filepath.Base(config.QuarantineFile))
}

func EmailRecipient(config *config.Config, recipient string, hdrs []headers.Header) error {
	filename := filepath.Base(config.QuarantineFile)
	quarantineId := filename[strings.LastIndex(filename, ".")+1:]
	sanitizedSender := strings.ReplaceAll(config.Sender, "@", "[at]")
	sanitizedSender = strings.ReplaceAll(sanitizedSender, ".", "[dot]")

	formattedHdrs := headers.FormatSelected(hdrs, []string{"Message-Id", "Sender", "From", "To", "Date", "Subject"})

	emailContent := fmt.Sprintf(
		RecipientEmailTemplate,
		config.EmailAdmin,
		sanitizedSender,
		config.Virus,
		quarantineId,
		formattedHdrs,
	)

	return prepareEmail(config.EmailService, recipient, emailContent, config.SMTPHost, config.SMTPPort, nil, "")
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

	hdrs := textproto.MIMEHeader{
		"From":         {sender},
		"To":           {recipient},
		"Subject":      {SubjectQuarantined},
		"MIME-Version": {"1.0"},
		"Content-Type": {contentType},
	}

	for k, v := range hdrs {
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

	return smtp.Send(smtpHost, smtpPort, sender, recipient, buf.Bytes())
}
