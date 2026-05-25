// Package repository は Domain 層で定義されたリポジトリインターフェースの gorm 実装を提供する。
package repository

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
)

// userModel は users テーブルへの gorm マッピング。
// domain.User はフィールドが非公開のため、Repository 境界で平坦化したモデルを介して相互変換する。
type userModel struct {
	ID          string    `gorm:"type:uuid;primaryKey"`
	Email       string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	DisplayName string    `gorm:"type:varchar(50);not null"`
	Status      string    `gorm:"type:varchar(20);not null"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (userModel) TableName() string { return "users" }

func toModel(u *domain.User) *userModel {
	return &userModel{
		ID:          u.ID().String(),
		Email:       u.Email().String(),
		DisplayName: u.DisplayName(),
		Status:      string(u.Status()),
	}
}

// toDomain は users テーブルの 1 行を *domain.User に復元する。
// 業務ルール（BR-U-*）は登録時に検証済みのため、ReconstructUser / RestoreEmail / RestoreUserID
// 経由で値オブジェクトを「無検査復元」する。これは domain パッケージで明示された方針。
func toDomain(m *userModel) *domain.User {
	return domain.ReconstructUser(
		domain.RestoreUserID(m.ID),
		domain.RestoreEmail(m.Email),
		m.DisplayName,
		domain.UserStatus(m.Status),
	)
}

// MigrateSchema は users テーブルのスキーマを最新化する。
// 起動時に cmd/api/main.go から 1 回だけ呼ぶ想定（学習目的のため AutoMigrate を採用）。
func MigrateSchema(db *gorm.DB) error {
	return db.AutoMigrate(&userModel{})
}

type userRepository struct {
	db *gorm.DB
}

// 実装漏れをビルド時に検出するためのアサーション。
var _ domain.UserRepository = (*userRepository)(nil)

func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{db: db}
}

// Insert は新規ユーザーを INSERT する。
// UNIQUE 制約（email）違反はインフラ固有の事象（SQLSTATE 23505）をドメイン語彙の sentinel error
// domain.ErrEmailDuplicate に翻訳して返す。BR-U-01 違反としての業務解釈は UserRegistrar の責務。
func (r *userRepository) Insert(user *domain.User) error {
	if err := r.db.Create(toModel(user)).Error; err != nil {
		if isUniqueViolation(err) {
			return domain.ErrEmailDuplicate
		}
		return err
	}
	return nil
}

// Update は既存ユーザーを UPDATE する（U-02 非活性化の status 書き換え等）。
// PrimaryKey ベースの全カラム更新（gorm Save）。Save は対象行がなくても INSERT してしまうが、
// 呼び出し元のユースケースが事前に FindByID で存在を確認している前提のため、最小実装（Save）にとどめる。
func (r *userRepository) Update(user *domain.User) error {
	return r.db.Save(toModel(user)).Error
}

// FindByID は主キーでユーザーを 1 件取得して復元する。
// gorm の未存在エラー（gorm.ErrRecordNotFound）はドメイン語彙の sentinel
// domain.ErrUserNotFound に翻訳して返す。HTTP 404 への解釈はユースケース層の責務。
func (r *userRepository) FindByID(id domain.UserID) (*domain.User, error) {
	var m userModel
	if err := r.db.First(&m, "id = ?", id.String()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return toDomain(&m), nil
}

func (r *userRepository) ExistsByEmail(email domain.Email) (bool, error) {
	var count int64
	if err := r.db.Model(&userModel{}).Where("email = ?", email.String()).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// isUniqueViolation は PostgreSQL の UNIQUE 制約違反（SQLSTATE 23505）を判定する。
// pgx ドライバ経由のエラーは *pgconn.PgError に unwrap できるため、これを基準にする。
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
