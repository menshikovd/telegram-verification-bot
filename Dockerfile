# Многоэтапная сборка для оптимизации размера
FROM golang:1.21-alpine AS builder

# Установка зависимостей для сборки
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Копируем go mod файлы
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bot cmd/main.go

# Финальный образ
FROM alpine:latest

# Установка CA сертификатов для HTTPS запросов
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Копируем скомпилированное приложение
COPY --from=builder /app/bot .

# Копируем credentials файл (если нужен)
COPY credentials.json ./

# Создаем пользователя для безопасности
RUN adduser -D -s /bin/sh botuser
RUN chown -R botuser:botuser /app
USER botuser

# Порт не нужен для Telegram бота, но указываем для документации
EXPOSE 8080

# Запуск приложения
CMD ["./bot"]
