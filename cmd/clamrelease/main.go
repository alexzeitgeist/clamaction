package main

import (
	"bytes"
	"fmt"
	"log"
	"net/mail"
	"net/textproto"
	"os"
	"path/filepath"
	"time"
	"unicode"

	"github.com/alexzeitgeist/clamaction/internal/config"
	"github.com/alexzeitgeist/clamaction/internal/metadata"
	"github.com/alexzeitgeist/clamaction/internal/smtp"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: app <quarantine_id>")
	}
	id := os.Args[1]

	if len(id) != 6 {
		log.Fatal("Invalid quarantine ID format. Must be exactly 6 letters.")
	}

	for _, ch := range id {
		if !unicode.IsLetter(ch) {
			log.Fatal("Invalid quarantine ID format. Must contain only letters.")
		}
	}

	cfg, err := config.Load(config.Release)
	if err != nil {
		log.Fatal("Error loading configuration:", err)
	}

	filePath := filepath.Join(cfg.QuarantineFolder, "virus."+id+".json")
	data, err := metadata.Load(filePath)
	if err != nil {
		log.Fatalf("Error loading metadata: %v", err)
	}

	eml, err := os.ReadFile(filepath.Join(cfg.QuarantineFolder, "virus."+id))
	if err != nil {
		log.Fatalf("Error loading virus: %v", err)
	}

	EnvelopeSender := data.MailFrom
	EnvelopeRecipients := data.Recipients

	headers := make(textproto.MIMEHeader)
	var msg bytes.Buffer

	for _, recipient := range EnvelopeRecipients {
		headers.Set("Resent-From", (&mail.Address{Address: cfg.EmailService}).String())
		headers.Set("Resent-To", (&mail.Address{Address: recipient}).String())
		headers.Set("Resent-Date", time.Now().Format(time.RFC1123Z))

		for key, values := range headers {
			for _, value := range values {
				fmt.Fprintf(&msg, "%s: %s\r\n", key, value)
			}
		}
		msg.Write(eml)

		err = smtp.Send(cfg.SMTPHost, cfg.SMTPPort, EnvelopeSender, recipient, msg.Bytes())
		if err != nil {
			fmt.Printf("Failed to send email: %v\n", err)
		}
	}
}
