#!/bin/bash

# Скрипт для деплоя Telegram бота на DigitalOcean
# Использование: ./deploy.sh

set -e

echo "🚀 Деплой Telegram бота на DigitalOcean"

# Проверяем, что все переменные заданы
if [ -z "$TELEGRAM_TOKEN" ] || [ -z "$ADMIN_ID" ] || [ -z "$SPREADSHEET_ID" ]; then
    echo "❌ Ошибка: Не заданы обязательные переменные окружения"
    echo "Создайте файл .env или экспортируйте переменные:"
    echo "export TELEGRAM_TOKEN=your_token"
    echo "export ADMIN_ID=your_id"
    echo "export SPREADSHEET_ID=your_spreadsheet_id"
    exit 1
fi

# Проверяем наличие credentials.json
if [ ! -f "credentials.json" ]; then
    echo "❌ Ошибка: Файл credentials.json не найден"
    echo "Скачайте файл сервисного аккаунта Google и поместите его в корень проекта"
    exit 1
fi

echo "📦 Сборка Docker образа..."
docker build -t telegram-verification-bot .

echo "🏷️ Тегирование для DigitalOcean Container Registry..."
# Замените 'your-registry' на ваш реальный registry
docker tag telegram-verification-bot registry.digitalocean.com/your-registry/telegram-bot:latest

echo "📤 Загрузка образа в Container Registry..."
docker push registry.digitalocean.com/your-registry/telegram-bot:latest

echo "✅ Деплой завершен!"
echo "Теперь можете запустить бота на сервере командой:"
echo "docker run -d --name telegram-bot --env-file .env registry.digitalocean.com/your-registry/telegram-bot:latest"
