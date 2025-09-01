package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"telegram_verification_bot/internal/config"
	"telegram_verification_bot/internal/models"
	"telegram_verification_bot/internal/sheets"
)

type Bot struct {
	api            *tgbotapi.BotAPI
	config         *config.Config
	sheets         *sheets.SheetsService
	registrations  map[int64]*models.RegistrationState
	mutex          sync.RWMutex
}

func NewBot(cfg *config.Config) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		return nil, err
	}

	sheetsService, err := sheets.NewSheetsService(cfg.CredentialsPath, cfg.SpreadsheetID)
	if err != nil {
		return nil, err
	}

	api.Debug = false
	log.Printf("Authorized on account %s", api.Self.UserName)

	return &Bot{
		api:           api,
		config:        cfg,
		sheets:        sheetsService,
		registrations: make(map[int64]*models.RegistrationState),
	}, nil
}

func (b *Bot) Start() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			go b.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			go b.handleCallbackQuery(update.CallbackQuery)
		}
	}

	return nil
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	userID := message.From.ID

	// Устанавливаем меню для новых пользователей
	b.ensureMenuSet(message)

	// Обработка команд и кнопок меню
	if message.IsCommand() || b.isMenuButton(message.Text) {
		switch {
		case message.Command() == "start" || message.Text == "🏠 Меню":
			b.handleStart(message)
		case message.Command() == "register" || message.Text == "📝 Регистрация":
			b.handleRegister(message)
		case message.Command() == "status" || message.Text == "📊 Статус":
			b.handleStatus(message)
		case message.Command() == "help" || message.Text == "❓ Справка":
			b.handleHelp(message)
		case message.Text == "👥 Пользователи" && message.From.ID == b.config.AdminID:
			b.handleListUsers(message)
		case message.Text == "🔍 Поиск" && message.From.ID == b.config.AdminID:
			b.handleAdminSearchMode(message)
		case message.Command() == "approve" || message.Command() == "reject":
			b.handleModeration(message)
		case message.Command() == "users":
			b.handleListUsers(message)
		}
		return
	}

	// Обработка процесса регистрации
	b.mutex.RLock()
	reg, exists := b.registrations[userID]
	b.mutex.RUnlock()
	if exists {
		b.handleRegistrationStep(message, reg)
		return
	}

	// Поиск по базе пользователей
	b.handleSearch(message)
}

func (b *Bot) handleStart(message *tgbotapi.Message) {
	text := `👋 Добро пожаловать в бот верификации!

Выберите действие:`

	// Создаем постоянное меню
	keyboard := b.createPermanentMenu(message.From.ID)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) handleRegister(message *tgbotapi.Message) {
	userID := message.From.ID

	// Проверяем, не зарегистрирован ли уже пользователь
	existingUser, _ := b.sheets.GetUser(userID)
	if existingUser != nil {
		var statusText string
		switch existingUser.Status {
		case models.StatusPending:
			statusText = "⏳ На рассмотрении"
		case models.StatusApproved:
			statusText = fmt.Sprintf("✅ Одобрена (роль: %s)", existingUser.Role)
		case models.StatusRejected:
			statusText = "❌ Отклонена"
		}

		text := fmt.Sprintf("Вы уже зарегистрированы!\nСтатус заявки: %s", statusText)
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		b.api.Send(msg)
		return
	}

	// Начинаем процесс регистрации
	reg := &models.RegistrationState{
		TelegramID: userID,
		Step:       models.StepFirstName,
		User: models.User{
			TelegramID:   userID,
			Username:     message.From.UserName,
			RegisterDate: time.Now(),
			Status:       models.StatusPending,
			Role:         models.RoleGuest,
		},
	}

	b.mutex.Lock()
	b.registrations[userID] = reg
	b.mutex.Unlock()

	text := "📝 Начинаем процесс регистрации!\n\n👤 Пожалуйста, введите ваше имя\n*в формате:* Иван"
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) handleRegistrationStep(message *tgbotapi.Message, reg *models.RegistrationState) {
	userID := message.From.ID

	switch reg.Step {
	case models.StepFirstName:
		reg.User.FirstName = message.Text
		reg.Step = models.StepLastName
		text := "✅ Отлично! Теперь введите вашу фамилию\n*в формате:* Иванов"
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		msg.ParseMode = "Markdown"
		b.api.Send(msg)

	case models.StepLastName:
		reg.User.LastName = message.Text
		reg.Step = models.StepPhone
		text := "✅ Хорошо! Введите ваш номер телефона\n*в формате:* +71234567890"
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		msg.ParseMode = "Markdown"
		b.api.Send(msg)

	case models.StepPhone:
		reg.User.Phone = message.Text
		reg.Step = models.StepEmail
		text := "✅ Принято! Введите ваш email\n*пример:* example@mail.com"
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		msg.ParseMode = "Markdown"
		b.api.Send(msg)

	case models.StepEmail:
		reg.User.Email = message.Text
		reg.Step = models.StepAddress
		text := `✅ Отлично! И наконец, введите ваш адрес по образцу:

🏘 *Поселок Green Forest Club:* GFC P11
🏘 *Поселок Green Forest Park:* GFP P11
🏘 *Green Forest Premium:* GFPr P11`
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		msg.ParseMode = "Markdown"
		b.api.Send(msg)

	case models.StepAddress:
		reg.User.Address = message.Text
		reg.Step = models.StepComplete

		// Сохраняем пользователя в Google Sheets
		err := b.sheets.AddUser(&reg.User)
		if err != nil {
			log.Printf("Error adding user to sheets: %v", err)
			text := "❌ Произошла ошибка при сохранении данных. Попробуйте позже."
			msg := tgbotapi.NewMessage(message.Chat.ID, text)
			b.api.Send(msg)
			return
		}

		// Удаляем состояние регистрации
		b.mutex.Lock()
		delete(b.registrations, userID)
		b.mutex.Unlock()

		// Отправляем подтверждение пользователю
		text := fmt.Sprintf(`✅ Регистрация завершена!

📋 Ваши данные:
👤 Имя: %s %s
📱 Телефон: %s
📧 Email: %s
🏠 Адрес: %s

⏳ Ваша заявка отправлена на модерацию. Ожидайте уведомления о результате.`, 
			reg.User.FirstName, reg.User.LastName, reg.User.Phone, reg.User.Email, reg.User.Address)

		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		b.api.Send(msg)

		// Отправляем уведомление администратору
		b.sendAdminNotification(&reg.User)
	}
}

