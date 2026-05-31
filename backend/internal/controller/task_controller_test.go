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
	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/query"
	querymocks "github.com/kuritaeiji/claude-code-ddd/internal/usecase/query/mocks"
)

// localesDir はテストファイルの位置から locales ディレクトリの絶対パスを返す。
func localesDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "locales")
}

// taskHandler は /tasks 系コマンドのテストハーネス。
// 実 DB の代わりに mocks の Task/User リポジトリで実ユースケースを組み立て、i18n ミドルウェアでラップする。
func taskHandler(t *testing.T, taskRepo *mocks.TaskRepository, userRepo *mocks.UserRepository) http.Handler {
	t.Helper()
	bundle, err := i18n.BuildBundle(localesDir(t))
	require.NoError(t, err)

	create := command.NewCreateTaskCommand(taskRepo, userRepo)
	update := command.NewUpdateTaskCommand(taskRepo, userRepo)
	del := command.NewDeleteTaskCommand(taskRepo)
	queryRepo := querymocks.NewTaskQueryRepository()
	listTasks := query.NewListTasksQuery(queryRepo)
	getTask := query.NewGetTaskQuery(queryRepo)
	ctrl := controller.NewTaskController(create, update, del, listTasks, getTask)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks", ctrl.HandleCreate)
	mux.HandleFunc("PATCH /tasks/{id}", ctrl.HandleUpdate)
	mux.HandleFunc("DELETE /tasks/{id}", ctrl.HandleDelete)
	return controller.NewI18nMiddleware(bundle)(mux)
}

