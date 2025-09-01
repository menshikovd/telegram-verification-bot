package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	TelegramToken   string `json:"telegram_token"`
	AdminID         int64  `json:"admin_id"`
	SpreadsheetID   string `json:"spreadsheet_id"`
	CredentialsPath string `json:"credentials_path"`
}

// LoadConfig загружает конфигурацию из файла или переменных окружения
func LoadConfig(path string) (*Config, error) {
	// Проверяем переменные окружения сначала
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	adminIDStr := os.Getenv("ADMIN_ID")
	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	credentialsPath := os.Getenv("CREDENTIALS_PATH")

	// Если все переменные заданы, используем их
	if telegramToken != "" && adminIDStr != "" && spreadsheetID != "" {
		adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ADMIN_ID: %v", err)
		}

		if credentialsPath == "" {
			credentialsPath = "./credentials.json" // значение по умолчанию
		}

		return &Config{
			TelegramToken:   telegramToken,
			AdminID:         adminID,
			SpreadsheetID:   spreadsheetID,
			CredentialsPath: credentialsPath,
		}, nil
	}

	// Если переменных нет, пробуем загрузить из файла
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not load config from file %s and environment variables are not set: %v", path, err)
	}
	defer file.Close()

	var config Config
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
