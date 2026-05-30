package controller_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kuritaeiji/claude-code-ddd/internal/controller"
	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	"github.com/kuritaeiji/claude-code-ddd/internal/domain/mocks"
	"github.com/kuritaeiji/claude-code-ddd/internal/infrastructure/i18n"
	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/command"
)

// taskHandler は /tasks 系のテストハーネス。
// 実 DB の代わりに mocks の Task/User リポジトリで実ユースケースを組み立て、i18n ミドルウェアでラップする。
func taskHandler(t *testing.T, taskRepo *mocks.TaskRepository, userRepo *mocks.UserRepository) http.Handler {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "locales")
	bundle, err := i18n.BuildBundle(dir)
	require.NoError(t, err)

	create := command.NewCreateTaskCommand(taskRepo, userRepo)
	update := command.NewUpdateTaskCommand(taskRepo, userRepo)
	del := command.NewDeleteTaskCommand(taskRepo)
	ctrl := controller.NewTaskController(create, update, del)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks", ctrl.HandleCreate)
	mux.HandleFunc("PATCH /tasks/{id}", ctrl.HandleUpdate)
	mux.HandleFunc("DELETE /tasks/{id}", ctrl.HandleDelete)
	return controller.NewI18nMiddleware(bundle)(mux)
}

