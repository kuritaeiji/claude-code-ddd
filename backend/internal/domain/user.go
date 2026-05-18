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
		return UserID{}, apperrors.NewValidationError("validation.user_id.invalid", nil)
	}
	return UserID{value: value}, nil
}

// RestoreUserID はリポジトリからの復元専用ファクトリ。
// 検証を行わないため、業務ロジックから呼び出してはならない（BR-U-* の入力検証経路を迂回する）。
// 呼び出し側はインフラ層のリポジトリ実装に限定すること。
func RestoreUserID(value string) UserID {
	return UserID{value: value}
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
		return Email{}, apperrors.NewValidationError("validation.email.required", nil)
	}
	addr, err := mail.ParseAddress(value)
	if err != nil || addr.Address != value {
		return Email{}, apperrors.NewValidationError("validation.email.invalid", nil)
	}
	return Email{value: value}, nil
}

// RestoreEmail はリポジトリからの復元専用ファクトリ。
// 検証を行わないため、業務ロジックから呼び出してはならない（BR-U-* の入力検証経路を迂回する）。
// 呼び出し側はインフラ層のリポジトリ実装に限定すること。
func RestoreEmail(value string) Email {
	return Email{value: value}
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
		return "", apperrors.NewValidationError("validation.user_status.invalid", nil)
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

// ReconstructUser はリポジトリからの復元専用ファクトリ。
// 復元は「過去に妥当だった状態の再構築」であり、業務ルール（BR-U-02 等）の再検査は行わない。
// 業務ルール変更時に過去レコードが取得不能になることを避けるため、
// および DB 不整合は ValidationError ではなく内部エラーとして扱うべきであるため。
// 呼び出し側はインフラ層のリポジトリ実装に限定すること。
func ReconstructUser(id UserID, email Email, displayName string, status UserStatus) *User {
	return &User{
		id:          id,
		email:       email,
		displayName: displayName,
		status:      status,
	}
}

func validateDisplayName(value string) error {
	length := utf8.RuneCountInString(value)
	if length < displayNameMinLen || length > displayNameMaxLen {
		return apperrors.NewValidationError("validation.display_name.length", map[string]any{
			"Min": displayNameMinLen,
			"Max": displayNameMaxLen,
		})
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
		return apperrors.NewValidationError("validation.user.already_inactive", nil)
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

// UserRepository は新規作成と更新を別メソッドに分ける。
// gorm の `Save` が「ID 有無で INSERT/UPDATE を切り替える」挙動はインフラの都合であり、
// ユースケースは登録（U-01）と非活性化（U-02）の文脈で意図を既に持っているため、
// その分岐をユースケース層で明示する。
type UserRepository interface {
	Insert(user *User) error
	Update(user *User) error
	ExistsByEmail(email Email) (bool, error)
}