func (b *Bot) sendAdminNotification(user *models.User) {
	text := fmt.Sprintf(`🆕 Новая заявка на верификацию!

👤 Пользователь: %s %s (@%s)
📱 ID: %d
📞 Телефон: %s
📧 Email: %s
🏠 Адрес: %s
📅 Дата: %s`,
		user.FirstName, user.LastName, user.Username, user.TelegramID,
		user.Phone, user.Email, user.Address,
		user.RegisterDate.Format("2006-01-02 15:04:05"))

	// Создаем кнопки для быстрой модерации
	keyboard := b.createModerationMenu(user.TelegramID)

	msg := tgbotapi.NewMessage(b.config.AdminID, text)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// createModerationMenu создает меню модерации для админа
func (b *Bot) createModerationMenu(userID int64) tgbotapi.InlineKeyboardMarkup {
	row1 := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("✅ Житель", fmt.Sprintf("approve_%d_житель", userID)),
		tgbotapi.NewInlineKeyboardButtonData("✅ Сосед", fmt.Sprintf("approve_%d_сосед", userID)),
		tgbotapi.NewInlineKeyboardButtonData("✅ ОК", fmt.Sprintf("approve_%d_ОК", userID)),
	}
	row2 := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("reject_%d", userID)),
	}

	return tgbotapi.NewInlineKeyboardMarkup(row1, row2)
}

func (b *Bot) handleStatus(message *tgbotapi.Message) {
	userID := message.From.ID

	user, err := b.sheets.GetUser(userID)
	if err != nil {
		text := "❓ Вы не найдены в системе. Используйте /register для регистрации."
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		b.api.Send(msg)
		return
	}

	var statusText string
	switch user.Status {
	case models.StatusPending:
		statusText = "⏳ На рассмотрении"
	case models.StatusApproved:
		statusText = fmt.Sprintf("✅ Одобрена (роль: %s)", user.Role)
	case models.StatusRejected:
		statusText = "❌ Отклонена"
		if user.AdminComment != "" {
			statusText += fmt.Sprintf("\nПричина: %s", user.AdminComment)
		}
	}

	text := fmt.Sprintf(`📋 Статус вашей заявки: %s

👤 Имя: %s %s
📧 Email: %s
📅 Дата регистрации: %s`, 
		statusText, user.FirstName, user.LastName, user.Email,
		user.RegisterDate.Format("2006-01-02 15:04"))

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	b.api.Send(msg)
}

