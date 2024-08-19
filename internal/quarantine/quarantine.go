package quarantine

import (
	"fmt"
	"io"
	"os"
)

func Store(emailFile, quarantineFile string) error {
	srcFile, err := os.Open(emailFile)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(quarantineFile)
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

	err = os.Remove(emailFile)
	if err != nil {
		return fmt.Errorf("failed to delete original file after quarantine: %v", err)
	}

	return nil
}
