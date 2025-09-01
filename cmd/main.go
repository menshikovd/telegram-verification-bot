package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"telegram_verification_bot/internal/bot"
	"telegram_verification_bot/internal/config"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig("configs/config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Создаем и запускаем бота
	b, err := bot.NewBot(cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	log.Println("🚀 Bot started successfully!")
	log.Printf("Admin ID: %d", cfg.AdminID)
	log.Println("📚 Available commands:")
	log.Println("  /start - welcome message")
	log.Println("  /register - start registration process")  
	log.Println("  /status - check application status")
	log.Println("  /help - show help")
	log.Println("  /users - list all users (admin only)")
	log.Println("  /approve ID role - approve user (admin only)")
	log.Println("  /reject ID reason - reject user (admin only)")
	log.Println()
	log.Println("Press Ctrl+C to stop the bot")

	// Обрабатываем сигналы для корректного завершения
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		log.Println("🛑 Shutting down bot...")
		os.Exit(0)
	}()

	// Запускаем бота
	if err := b.Start(); err != nil {
		log.Fatalf("Bot error: %v", err)
	}
}
