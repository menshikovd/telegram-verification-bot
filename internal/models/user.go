package models

import "time"

// UserRole определяет роль пользователя
type UserRole string

const (
	RoleGuest    UserRole = "гость"
	RoleResident UserRole = "житель"
	RoleNeighbor UserRole = "сосед"
	RoleOK       UserRole = "ОК"
)

// UserStatus определяет статус верификации
type UserStatus string

const (
	StatusPending  UserStatus = "pending"
	StatusApproved UserStatus = "approved"
	StatusRejected UserStatus = "rejected"
)

// User представляет пользователя в системе
type User struct {
	TelegramID    int64      `json:"telegram_id"`
	Username      string     `json:"username"`
	FirstName     string     `json:"first_name"`
	LastName      string     `json:"last_name"`
	Phone         string     `json:"phone"`
	Email         string     `json:"email"`
	Address       string     `json:"address"`
	RegisterDate  time.Time  `json:"register_date"`
	Status        UserStatus `json:"status"`
	Role          UserRole   `json:"role"`
	AdminComment  string     `json:"admin_comment"`
}

// RegistrationState хранит состояние процесса регистрации
type RegistrationState struct {
	TelegramID int64
	Step       int
	User       User
}

// Registration steps
const (
	StepFirstName = iota
	StepLastName
	StepPhone
	StepEmail
	StepAddress
	StepComplete
)
