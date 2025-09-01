package main

import (
	"fmt"
	"log"

	"telegram_verification_bot/internal/sheets"
)

func main() {
	// Замените на ваши данные
	credentialsPath := "./configs/credentials.json"
	spreadsheetID := "1DNoIwZkEYGj_famWIyfrnW84Wa7bZHzqCSjyjfm8tzY"

	sheetsService, err := sheets.NewSheetsService(credentialsPath, spreadsheetID)
	if err != nil {
		log.Fatalf("Failed to create sheets service: %v", err)
	}

	fmt.Println("Создаю заголовки в Google Таблице...")
	
	err = sheetsService.SetupHeaders()
	if err != nil {
		log.Fatalf("Failed to setup headers: %v", err)
	}

	fmt.Println("✅ Заголовки успешно созданы!")
	fmt.Println("Проверьте таблицу: https://docs.google.com/spreadsheets/d/1DNoIwZkEYGj_famWIyfrnW84Wa7bZHzqCSjyjfm8tzY/edit")
}
