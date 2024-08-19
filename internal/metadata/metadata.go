package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Metadata struct {
	MailFrom       string   `json:"envelope_sender"`
	Recipients     []string `json:"envelope_recipients"`
	Virus          string   `json:"virus_name"`
	QuarantineTime string   `json:"quarantine_time"`
}

func New(sender string, recipients []string, virus string) *Metadata {
	return &Metadata{
		QuarantineTime: time.Now().Format(time.RFC3339),
		Virus:          virus,
		MailFrom:       sender,
		Recipients:     recipients,
	}
}

func Save(filePath string, metadata *Metadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata to JSON: %v", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata JSON file: %v", err)
	}

	return nil
}

func Load(filePath string) (*Metadata, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata JSON file: %v", err)
	}

	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON metadata: %v", err)
	}

	return &metadata, nil
}