func (b *Bot) handleModeration(message *tgbotapi.Message) {
	// Только администратор может модерировать
	if message.From.ID != b.config.AdminID {
		text := "❌ У вас нет прав для выполнения этой команды."
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		b.api.Send(msg)
		return
	}

	args := strings.Fields(message.Text)
	if len(args) < 2 {
		text := "❌ Неверный формат команды.\nИспользуйте: /approve ID роль или /reject ID причина"
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		b.api.Send(msg)
		return
	}

	userID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		text := "❌ Неверный ID пользователя."
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		b.api.Send(msg)
		return
	}

	if message.Command() == "approve" {
		if len(args) < 3 {
			text := "❌ Укажите роль: житель, сосед, ОК"
			msg := tgbotapi.NewMessage(message.Chat.ID, text)
			b.api.Send(msg)
			return
		}

		role := models.UserRole(args[2])
		if role != models.RoleResident && role != models.RoleNeighbor && role != models.RoleOK {
			text := "❌ Недопустимая роль. Используйте: житель, сосед, ОК"
			msg := tgbotapi.NewMessage(message.Chat.ID, text)
			b.api.Send(msg)
			return
		}

		err = b.sheets.UpdateUserStatus(userID, models.StatusApproved, role, "")
		if err != nil {
			text := "❌ Ошибка при обновлении статуса."
			msg := tgbotapi.NewMessage(message.Chat.ID, text)
			b.api.Send(msg)
			return
		}

		// Уведомляем пользователя
		userMsg := tgbotapi.NewMessage(userID, fmt.Sprintf("🎉 Ваша заявка одобрена!\nВаша роль: %s", role))
		b.api.Send(userMsg)

		// Подтверждаем админу
		text := fmt.Sprintf("✅ Пользователь %d одобрен с ролью: %s", userID, role)
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		b.api.Send(msg)

	} else if message.Command() == "reject" {
		reason := strings.Join(args[2:], " ")
		if reason == "" {
			reason = "Не указана"
		}

		err = b.sheets.UpdateUserStatus(userID, models.StatusRejected, models.RoleGuest, reason)
		if err != nil {
			text := "❌ Ошибка при обновлении статуса."
			msg := tgbotapi.NewMessage(message.Chat.ID, text)
			b.api.Send(msg)
			return
		}

		// Уведомляем пользователя
		userMsg := tgbotapi.NewMessage(userID, fmt.Sprintf("❌ Ваша заявка отклонена.\nПричина: %s", reason))
		b.api.Send(userMsg)

		// Подтверждаем админу
		text := fmt.Sprintf("❌ Пользователь %d отклонен. Причина: %s", userID, reason)
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		b.api.Send(msg)
	}
}

func (b *Bot) handleListUsers(message *tgbotapi.Message) {
	// Только администратор может просматривать список
	if message.From.ID != b.config.AdminID {
		text := "❌ У вас нет прав для выполнения этой команды."
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		b.api.Send(msg)
		return
	}

	users, err := b.sheets.GetAllUsers()
	if err != nil {
		text := "❌ Ошибка при получении списка пользователей."
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		b.api.Send(msg)
		return
	}

	if len(users) == 0 {
		text := "📝 Список пользователей пуст."
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		b.api.Send(msg)
		return
	}

	text := "👥 Список пользователей:\n\n"
	for i, user := range users {
		status := string(user.Status)
		switch user.Status {
		case models.StatusPending:
			status = "⏳ На рассмотрении"
		case models.StatusApproved:
			status = "✅ Одобрен"
		case models.StatusRejected:
			status = "❌ Отклонен"
		}

		text += fmt.Sprintf("%d. %s %s (@%s)\n   ID: %d | %s | Роль: %s\n\n",
			i+1, user.FirstName, user.LastName, user.Username,
			user.TelegramID, status, user.Role)

		// Telegram ограничивает размер сообщения
		if len(text) > 3500 {
			msg := tgbotapi.NewMessage(message.Chat.ID, text)
			b.api.Send(msg)
			text = ""
		}
	}

	if text != "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		b.api.Send(msg)
	}
}

