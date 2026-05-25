package controller_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kuritaeiji/claude-code-ddd/internal/controller"
	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	"github.com/kuritaeiji/claude-code-ddd/internal/domain/mocks"
	"github.com/kuritaeiji/claude-code-ddd/internal/infrastructure/event"
	"github.com/kuritaeiji/claude-code-ddd/internal/infrastructure/i18n"
	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/command"
)

// repoBackedHandler は POST /users 用のテストハーネス。
// 実 DB の代わりに mocks.UserRepository を使って Registrar / Command / Controller を組み立て、
// i18n ミドルウェアでラップした http.Handler を返す。
// locales ディレクトリはテスト実行 cwd に依存しないよう、このテストファイルからの相対で解決する。
func repoBackedHandler(t *testing.T, repo *mocks.UserRepository) http.Handler {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "locales")
	bundle, err := i18n.BuildBundle(dir)
	require.NoError(t, err)

	registrar := domain.NewUserRegistrar(repo)
	registerCmd := command.NewRegisterUserCommand(registrar)
	// ハンドラ未登録の実 EventBus を使う。UserDeactivated は publish されるが購読者がいないため no-op。
	deactivateCmd := command.NewDeactivateUserCommand(repo, event.NewInMemoryBus())
	ctrl := controller.NewUserController(registerCmd, deactivateCmd)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /users", ctrl.HandleRegister)
	mux.HandleFunc("PATCH /users/{id}/deactivate", ctrl.HandleDeactivate)
	return controller.NewI18nMiddleware(bundle)(mux)
}

func doPost(t *testing.T, h http.Handler, body string, acceptLanguage string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if acceptLanguage != "" {
		req.Header.Set("Accept-Language", acceptLanguage)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func doPatch(t *testing.T, h http.Handler, path string, acceptLanguage string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, path, nil)
	if acceptLanguage != "" {
		req.Header.Set("Accept-Language", acceptLanguage)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder, out any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(rec.Body).Decode(out))
}

func TestUserController_Register_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewUserRepository()
	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	repo.On("ExistsByEmail", email).Return(false, nil).Once()
	repo.On("Insert", mock.AnythingOfType("*domain.User")).Return(nil).Once()

	rec := doPost(t, repoBackedHandler(t, repo), `{"email":"user@example.com","displayName":"Alice"}`, "")

	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp controller.RegisterUserResponse
	decodeBody(t, rec, &resp)
	assert.Equal(t, "user@example.com", resp.Email)
	assert.Equal(t, "Alice", resp.DisplayName)
	assert.Equal(t, string(domain.UserStatusActive), resp.Status)
	assert.NotEmpty(t, resp.ID)
	repo.AssertExpectations(t)
}

func TestUserController_Register_InvalidJSON(t *testing.T) {
	t.Parallel()

	repo := mocks.NewUserRepository()
	rec := doPost(t, repoBackedHandler(t, repo), `not-json`, "")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	repo.AssertNotCalled(t, "Insert")
}

func TestUserController_Register_InvalidEmail(t *testing.T) {
	t.Parallel()

	repo := mocks.NewUserRepository()
	rec := doPost(t, repoBackedHandler(t, repo), `{"email":"not-an-email","displayName":"Alice"}`, "")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	repo.AssertNotCalled(t, "ExistsByEmail")
}

func TestUserController_Register_DuplicateEmail(t *testing.T) {
	t.Parallel()

	repo := mocks.NewUserRepository()
	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	repo.On("ExistsByEmail", email).Return(true, nil).Once()

	rec := doPost(t, repoBackedHandler(t, repo), `{"email":"user@example.com","displayName":"Alice"}`, "")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]string
	decodeBody(t, rec, &body)
	assert.Contains(t, body["message"], "already registered")
}

func TestUserController_Register_DisplayNameTooLong(t *testing.T) {
	t.Parallel()

	repo := mocks.NewUserRepository()
	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	repo.On("ExistsByEmail", email).Return(false, nil).Once()

	long := strings.Repeat("a", 51)
	rec := doPost(t, repoBackedHandler(t, repo), `{"email":"user@example.com","displayName":"`+long+`"}`, "")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserController_Register_RepositoryError(t *testing.T) {
	t.Parallel()

	repo := mocks.NewUserRepository()
	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	repo.On("ExistsByEmail", email).Return(false, nil).Once()
	repo.On("Insert", mock.AnythingOfType("*domain.User")).Return(errors.New("db down")).Once()

	rec := doPost(t, repoBackedHandler(t, repo), `{"email":"user@example.com","displayName":"Alice"}`, "")

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestUserController_Register_JapaneseAcceptLanguage(t *testing.T) {
	t.Parallel()

	repo := mocks.NewUserRepository()
	rec := doPost(t, repoBackedHandler(t, repo), `{"email":"not-an-email","displayName":"Alice"}`, "ja")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]string
	decodeBody(t, rec, &body)
	assert.Contains(t, body["message"], "メールアドレス")
}

func TestUserController_Deactivate_Success(t *testing.T) {
	t.Parallel()

	id := domain.GenerateUserID()
	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	user := domain.ReconstructUser(id, email, "Alice", domain.UserStatusActive)

	repo := mocks.NewUserRepository()
	repo.On("FindByID", id).Return(user, nil).Once()
	repo.On("Update", mock.AnythingOfType("*domain.User")).Return(nil).Once()

	rec := doPatch(t, repoBackedHandler(t, repo), "/users/"+id.String()+"/deactivate", "")

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp controller.DeactivateUserResponse
	decodeBody(t, rec, &resp)
	assert.Equal(t, id.String(), resp.ID)
	assert.Equal(t, string(domain.UserStatusInactive), resp.Status)
	repo.AssertExpectations(t)
}

func TestUserController_Deactivate_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewUserRepository()
	rec := doPatch(t, repoBackedHandler(t, repo), "/users/not-a-uuid/deactivate", "")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	repo.AssertNotCalled(t, "FindByID")
}

func TestUserController_Deactivate_NotFound(t *testing.T) {
	t.Parallel()

	id := domain.GenerateUserID()
	repo := mocks.NewUserRepository()
	repo.On("FindByID", id).Return(nil, domain.ErrUserNotFound).Once()

	rec := doPatch(t, repoBackedHandler(t, repo), "/users/"+id.String()+"/deactivate", "")

	assert.Equal(t, http.StatusNotFound, rec.Code)
	repo.AssertNotCalled(t, "Update")
}

func TestUserController_Deactivate_AlreadyInactive(t *testing.T) {
	t.Parallel()

	id := domain.GenerateUserID()
	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	inactive := domain.ReconstructUser(id, email, "Alice", domain.UserStatusInactive)

	repo := mocks.NewUserRepository()
	repo.On("FindByID", id).Return(inactive, nil).Once()

	rec := doPatch(t, repoBackedHandler(t, repo), "/users/"+id.String()+"/deactivate", "ja")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]string
	decodeBody(t, rec, &body)
	assert.Contains(t, body["message"], "非活性")
	repo.AssertNotCalled(t, "Update")
}