func doTaskReq(t *testing.T, h http.Handler, method, path, body, acceptLanguage string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if acceptLanguage != "" {
		req.Header.Set("Accept-Language", acceptLanguage)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func existingTask(id domain.TaskID) *domain.Task {
	return domain.ReconstructTask(
		id.String(), "old title", "old description",
		domain.TaskStatusTodo, domain.PriorityLow, nil, nil,
	)
}

func TestTaskController_Create_Success(t *testing.T) {
	t.Parallel()

	taskRepo := mocks.NewTaskRepository()
	userRepo := mocks.NewUserRepository()
	taskRepo.On("Insert", mock.AnythingOfType("*domain.Task")).Return(nil).Once()

	rec := doTaskReq(t, taskHandler(t, taskRepo, userRepo), http.MethodPost, "/tasks",
		`{"title":"Write report","description":"q2","priority":"HIGH"}`, "")

	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp controller.TaskResponse
	decodeBody(t, rec, &resp)
	assert.NotEmpty(t, resp.ID)
	assert.Equal(t, "Write report", resp.Title)
	assert.Equal(t, string(domain.TaskStatusTodo), resp.Status)
	assert.Equal(t, "HIGH", resp.Priority)
	assert.Nil(t, resp.DueDate)
	taskRepo.AssertExpectations(t)
}

func TestTaskController_Create_WithDueDate(t *testing.T) {
	t.Parallel()

	taskRepo := mocks.NewTaskRepository()
	userRepo := mocks.NewUserRepository()
	taskRepo.On("Insert", mock.AnythingOfType("*domain.Task")).Return(nil).Once()

	due := time.Now().Add(72 * time.Hour).UTC().Format("2006-01-02")
	rec := doTaskReq(t, taskHandler(t, taskRepo, userRepo), http.MethodPost, "/tasks",
		`{"title":"x","priority":"LOW","dueDate":"`+due+`"}`, "")

	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp controller.TaskResponse
	decodeBody(t, rec, &resp)
	require.NotNil(t, resp.DueDate)
	assert.Equal(t, due, *resp.DueDate)
}

func TestTaskController_Create_InvalidJSON(t *testing.T) {
	t.Parallel()

	taskRepo := mocks.NewTaskRepository()
	userRepo := mocks.NewUserRepository()

	rec := doTaskReq(t, taskHandler(t, taskRepo, userRepo), http.MethodPost, "/tasks", `not-json`, "")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	taskRepo.AssertNotCalled(t, "Insert")
}

func TestTaskController_Create_InvalidPriority(t *testing.T) {
	t.Parallel()

	taskRepo := mocks.NewTaskRepository()
	userRepo := mocks.NewUserRepository()

	rec := doTaskReq(t, taskHandler(t, taskRepo, userRepo), http.MethodPost, "/tasks",
		`{"title":"x","priority":"URGENT"}`, "ja")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]string
	decodeBody(t, rec, &body)
	assert.Contains(t, body["message"], "優先度")
	taskRepo.AssertNotCalled(t, "Insert")
}

func TestTaskController_Create_InvalidDueDateFormat(t *testing.T) {
	t.Parallel()

	taskRepo := mocks.NewTaskRepository()
	userRepo := mocks.NewUserRepository()

	rec := doTaskReq(t, taskHandler(t, taskRepo, userRepo), http.MethodPost, "/tasks",
		`{"title":"x","priority":"LOW","dueDate":"2026/01/01"}`, "")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	taskRepo.AssertNotCalled(t, "Insert")
}

func TestTaskController_Create_PastDueDate(t *testing.T) {
	t.Parallel()

	taskRepo := mocks.NewTaskRepository()
	userRepo := mocks.NewUserRepository()

	past := time.Now().Add(-72 * time.Hour).UTC().Format("2006-01-02")
	rec := doTaskReq(t, taskHandler(t, taskRepo, userRepo), http.MethodPost, "/tasks",
		`{"title":"x","priority":"LOW","dueDate":"`+past+`"}`, "")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	taskRepo.AssertNotCalled(t, "Insert")
}

func TestTaskController_Update_Success(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	taskRepo := mocks.NewTaskRepository()
	userRepo := mocks.NewUserRepository()
	taskRepo.On("FindByID", id).Return(existingTask(id), nil).Once()
	taskRepo.On("Update", mock.AnythingOfType("*domain.Task")).Return(nil).Once()

	rec := doTaskReq(t, taskHandler(t, taskRepo, userRepo), http.MethodPatch, "/tasks/"+id.String(),
		`{"title":"updated","status":"DONE","priority":"CRITICAL"}`, "")

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp controller.TaskResponse
	decodeBody(t, rec, &resp)
	assert.Equal(t, id.String(), resp.ID)
	assert.Equal(t, "updated", resp.Title)
	assert.Equal(t, "DONE", resp.Status)
	taskRepo.AssertExpectations(t)
}

func TestTaskController_Update_InvalidID(t *testing.T) {
	t.Parallel()

	taskRepo := mocks.NewTaskRepository()
	userRepo := mocks.NewUserRepository()

	rec := doTaskReq(t, taskHandler(t, taskRepo, userRepo), http.MethodPatch, "/tasks/not-a-uuid",
		`{"title":"x","status":"TODO","priority":"LOW"}`, "")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	taskRepo.AssertNotCalled(t, "FindByID")
}

func TestTaskController_Update_NotFound(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	taskRepo := mocks.NewTaskRepository()
	userRepo := mocks.NewUserRepository()
	taskRepo.On("FindByID", id).Return(nil, domain.ErrTaskNotFound).Once()

	rec := doTaskReq(t, taskHandler(t, taskRepo, userRepo), http.MethodPatch, "/tasks/"+id.String(),
		`{"title":"x","status":"TODO","priority":"LOW"}`, "")

	assert.Equal(t, http.StatusNotFound, rec.Code)
	taskRepo.AssertNotCalled(t, "Update")
}

func TestTaskController_Delete_Success(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	taskRepo := mocks.NewTaskRepository()
	userRepo := mocks.NewUserRepository()
	taskRepo.On("FindByID", id).Return(existingTask(id), nil).Once()
	taskRepo.On("Delete", id).Return(nil).Once()

	rec := doTaskReq(t, taskHandler(t, taskRepo, userRepo), http.MethodDelete, "/tasks/"+id.String(), "", "")

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
	taskRepo.AssertExpectations(t)
}

func TestTaskController_Delete_NotFound(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	taskRepo := mocks.NewTaskRepository()
	userRepo := mocks.NewUserRepository()
	taskRepo.On("FindByID", id).Return(nil, domain.ErrTaskNotFound).Once()

	rec := doTaskReq(t, taskHandler(t, taskRepo, userRepo), http.MethodDelete, "/tasks/"+id.String(), "", "")

	assert.Equal(t, http.StatusNotFound, rec.Code)
	taskRepo.AssertNotCalled(t, "Delete")
}
