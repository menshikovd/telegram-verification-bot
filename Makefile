.PHONY: build run setup-sheets clean help

# Основные команды
build:
	@echo "🔨 Building bot..."
	go build -o bin/telegram_bot cmd/main.go

run: build
	@echo "🚀 Starting bot..."
	./bin/telegram_bot

setup-sheets:
	@echo "📋 Setting up Google Sheets headers..."
	go run cmd/setup_sheets.go

clean:
	@echo "🧹 Cleaning..."
	rm -rf bin/

deps:
	@echo "📦 Installing dependencies..."
	go mod tidy
	go mod download

test:
	@echo "🧪 Running tests..."
	go test ./...

help:
	@echo "Available commands:"
	@echo "  make build        - Build the bot binary"
	@echo "  make run          - Build and run the bot"
	@echo "  make setup-sheets - Setup Google Sheets headers"
	@echo "  make deps         - Install dependencies"
	@echo "  make clean        - Clean build files"
	@echo "  make test         - Run tests"
	@echo "  make help         - Show this help"
