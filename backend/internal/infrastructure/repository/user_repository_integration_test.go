package repository

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	"github.com/kuritaeiji/claude-code-ddd/internal/infrastructure/config"
)

// newTestDB は PostgreSQL に接続し、全テーブルを作成・クリーンしてから *gorm.DB を返す。
// 各テストの先頭で必ず TRUNCATE するため、同一プロセス内のテスト実行順序に依存しない。
// t.Parallel() を使わない前提（同じ DB を共有するため TRUNCATE が並列テストを潰すリスクがある）。
// task_assignees は tasks・users への FK を持つため、CASCADE 付きで一括 TRUNCATE する。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// UNIQUE 違反テストでは gorm のデフォルト Logger が ERROR を吐いてテスト出力を汚すため、
	// テスト中は Silent に切り替える。ロガーの動作確認自体はテスト対象外。
	db, err := gorm.Open(postgres.Open(testDSN(t)), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err, "PostgreSQL connection failed. Start the DB and run tests via `make up && make test` (which loads .env.local). To target another DB, set TEST_DATABASE_URL.")
	require.NoError(t, MigrateSchema(db))
	require.NoError(t, db.Exec("TRUNCATE TABLE task_assignees, tasks, users CASCADE").Error)
	return db
}

// testDSN は TEST_DATABASE_URL を最優先で読み、未設定なら config.Load() で組み立てた DSN を返す。
// 環境変数は .env.local が source of truth で、`make test` 経由で export される。
func testDSN(t *testing.T) string {
	t.Helper()
	if dsn := os.Getenv("TEST_DATABASE_URL"); dsn != "" {
		return dsn
	}
	cfg, err := config.Load()
	require.NoError(t, err)
	return cfg.DB.DSN()
}

// newActiveUser は ReconstructUser ベースでテスト用 *domain.User を組み立てる。
// UserRegistrar.Register を経由すると内部で repo.Insert を呼んでしまい
// Repository 単独の挙動確認にならないため、ここでは復元経路で人工的に作る。
func newActiveUser(t *testing.T, emailValue, displayName string) *domain.User {
	t.Helper()
	email, err := domain.NewEmail(emailValue)
	require.NoError(t, err)
	return domain.ReconstructUser(domain.GenerateUserID().String(), email.String(), displayName, domain.UserStatusActive)
}

func TestIntegration_UserRepository_Insert(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)

	user := newActiveUser(t, "alice@example.com", "Alice")
	require.NoError(t, repo.Insert(user))

	var m userModel
	require.NoError(t, db.First(&m, "email = ?", "alice@example.com").Error)
	assert.Equal(t, user.ID().String(), m.ID)
	assert.Equal(t, "alice@example.com", m.Email)
	assert.Equal(t, "Alice", m.DisplayName)
	assert.Equal(t, string(domain.UserStatusActive), m.Status)
	assert.False(t, m.CreatedAt.IsZero(), "CreatedAt must be set by gorm")
	assert.False(t, m.UpdatedAt.IsZero(), "UpdatedAt must be set by gorm")
}

// 同一 email の 2 回目 Insert で UNIQUE 制約に当たり、Repository が
// domain.ErrEmailDuplicate に翻訳して返すことを検証する。
// BR-U-01 のレース時最終防衛線として最も重要な経路。
func TestIntegration_UserRepository_Insert_DuplicateEmail(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)

	require.NoError(t, repo.Insert(newActiveUser(t, "dup@example.com", "Alice")))

	err := repo.Insert(newActiveUser(t, "dup@example.com", "Bob"))
	require.ErrorIs(t, err, domain.ErrEmailDuplicate)
}

func TestIntegration_UserRepository_FindByID(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)

	user := newActiveUser(t, "find@example.com", "Alice")
	require.NoError(t, repo.Insert(user))

	got, err := repo.FindByID(user.ID())
	require.NoError(t, err)
	assert.True(t, user.ID().Equals(got.ID()))
	assert.Equal(t, "find@example.com", got.Email().String())
	assert.Equal(t, "Alice", got.DisplayName())
	assert.Equal(t, domain.UserStatusActive, got.Status())
}

func TestIntegration_UserRepository_FindByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)

	_, err := repo.FindByID(domain.GenerateUserID())
	require.ErrorIs(t, err, domain.ErrUserNotFound)
}

// Update による非活性化（status 列の書き換え）が永続化されることを検証する。
// U-02 のユースケースが FindByID → Deactivate → Update で状態遷移を保存する経路の最終防衛線。
func TestIntegration_UserRepository_Update_Deactivate(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)

	user := newActiveUser(t, "deact@example.com", "Alice")
	require.NoError(t, repo.Insert(user))

	require.NoError(t, user.Deactivate())
	require.NoError(t, repo.Update(user))

	got, err := repo.FindByID(user.ID())
	require.NoError(t, err)
	assert.Equal(t, domain.UserStatusInactive, got.Status())
}
