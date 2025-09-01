#!/bin/bash

echo "🔨 Building Telegram Bot for DigitalOcean App Platform"

# Сборка Go приложения
go mod tidy
go build -o bot cmd/main.go

echo "✅ Build completed successfully"
