package domain

import (
	"net/mail"
	"unicode/utf8"

	"github.com/google/uuid"

	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

const (
	displayNameMinLen = 1
	displayNameMaxLen = 50
)

type UserID struct {
	value string
}

func NewUserID(value string) (UserID, error) {
	if _, err := uuid.Parse(value); err != nil {
		return UserID{}, apperrors.NewValidationError("UserID", "must be a valid UUID")
	}
	return UserID{value: value}, nil
}

func GenerateUserID() UserID {
	return UserID{value: uuid.NewString()}
}

func (u UserID) String() string { return u.value }

func (u UserID) Equals(other UserID) bool { return u.value == other.value }

type Email struct {
	value string
}

func NewEmail(value string) (Email, error) {
	if value == "" {
		return Email{}, apperrors.NewValidationError("email", "is required")
	}
	addr, err := mail.ParseAddress(value)
	if err != nil || addr.Address != value {
		return Email{}, apperrors.NewValidationError("email", "must be a valid email address")
	}
	return Email{value: value}, nil
}

func (e Email) String() string { return e.value }

func (e Email) Equals(other Email) bool { return e.value == other.value }

type UserStatus string

const (
	UserStatusActive   UserStatus = "ACTIVE"
	UserStatusInactive UserStatus = "INACTIVE"
)

func NewUserStatus(value string) (UserStatus, error) {
	switch UserStatus(value) {
	case UserStatusActive, UserStatusInactive:
		return UserStatus(value), nil
	default:
		return "", apperrors.NewValidationError("status", "must be ACTIVE or INACTIVE")
	}
}

func (s UserStatus) IsActive() bool { return s == UserStatusActive }

type User struct {
	id          UserID
	email       Email
	displayName string
	status      UserStatus
	events      []DomainEvent
}

// newUser はパッケージ外から直接呼べない。
// 公開すると email 一意性チェック（BR-U-01）をスキップして User を生成できてしまうため、
// 外部からの正規の生成パスは UserRegistrar.Register のみとする。
func newUser(email Email, displayName string) (*User, error) {
	if err := validateDisplayName(displayName); err != nil {
		return nil, err
	}
	return &User{
		id:          GenerateUserID(),
		email:       email,
		displayName: displayName,
		status:      UserStatusActive,
	}, nil
}

// ReconstructUser はリポジトリからの復元用ファクトリ。
// 学習目的で防御的に displayName を再検証している（DB スキーマと不変条件のドリフト検出）。
func ReconstructUser(id UserID, email Email, displayName string, status UserStatus) (*User, error) {
	if err := validateDisplayName(displayName); err != nil {
		return nil, err
	}
	return &User{
		id:          id,
		email:       email,
		displayName: displayName,
		status:      status,
	}, nil
}

func validateDisplayName(value string) error {
	length := utf8.RuneCountInString(value)
	if length < displayNameMinLen || length > displayNameMaxLen {
		return apperrors.NewValidationError("displayName", "must be 1-50 characters")
	}
	return nil
}

func (u *User) ID() UserID          { return u.id }
func (u *User) Email() Email        { return u.email }
func (u *User) DisplayName() string { return u.displayName }
func (u *User) Status() UserStatus  { return u.status }

// Deactivate は ACTIVE → INACTIVE への遷移時のみ UserDeactivated を発行する。
// 既に INACTIVE の場合は BR-U-03 によりバリデーションエラーを返す。
func (u *User) Deactivate() error {
	if u.status == UserStatusInactive {
		return apperrors.NewValidationError("status", "user is already inactive")
	}
	u.status = UserStatusInactive
	u.events = append(u.events, NewUserDeactivated(u.id))
	return nil
}

func (u *User) PullEvents() []DomainEvent {
	events := u.events
	u.events = nil
	return events
}

type UserRepository interface {
	Save(user *User) error
	FindByID(id UserID) (*User, error)
	FindByEmail(email Email) (*User, error)
	ExistsByEmail(email Email) (bool, error)
}
