package main

import (
	"github.com/alexzeitgeist/clamaction/internal/config"
	"github.com/alexzeitgeist/clamaction/internal/headers"
	"github.com/alexzeitgeist/clamaction/internal/metadata"
	"github.com/alexzeitgeist/clamaction/internal/notification"
	"github.com/alexzeitgeist/clamaction/internal/quarantine"
	"log"
	"os"
)

func main() {
	log.SetOutput(os.Stdout)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if err := quarantine.Store(cfg.Email, cfg.QuarantineFile); err != nil {
		log.Fatalf("Failed to quarantine virus: %v", err)
	}

	meta := metadata.New(cfg.Sender, cfg.Recipients, cfg.Virus)
	if err := metadata.Save(cfg.QuarantineFile+".json", meta); err != nil {
		log.Fatalf("Failed to save email metadata: %v", err)
	}

	emlFile, err := os.ReadFile(cfg.QuarantineFile)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	hdrs, err := headers.Parse(emlFile)
	if err != nil {
		log.Fatalf("Failed to parse headers: %v", err)
	}

	if err := notification.EmailAdmin(cfg, emlFile, hdrs); err != nil {
		log.Fatalf("Failed to notify admin: %v", err)
	}

	for _, recipient := range cfg.Recipients {
		if recipient != "" {
			if err := notification.EmailRecipient(cfg, recipient, hdrs); err != nil {
				log.Fatalf("Failed to notify recipient: %v", err)
			}
		}
	}
}
