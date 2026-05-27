package messages

import (
	"fmt"
	"os"
	"path/filepath"
)

const basePath = "database/messages"

func filePath(channelID string) string {
	return filepath.Join(basePath, fmt.Sprintf("%s.txt", channelID))
}

func Append(channelID string, message string) error {
	path := filePath(channelID)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create directories: %w", err)
	}

	file, err := os.OpenFile(
		path,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(message + "\n"); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

func Read(channelID string) (string, error) {
	data, err := os.ReadFile(filePath(channelID))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}

		return "", fmt.Errorf("read file: %w", err)
	}

	return string(data), nil
}

func Clear(channelID string) error {
	path := filePath(channelID)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create directories: %w", err)
	}

	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return fmt.Errorf("clear file: %w", err)
	}
	defer file.Close()

	return nil
}

func Open(channelID string) (*os.File, error) {
	path := filePath(channelID)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}
