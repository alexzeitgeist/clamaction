package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func Load() (*Config, error) {
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

func getEnvVar(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("missing required environment variable: %s", key)
	}
	return value, nil
}
