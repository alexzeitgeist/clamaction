package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Email            string
	Recipients       []string
	Sender           string
	Virus            string
	SMTPHost         string
	SMTPPort         string
	EmailAdmin       string
	EmailService     string
	QuarantineFolder string
	QuarantineFile   string
	Debug            bool
}

// ConfigType is used to specify which configuration to load
type ConfigType int

const (
	Action ConfigType = iota
	Release
)

func Load(configType ConfigType) (*Config, error) {
	var requiredVars []string

	switch configType {
	case Action:
		requiredVars = []string{
			"EMAIL", "VIRUS", "RECIPIENTS", "SENDER", "EMAIL_ADMIN", "EMAIL_SERVICE", "QUARANTINE_FOLDER",
		}
	case Release:
		requiredVars = []string{
			"EMAIL_SERVICE", "QUARANTINE_FOLDER",
		}
	default:
		return nil, fmt.Errorf("invalid config type")
	}

	return loadConfig(requiredVars)
}

func loadConfig(requiredVars []string) (*Config, error) {
	config := &Config{
		SMTPHost: "localhost",
		SMTPPort: "25",
	}

	for _, key := range requiredVars {
		value, err := getEnvVar(key)
		if err != nil {
			return nil, err
		}

		switch key {
		case "EMAIL":
			config.Email = value
		case "VIRUS":
			config.Virus = value
		case "RECIPIENTS":
			config.Recipients = strings.Split(value, "\n")
		case "SENDER":
			config.Sender = value
		case "EMAIL_ADMIN":
			config.EmailAdmin = value
		case "EMAIL_SERVICE":
			config.EmailService = value
		case "QUARANTINE_FOLDER":
			config.QuarantineFolder = strings.TrimRight(value, "/")
		}
	}

	if smtpHost := os.Getenv("SMTP_HOST"); smtpHost != "" {
		config.SMTPHost = smtpHost
	}
	if smtpPort := os.Getenv("SMTP_PORT"); smtpPort != "" {
		config.SMTPPort = smtpPort
	}

	if config.Email != "" && config.QuarantineFolder != "" {
		config.QuarantineFile = filepath.Join(config.QuarantineFolder, filepath.Base(config.Email))
	}

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
