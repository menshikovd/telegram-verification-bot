# Настройка конфигурации

## 1. Google Credentials
Поместите скачанный JSON файл с credentials в эту папку и переименуйте его в `credentials.json`

## 2. Config.json
Скопируйте `config.example.json` в `config.json` и заполните нужные значения:

```bash
cp config.example.json config.json
```

Затем отредактируйте config.json:
- `telegram_token`: токен от @BotFather
- `admin_id`: ваш Telegram ID (можете узнать через @userinfobot)
- `spreadsheet_id`: ID Google таблицы из URL
- `credentials_path`: путь к файлу credentials.json

## 3. Структура файлов в configs/
```
configs/
├── README.md
├── config.example.json
├── config.json        (создать самостоятельно)
└── credentials.json   (скачать из Google Cloud)
```