func (b *Bot) handleSearch(message *tgbotapi.Message) {
	// Проверяем, зарегистрирован ли пользователь
	userID := message.From.ID
	currentUser, err := b.sheets.GetUser(userID)
	if err != nil || currentUser.Status != models.StatusApproved {
		text := "❓ Для использования поиска необходимо пройти верификацию. Используйте /register"
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		b.api.Send(msg)
		return
	}

	query := strings.ToLower(message.Text)
	users, err := b.sheets.GetAllUsers()
	if err != nil {
		text := "❌ Ошибка при поиске."
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		b.api.Send(msg)
		return
	}

	var results []string
	for _, user := range users {
		if user.Status != models.StatusApproved {
			continue
		}

		// Поиск по всем полям: имя, фамилия, username, телефон, email, адрес
		searchText := strings.ToLower(fmt.Sprintf("%s %s %s %s %s %s",
			user.FirstName, user.LastName, user.Username, user.Phone, user.Email, user.Address))

		if strings.Contains(searchText, query) {
			result := fmt.Sprintf("👤 %s %s (@%s)\n🏠 %s | Роль: %s",
				user.FirstName, user.LastName, user.Username, user.Address, user.Role)
			results = append(results, result)
		}

		if len(results) >= 10 { // Ограничиваем количество результатов
			break
		}
	}

	if len(results) == 0 {
		text := "🔍 По вашему запросу ничего не найдено."
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		b.api.Send(msg)
		return
	}

	text := fmt.Sprintf("🔍 Результаты поиска по запросу \"%s\":\n\n%s",
		message.Text, strings.Join(results, "\n\n"))

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	b.api.Send(msg)
}

func (b *Bot) handleHelp(message *tgbotapi.Message) {
	text := `📚 Справка по боту

👥 Основные команды:
🔹 /start - приветствие и основная информация
🔹 /register - начать процесс регистрации
🔹 /status - проверить статус заявки
🔹 /help - эта справка

🔍 Поиск:
После одобрения заявки вы можете искать других пользователей, просто отправив текстовое сообщение.

👨‍💼 Команды администратора:
🔹 /users - список всех пользователей
🔹 /approve ID роль - одобрить заявку
🔹 /reject ID причина - отклонить заявку

📝 Доступные роли: житель, сосед, ОК`

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	b.api.Send(msg)
}

func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID
	data := callback.Data

	// Отвечаем на callback
	msg := tgbotapi.NewCallback(callback.ID, "")
	b.api.Request(msg)

	// Обрабатываем действия
	switch data {
	case "register":
		// Создаем фейковое сообщение для обработки регистрации
		fakeMsg := &tgbotapi.Message{
			From: callback.From,
			Chat: callback.Message.Chat,
			Text: "/register",
		}
		b.handleRegister(fakeMsg)

	case "status":
		fakeMsg := &tgbotapi.Message{
			From: callback.From,
			Chat: callback.Message.Chat,
			Text: "/status",
		}
		b.handleStatus(fakeMsg)

	case "help":
		fakeMsg := &tgbotapi.Message{
			From: callback.From,
			Chat: callback.Message.Chat,
			Text: "/help",
		}
		b.handleHelp(fakeMsg)

	case "admin_users":
		if userID == b.config.AdminID {
			fakeMsg := &tgbotapi.Message{
				From: callback.From,
				Chat: callback.Message.Chat,
				Text: "/users",
			}
			b.handleListUsers(fakeMsg)
		}

	case "admin_search":
		if userID == b.config.AdminID {
			text := "🔍 Введите запрос для поиска пользователей (имя, фамилия, телефон, email, адрес):"
			msg := tgbotapi.NewMessage(callback.Message.Chat.ID, text)
			b.api.Send(msg)
		}

	// Обработка модерации через кнопки
	default:
		if strings.HasPrefix(data, "approve_") {
			if userID == b.config.AdminID {
				b.handleInlineApproval(callback)
			}
		} else if strings.HasPrefix(data, "reject_") {
			if userID == b.config.AdminID {
				b.handleInlineRejection(callback)
			}
		}
	}
}

// createMainMenu создает основное меню для пользователей
func (b *Bot) createMainMenu(userID int64) tgbotapi.InlineKeyboardMarkup {
	var buttons [][]tgbotapi.InlineKeyboardButton

	// Основные кнопки для всех пользователей
	row1 := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("📝 Зарегистрироваться", "register"),
		tgbotapi.NewInlineKeyboardButtonData("📊 Проверить статус", "status"),
	}
	row2 := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("❓ Справка", "help"),
	}

	buttons = append(buttons, row1, row2)

	// Дополнительные кнопки для администратора
	if userID == b.config.AdminID {
		adminRow1 := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("👥 Список пользователей", "admin_users"),
		}
		adminRow2 := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🔍 Поиск пользователя", "admin_search"),
		}
		buttons = append(buttons, adminRow1, adminRow2)
	}

	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// handleInlineApproval обрабатывает одобрение через inline кнопки
