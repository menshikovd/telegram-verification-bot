package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	TelegramToken     string `json:"telegram_token"`
	AdminID          int64  `json:"admin_id"`
	SpreadsheetID    string `json:"spreadsheet_id"`
	CredentialsPath  string `json:"credentials_path"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