// taskQueryHandler は GET /tasks 系クエリのテストハーネス。
// クエリリポジトリのみモックし、コマンド側は空のモックで構築する。
func taskQueryHandler(t *testing.T, queryRepo *querymocks.TaskQueryRepository) http.Handler {
	t.Helper()
	bundle, err := i18n.BuildBundle(localesDir(t))
	require.NoError(t, err)

	taskRepo := mocks.NewTaskRepository()
	userRepo := mocks.NewUserRepository()
	create := command.NewCreateTaskCommand(taskRepo, userRepo)
	update := command.NewUpdateTaskCommand(taskRepo, userRepo)
	del := command.NewDeleteTaskCommand(taskRepo)
	listTasks := query.NewListTasksQuery(queryRepo)
	getTask := query.NewGetTaskQuery(queryRepo)
	ctrl := controller.NewTaskController(create, update, del, listTasks, getTask)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /tasks", ctrl.HandleList)
	mux.HandleFunc("GET /tasks/{id}", ctrl.HandleGet)
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

// --- Q-01 HandleList ---

func doGetReq(t *testing.T, h http.Handler, path, acceptLanguage string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if acceptLanguage != "" {
		req.Header.Set("Accept-Language", acceptLanguage)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func sampleListItems() []query.TaskListItemDTO {
	due := time.Now().Add(48 * time.Hour)
	return []query.TaskListItemDTO{
		{ID: "id-1", Title: "Task A", Status: "TODO", Priority: "HIGH", DueDate: &due, AssigneeNames: []string{"Alice"}},
		{ID: "id-2", Title: "Task B", Status: "IN_PROGRESS", Priority: "LOW", AssigneeNames: []string{}},
	}
}

func TestTaskController_List_NoFilter(t *testing.T) {
	t.Parallel()

	queryRepo := querymocks.NewTaskQueryRepository()
	params := query.ListTasksParams{}
	queryRepo.On("ListTasks", params).Return(sampleListItems(), nil).Once()

	rec := doGetReq(t, taskQueryHandler(t, queryRepo), "/tasks", "")

	assert.Equal(t, http.StatusOK, rec.Code)
	var items []controller.TaskListItemResponse
	decodeBody(t, rec, &items)
	assert.Len(t, items, 2)
	assert.Equal(t, "id-1", items[0].ID)
	assert.Equal(t, []string{"Alice"}, items[0].AssigneeNames)
	queryRepo.AssertExpectations(t)
}

func TestTaskController_List_FilterBySingleStatus(t *testing.T) {
	t.Parallel()

	queryRepo := querymocks.NewTaskQueryRepository()
	params := query.ListTasksParams{Statuses: []string{"TODO"}}
	queryRepo.On("ListTasks", params).Return(sampleListItems()[:1], nil).Once()

	rec := doGetReq(t, taskQueryHandler(t, queryRepo), "/tasks?status=TODO", "")

	assert.Equal(t, http.StatusOK, rec.Code)
	var items []controller.TaskListItemResponse
	decodeBody(t, rec, &items)
	assert.Len(t, items, 1)
	queryRepo.AssertExpectations(t)
}

func TestTaskController_List_FilterByMultipleStatuses(t *testing.T) {
	t.Parallel()

	queryRepo := querymocks.NewTaskQueryRepository()
	params := query.ListTasksParams{Statuses: []string{"TODO", "IN_PROGRESS"}}
	queryRepo.On("ListTasks", params).Return(sampleListItems(), nil).Once()

	rec := doGetReq(t, taskQueryHandler(t, queryRepo), "/tasks?status=TODO&status=IN_PROGRESS", "")

	assert.Equal(t, http.StatusOK, rec.Code)
	var items []controller.TaskListItemResponse
	decodeBody(t, rec, &items)
	assert.Len(t, items, 2)
	queryRepo.AssertExpectations(t)
}

func TestTaskController_List_FilterByMultipleStatusesAndPriorities(t *testing.T) {
	t.Parallel()

	queryRepo := querymocks.NewTaskQueryRepository()
	params := query.ListTasksParams{
		Statuses:   []string{"TODO", "DONE"},
		Priorities: []string{"HIGH", "CRITICAL"},
	}
	queryRepo.On("ListTasks", params).Return(sampleListItems()[:1], nil).Once()

	rec := doGetReq(t, taskQueryHandler(t, queryRepo),
		"/tasks?status=TODO&status=DONE&priority=HIGH&priority=CRITICAL", "")

	assert.Equal(t, http.StatusOK, rec.Code)
	var items []controller.TaskListItemResponse
	decodeBody(t, rec, &items)
	assert.Len(t, items, 1)
	queryRepo.AssertExpectations(t)
}

func TestTaskController_List_InvalidStatus(t *testing.T) {
	t.Parallel()

	queryRepo := querymocks.NewTaskQueryRepository()

	rec := doGetReq(t, taskQueryHandler(t, queryRepo), "/tasks?status=INVALID", "ja")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]string
	decodeBody(t, rec, &body)
	assert.Contains(t, body["message"], "ステータス")
	queryRepo.AssertNotCalled(t, "ListTasks")
}

func TestTaskController_List_InvalidPriority(t *testing.T) {
	t.Parallel()

	queryRepo := querymocks.NewTaskQueryRepository()

	rec := doGetReq(t, taskQueryHandler(t, queryRepo), "/tasks?priority=INVALID", "ja")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]string
	decodeBody(t, rec, &body)
	assert.Contains(t, body["message"], "優先度")
	queryRepo.AssertNotCalled(t, "ListTasks")
}

func TestTaskController_List_EmptyResult(t *testing.T) {
	t.Parallel()

	queryRepo := querymocks.NewTaskQueryRepository()
	params := query.ListTasksParams{}
	queryRepo.On("ListTasks", params).Return([]query.TaskListItemDTO{}, nil).Once()

	rec := doGetReq(t, taskQueryHandler(t, queryRepo), "/tasks", "")

	assert.Equal(t, http.StatusOK, rec.Code)
	var items []controller.TaskListItemResponse
	decodeBody(t, rec, &items)
	assert.Empty(t, items)
}

// --- Q-02 HandleGet ---

func TestTaskController_Get_Success(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	due := time.Now().Add(48 * time.Hour)
	dto := &query.TaskDetailDTO{
		ID:            id.String(),
		Title:         "My Task",
		Description:   "details",
		Status:        "TODO",
		Priority:      "HIGH",
		DueDate:       &due,
		AssigneeNames: []string{"Bob"},
	}

	queryRepo := querymocks.NewTaskQueryRepository()
	queryRepo.On("GetTaskByID", id.String()).Return(dto, nil).Once()

	rec := doGetReq(t, taskQueryHandler(t, queryRepo), "/tasks/"+id.String(), "")

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp controller.TaskDetailResponse
	decodeBody(t, rec, &resp)
	assert.Equal(t, id.String(), resp.ID)
	assert.Equal(t, "My Task", resp.Title)
	assert.Equal(t, "details", resp.Description)
	assert.Equal(t, []string{"Bob"}, resp.AssigneeNames)
	require.NotNil(t, resp.DueDate)
	queryRepo.AssertExpectations(t)
}

func TestTaskController_Get_InvalidID(t *testing.T) {
	t.Parallel()

	queryRepo := querymocks.NewTaskQueryRepository()

	rec := doGetReq(t, taskQueryHandler(t, queryRepo), "/tasks/not-a-uuid", "")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	queryRepo.AssertNotCalled(t, "GetTaskByID")
}

func TestTaskController_Get_NotFound(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	queryRepo := querymocks.NewTaskQueryRepository()
	queryRepo.On("GetTaskByID", id.String()).Return(nil, domain.ErrTaskNotFound).Once()

	rec := doGetReq(t, taskQueryHandler(t, queryRepo), "/tasks/"+id.String(), "ja")

	assert.Equal(t, http.StatusNotFound, rec.Code)
	var body map[string]string
	decodeBody(t, rec, &body)
	assert.Contains(t, body["message"], "見つかりません")
}