func (b *Bot) handleInlineApproval(callback *tgbotapi.CallbackQuery) {
	parts := strings.Split(callback.Data, "_")
	if len(parts) < 3 {
		return
	}

	userID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	role := models.UserRole(parts[2])

	err = b.sheets.UpdateUserStatus(userID, models.StatusApproved, role, "")
	if err != nil {
		text := "❌ Ошибка при обновлении статуса."
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, text)
		b.api.Send(msg)
		return
	}

	// Уведомляем пользователя
	userMsg := tgbotapi.NewMessage(userID, fmt.Sprintf("🎉 Ваша заявка одобрена!\nВаша роль: %s", role))
	b.api.Send(userMsg)

	// Обновляем сообщение админа
	newText := fmt.Sprintf("✅ Пользователь одобрен с ролью: %s", role)
	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, newText)
	b.api.Send(editMsg)
}

// handleInlineRejection обрабатывает отклонение через inline кнопки
func (b *Bot) handleInlineRejection(callback *tgbotapi.CallbackQuery) {
	parts := strings.Split(callback.Data, "_")
	if len(parts) < 2 {
		return
	}

	userID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	reason := "Отклонено администратором"
	err = b.sheets.UpdateUserStatus(userID, models.StatusRejected, models.RoleGuest, reason)
	if err != nil {
		text := "❌ Ошибка при обновлении статуса."
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, text)
		b.api.Send(msg)
		return
	}

	// Уведомляем пользователя
	userMsg := tgbotapi.NewMessage(userID, fmt.Sprintf("❌ Ваша заявка отклонена.\nПричина: %s", reason))
	b.api.Send(userMsg)

	// Обновляем сообщение админа
	newText := fmt.Sprintf("❌ Пользователь отклонен. Причина: %s", reason)
	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, newText)
	b.api.Send(editMsg)
}

// createPermanentMenu создает постоянное меню с кнопками
func (b *Bot) createPermanentMenu(userID int64) tgbotapi.ReplyKeyboardMarkup {
	var buttons [][]tgbotapi.KeyboardButton

	// Основные кнопки для всех пользователей
	row1 := []tgbotapi.KeyboardButton{
		tgbotapi.NewKeyboardButton("📝 Регистрация"),
		tgbotapi.NewKeyboardButton("📊 Статус"),
	}
	row2 := []tgbotapi.KeyboardButton{
		tgbotapi.NewKeyboardButton("❓ Справка"),
		tgbotapi.NewKeyboardButton("🏠 Меню"),
	}

	buttons = append(buttons, row1, row2)

	// Дополнительные кнопки для администратора
	if userID == b.config.AdminID {
		adminRow := []tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButton("👥 Пользователи"),
			tgbotapi.NewKeyboardButton("🔍 Поиск"),
		}
		buttons = append(buttons, adminRow)
	}

	return tgbotapi.NewReplyKeyboard(buttons...)
}

// isMenuButton проверяет, является ли текст кнопкой меню
func (b *Bot) isMenuButton(text string) bool {
	menuButtons := []string{
		"📝 Регистрация",
		"📊 Статус",
		"❓ Справка",
		"🏠 Меню",
		"👥 Пользователи",
		"🔍 Поиск",
	}

	for _, button := range menuButtons {
		if text == button {
			return true
		}
	}
	return false
}

// handleAdminSearchMode обрабатывает включение режима поиска для админа
func (b *Bot) handleAdminSearchMode(message *tgbotapi.Message) {
	if message.From.ID != b.config.AdminID {
		text := "❌ У вас нет прав для выполнения этой команды."
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		b.api.Send(msg)
		return
	}

	text := "🔍 Введите запрос для поиска пользователей:\n\nМожно искать по: имени, фамилии, телефону, email, адресу"
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	b.api.Send(msg)
}

// ensureMenuSet устанавливает меню для новых пользователей
func (b *Bot) ensureMenuSet(message *tgbotapi.Message) {
	// Проверяем, если это первое сообщение или команда start
	if message.IsCommand() && message.Command() == "start" {
		return // Меню будет установлено в handleStart
	}

	// Для всех остальных сообщений проверяем, есть ли меню
	// Если нет - устанавливаем
	if !b.isMenuButton(message.Text) && !message.IsCommand() {
		// Устанавливаем меню тихо, чтобы не мешать основному функционалу
		keyboard := b.createPermanentMenu(message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "🔄 Меню обновлено")
		msg.ReplyMarkup = keyboard
		b.api.Send(msg)
	}
}
