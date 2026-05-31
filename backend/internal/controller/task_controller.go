package controller

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/kuritaeiji/claude-code-ddd/internal/infrastructure/i18n"
	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/command"
	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/query"
	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

// dueDateLayout はリクエスト/レスポンスで期日を表すワイヤーフォーマット（schema.md の DATE に対応）。
const dueDateLayout = "2006-01-02"

type TaskController struct {
	create    *command.CreateTaskCommand
	update    *command.UpdateTaskCommand
	del       *command.DeleteTaskCommand
	listTasks *query.ListTasksQuery
	getTask   *query.GetTaskQuery
}

func NewTaskController(
	create *command.CreateTaskCommand,
	update *command.UpdateTaskCommand,
	del *command.DeleteTaskCommand,
	listTasks *query.ListTasksQuery,
	getTask *query.GetTaskQuery,
) *TaskController {
	return &TaskController{create: create, update: update, del: del, listTasks: listTasks, getTask: getTask}
}

type CreateTaskRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"`
	DueDate     *string  `json:"dueDate"`
	AssigneeIDs []string `json:"assigneeIds"`
}

type UpdateTaskRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	DueDate     *string  `json:"dueDate"`
	AssigneeIDs []string `json:"assigneeIds"`
}

type TaskResponse struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	DueDate     *string  `json:"dueDate"`
	AssigneeIDs []string `json:"assigneeIds"`
}

type TaskListItemResponse struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Status        string   `json:"status"`
	Priority      string   `json:"priority"`
	DueDate       *string  `json:"dueDate"`
	AssigneeNames []string `json:"assigneeNames"`
}

type TaskDetailResponse struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Status        string   `json:"status"`
	Priority      string   `json:"priority"`
	DueDate       *string  `json:"dueDate"`
	AssigneeNames []string `json:"assigneeNames"`
}

// HandleCreate は POST /tasks を処理する。
// JSON デコード失敗時は 400、Execute から返るドメインエラーは writeError でステータスを決定する。
func (c *TaskController) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		translator := i18n.TranslatorFrom(r.Context())
		writeJSON(w, http.StatusBadRequest, errorResponse{Message: translator.Translate("common.invalid_request_body", nil)})
		return
	}

	dueDate, err := parseDueDate(req.DueDate)
	if err != nil {
		writeError(w, r, err)
		return
	}

	dto, err := c.create.Execute(command.CreateTaskParams{
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		DueDate:     dueDate,
		AssigneeIDs: req.AssigneeIDs,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, toTaskResponse(dto))
}

// HandleUpdate は PATCH /tasks/{id} を処理する。
// パスパラメータ id をそのまま Usecase に渡し、UUID 検証・未存在判定はドメイン／ユースケース層に委ねる。
func (c *TaskController) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		translator := i18n.TranslatorFrom(r.Context())
		writeJSON(w, http.StatusBadRequest, errorResponse{Message: translator.Translate("common.invalid_request_body", nil)})
		return
	}

	dueDate, err := parseDueDate(req.DueDate)
	if err != nil {
		writeError(w, r, err)
		return
	}

	dto, err := c.update.Execute(command.UpdateTaskParams{
		ID:          r.PathValue("id"),
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		Priority:    req.Priority,
		DueDate:     dueDate,
		AssigneeIDs: req.AssigneeIDs,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toTaskResponse(dto))
}

// HandleDelete は DELETE /tasks/{id} を処理する。成功時は本文なしの 204 を返す。
func (c *TaskController) HandleDelete(w http.ResponseWriter, r *http.Request) {
	if err := c.del.Execute(command.DeleteTaskParams{ID: r.PathValue("id")}); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleList は GET /tasks を処理する。
// 繰り返しクエリパラメータ status・priority で複数値フィルタできる（例: ?status=TODO&status=DONE）。
// 同一パラメータの複数値は OR、status と priority の間は AND。不正な列挙値は 400、DB エラーは 500 を返す。
func (c *TaskController) HandleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := query.ListTasksParams{
		Statuses:   nonEmpty(q["status"]),
		Priorities: nonEmpty(q["priority"]),
	}

	dtos, err := c.listTasks.Execute(params)
	if err != nil {
		writeError(w, r, err)
		return
	}

	items := make([]TaskListItemResponse, len(dtos))
	for i, dto := range dtos {
		items[i] = TaskListItemResponse{
			ID:            dto.ID,
			Title:         dto.Title,
			Status:        dto.Status,
			Priority:      dto.Priority,
			DueDate:       formatDueDate(dto.DueDate),
			AssigneeNames: dto.AssigneeNames,
		}
	}
	writeJSON(w, http.StatusOK, items)
}

// HandleGet は GET /tasks/{id} を処理する。
// 不正 UUID は 400、未存在は 404、DB エラーは 500 を返す。
func (c *TaskController) HandleGet(w http.ResponseWriter, r *http.Request) {
	dto, err := c.getTask.Execute(query.GetTaskParams{ID: r.PathValue("id")})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, TaskDetailResponse{
		ID:            dto.ID,
		Title:         dto.Title,
		Description:   dto.Description,
		Status:        dto.Status,
		Priority:      dto.Priority,
		DueDate:       formatDueDate(dto.DueDate),
		AssigneeNames: dto.AssigneeNames,
	})
}

// nonEmpty は繰り返しクエリパラメータから空文字要素を除いたスライスを返す。
// `?status=` のような空指定をフィルタなし扱いにするため、検証前に除外する。
// 非空要素が無ければ nil を返し、フィルタ未指定（ListTasksParams のゼロ値）と一致させる。
func nonEmpty(values []string) []string {
	var result []string
	for _, v := range values {
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

func toTaskResponse(dto command.TaskDTO) TaskResponse {
	return TaskResponse{
		ID:          dto.ID,
		Title:       dto.Title,
		Description: dto.Description,
		Status:      dto.Status,
		Priority:    dto.Priority,
		DueDate:     formatDueDate(dto.DueDate),
		AssigneeIDs: dto.AssigneeIDs,
	}
}

// formatDueDate は期日 *time.Time をワイヤーフォーマット文字列（YYYY-MM-DD）へ変換する。
// nil（期日なし）は nil を返す。出力側のフォーマットも parseDueDate と同じく dueDateLayout を
// 唯一の基準として Controller 層が担う。
func formatDueDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(dueDateLayout)
	return &s
}

// parseDueDate は dueDate 文字列（YYYY-MM-DD）を *time.Time に変換する。
// 未指定・空文字は nil（期日なし）。ワイヤーフォーマットの変換は Controller の責務とし、
// 形式不正は ValidationError として返す（過去日付不可の業務検証はドメイン層 NewDueDate が担う）。
func parseDueDate(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(dueDateLayout, *s)
	if err != nil {
		return nil, apperrors.NewValidationError("validation.due_date.invalid_format", nil)
	}
	return &t, nil
}
