package sheets

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"telegram_verification_bot/internal/models"
)

type SheetsService struct {
	service       *sheets.Service
	spreadsheetID string
}

func NewSheetsService(credentialsPath, spreadsheetID string) (*SheetsService, error) {
	ctx := context.Background()
	
	service, err := sheets.NewService(ctx, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		return nil, fmt.Errorf("unable to create sheets service: %v", err)
	}

	return &SheetsService{
		service:       service,
		spreadsheetID: spreadsheetID,
	}, nil
}

// SetupHeaders создает заголовки в таблице
func (s *SheetsService) SetupHeaders() error {
	headers := []interface{}{
		"User ID", "Username", "Имя", "Фамилия", "Телефон", 
		"Email", "Адрес", "Дата регистрации", "Статус", "Роль", "Админ комментарий",
	}

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{headers},
	}

	_, err := s.service.Spreadsheets.Values.Update(
		s.spreadsheetID,
		"A1:K1",
		valueRange,
	).ValueInputOption("RAW").Do()

	if err != nil {
		return fmt.Errorf("unable to write headers: %v", err)
	}

	return nil
}

// AddUser добавляет нового пользователя в таблицу
func (s *SheetsService) AddUser(user *models.User) error {
	values := []interface{}{
		user.TelegramID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.Phone,
		user.Email,
		user.Address,
		user.RegisterDate.Format("2006-01-02 15:04:05"),
		string(user.Status),
		string(user.Role),
		user.AdminComment,
	}

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{values},
	}

	_, err := s.service.Spreadsheets.Values.Append(
		s.spreadsheetID,
		"A:K",
		valueRange,
	).ValueInputOption("RAW").Do()

	if err != nil {
		return fmt.Errorf("unable to add user: %v", err)
	}

	return nil
}

// GetUser получает пользователя по Telegram ID
func (s *SheetsService) GetUser(telegramID int64) (*models.User, error) {
	resp, err := s.service.Spreadsheets.Values.Get(
		s.spreadsheetID,
		"A:K",
	).Do()

	if err != nil {
		return nil, fmt.Errorf("unable to get user data: %v", err)
	}

	for i, row := range resp.Values {
		if i == 0 { // Пропускаем заголовки
			continue
		}
		
		if len(row) > 0 {
			if fmt.Sprintf("%v", row[0]) == fmt.Sprintf("%d", telegramID) {
				user := &models.User{}
				if len(row) > 0 { user.TelegramID = telegramID }
				if len(row) > 1 { user.Username = fmt.Sprintf("%v", row[1]) }
				if len(row) > 2 { user.FirstName = fmt.Sprintf("%v", row[2]) }
				if len(row) > 3 { user.LastName = fmt.Sprintf("%v", row[3]) }
				if len(row) > 4 { user.Phone = fmt.Sprintf("%v", row[4]) }
				if len(row) > 5 { user.Email = fmt.Sprintf("%v", row[5]) }
				if len(row) > 6 { user.Address = fmt.Sprintf("%v", row[6]) }
				if len(row) > 8 { user.Status = models.UserStatus(fmt.Sprintf("%v", row[8])) }
				if len(row) > 9 { user.Role = models.UserRole(fmt.Sprintf("%v", row[9])) }
				if len(row) > 10 { user.AdminComment = fmt.Sprintf("%v", row[10]) }
				
				return user, nil
			}
		}
	}

	return nil, fmt.Errorf("user not found")
}

// UpdateUserStatus обновляет статус и роль пользователя
func (s *SheetsService) UpdateUserStatus(telegramID int64, status models.UserStatus, role models.UserRole, comment string) error {
	resp, err := s.service.Spreadsheets.Values.Get(
		s.spreadsheetID,
		"A:K",
	).Do()

	if err != nil {
		return fmt.Errorf("unable to get data: %v", err)
	}

	for i, row := range resp.Values {
		if i == 0 { // Пропускаем заголовки
			continue
		}
		
		if len(row) > 0 && fmt.Sprintf("%v", row[0]) == fmt.Sprintf("%d", telegramID) {
			rowIndex := i + 1
			
			// Обновляем статус (колонка I)
			statusRange := fmt.Sprintf("I%d", rowIndex)
			statusValue := &sheets.ValueRange{
				Values: [][]interface{}{{string(status)}},
			}
			_, err = s.service.Spreadsheets.Values.Update(
				s.spreadsheetID,
				statusRange,
				statusValue,
			).ValueInputOption("RAW").Do()
			
			if err != nil {
				return fmt.Errorf("unable to update status: %v", err)
			}

			// Обновляем роль (колонка J)
			roleRange := fmt.Sprintf("J%d", rowIndex)
			roleValue := &sheets.ValueRange{
				Values: [][]interface{}{{string(role)}},
			}
			_, err = s.service.Spreadsheets.Values.Update(
				s.spreadsheetID,
				roleRange,
				roleValue,
			).ValueInputOption("RAW").Do()
			
			if err != nil {
				return fmt.Errorf("unable to update role: %v", err)
			}

			// Обновляем комментарий (колонка K)
			if comment != "" {
				commentRange := fmt.Sprintf("K%d", rowIndex)
				commentValue := &sheets.ValueRange{
					Values: [][]interface{}{{comment}},
				}
				_, err = s.service.Spreadsheets.Values.Update(
					s.spreadsheetID,
					commentRange,
					commentValue,
				).ValueInputOption("RAW").Do()
				
				if err != nil {
					return fmt.Errorf("unable to update comment: %v", err)
				}
			}

			return nil
		}
	}

	return fmt.Errorf("user not found")
}

// GetAllUsers получает всех пользователей
func (s *SheetsService) GetAllUsers() ([]*models.User, error) {
	resp, err := s.service.Spreadsheets.Values.Get(
		s.spreadsheetID,
		"A:K",
	).Do()

	if err != nil {
		return nil, fmt.Errorf("unable to get users: %v", err)
	}

	var users []*models.User
	for i, row := range resp.Values {
		if i == 0 { // Пропускаем заголовки
			continue
		}
		
		if len(row) > 0 {
			user := &models.User{}
			if len(row) > 0 { 
				fmt.Sscanf(fmt.Sprintf("%v", row[0]), "%d", &user.TelegramID)
			}
			if len(row) > 1 { user.Username = fmt.Sprintf("%v", row[1]) }
			if len(row) > 2 { user.FirstName = fmt.Sprintf("%v", row[2]) }
			if len(row) > 3 { user.LastName = fmt.Sprintf("%v", row[3]) }
			if len(row) > 4 { user.Phone = fmt.Sprintf("%v", row[4]) }
			if len(row) > 5 { user.Email = fmt.Sprintf("%v", row[5]) }
			if len(row) > 6 { user.Address = fmt.Sprintf("%v", row[6]) }
			if len(row) > 7 { 
				user.RegisterDate, _ = time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%v", row[7]))
			}
			if len(row) > 8 { user.Status = models.UserStatus(fmt.Sprintf("%v", row[8])) }
			if len(row) > 9 { user.Role = models.UserRole(fmt.Sprintf("%v", row[9])) }
			if len(row) > 10 { user.AdminComment = fmt.Sprintf("%v", row[10]) }
			
			users = append(users, user)
		}
	}

	return users, nil
}
