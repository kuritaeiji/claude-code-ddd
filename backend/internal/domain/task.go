package domain

import (
	"errors"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

// ErrTaskNotFound は TaskRepository.FindByID が対象タスクを見つけられなかった際に返す sentinel error。
// インフラ層は DB 固有の「未存在」を この sentinel に翻訳して返し、
// HTTP 404 への解釈（NotFoundError 化）はユースケース層が行う。
var ErrTaskNotFound = errors.New("task not found")

const (
	titleMinLen       = 1
	titleMaxLen       = 200
	descriptionMaxLen = 5000
)

// --- TaskID ---

type TaskID struct {
	value string
}

func NewTaskID(value string) (TaskID, error) {
	if _, err := uuid.Parse(value); err != nil {
		return TaskID{}, apperrors.NewValidationError("validation.task_id.invalid", nil)
	}
	return TaskID{value: value}, nil
}

func GenerateTaskID() TaskID {
	return TaskID{value: uuid.NewString()}
}

func (t TaskID) String() string           { return t.value }
func (t TaskID) Equals(other TaskID) bool { return t.value == other.value }

// --- TaskStatus ---

type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "TODO"
	TaskStatusInProgress TaskStatus = "IN_PROGRESS"
	TaskStatusDone       TaskStatus = "DONE"
)

func NewTaskStatus(value string) (TaskStatus, error) {
	switch TaskStatus(value) {
	case TaskStatusTodo, TaskStatusInProgress, TaskStatusDone:
		return TaskStatus(value), nil
	default:
		return "", apperrors.NewValidationError("validation.task_status.invalid", nil)
	}
}

// --- Priority ---

type Priority string

const (
	PriorityLow      Priority = "LOW"
	PriorityMedium   Priority = "MEDIUM"
	PriorityHigh     Priority = "HIGH"
	PriorityCritical Priority = "CRITICAL"
)

func NewPriority(value string) (Priority, error) {
	switch Priority(value) {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
		return Priority(value), nil
	default:
		return "", apperrors.NewValidationError("validation.priority.invalid", nil)
	}
}

// --- DueDate ---

type DueDate struct {
	value time.Time
}

// NewDueDate は BR-T-03 に従い、過去日付を拒否する。
// 日付比較は日付成分のみで行う（時刻・タイムゾーンの影響を排除）。
func NewDueDate(value time.Time) (DueDate, error) {
	d := truncateToDate(value)
	today := truncateToDate(time.Now())
	if d.Before(today) {
		return DueDate{}, apperrors.NewValidationError("validation.due_date.past", nil)
	}
	return DueDate{value: d}, nil
}

func (d DueDate) Value() time.Time          { return d.value }
func (d DueDate) String() string            { return d.value.Format("2006-01-02") }
func (d DueDate) Equals(other DueDate) bool { return d.value.Equal(other.value) }

// truncateToDate は入力を UTC に変換してから日付成分（年月日）のみを UTC midnight に正規化する。
// UTC 統一により、タイムゾーンが異なる time.Time 同士の日付比較を一貫させる。
func truncateToDate(t time.Time) time.Time {
	t = t.UTC()
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// --- Task ---

type Task struct {
	id          TaskID
	title       string
	description string
	status      TaskStatus
	priority    Priority
	dueDate     *DueDate
	assigneeIDs []UserID
}

// NewTask は BR-T-01（ステータス=TODO）・BR-T-03〜BR-T-06 を検証してタスクを生成する。
// すべての入力をドメイン層で検証する。
func NewTask(title, description, priority string, dueDate *time.Time, assignees []*User) (*Task, error) {
	if err := validateTitle(title); err != nil {
		return nil, err
	}
	if err := validateDescription(description); err != nil {
		return nil, err
	}
	p, err := NewPriority(priority)
	if err != nil {
		return nil, err
	}
	dd, err := buildDueDate(dueDate)
	if err != nil {
		return nil, err
	}
	ids, err := activeAssigneeIDs(assignees)
	if err != nil {
		return nil, err
	}
	return &Task{
		id:          GenerateTaskID(),
		title:       title,
		description: description,
		status:      TaskStatusTodo, // BR-T-01
		priority:    p,
		dueDate:     dd,
		assigneeIDs: ids,
	}, nil
}

// ReconstructTask はリポジトリからの復元専用ファクトリ。
// DB のプリミティブ値を直接受け取り、値オブジェクトを無検査で組み立てる（検証専用の公開ファクトリを増やさないため）。
// 業務ルールの再検査を行わない（過去レコードの取得不能回避のため）。dueDate は日付成分のみに正規化する。
func ReconstructTask(id, title, description string, status TaskStatus, priority Priority, dueDate *time.Time, assigneeIDs []string) *Task {
	var dd *DueDate
	if dueDate != nil {
		d := DueDate{value: truncateToDate(*dueDate)}
		dd = &d
	}
	ids := make([]UserID, len(assigneeIDs))
	for i, a := range assigneeIDs {
		ids[i] = UserID{value: a}
	}
	return &Task{
		id:          TaskID{value: id},
		title:       title,
		description: description,
		status:      status,
		priority:    priority,
		dueDate:     dd,
		assigneeIDs: ids,
	}
}

func (t *Task) ID() TaskID          { return t.id }
func (t *Task) Title() string       { return t.title }
func (t *Task) Description() string { return t.description }
func (t *Task) Status() TaskStatus  { return t.status }
func (t *Task) Priority() Priority  { return t.priority }
func (t *Task) DueDate() *DueDate   { return t.dueDate }

func (t *Task) AssigneeIDs() []UserID {
	ids := make([]UserID, len(t.assigneeIDs))
	copy(ids, t.assigneeIDs)
	return ids
}

// Update は T-02 に対応し、すべてのフィールドを一括で更新する。
// すべての入力をドメイン層で検証してからまとめて適用するため、検証失敗時はフィールドが変更されない。
// status は BR-T-02 に従い遷移順序の制約を持たない。
func (t *Task) Update(title, description, status, priority string, dueDate *time.Time, assignees []*User) error {
	if err := validateTitle(title); err != nil {
		return err
	}
	if err := validateDescription(description); err != nil {
		return err
	}
	s, err := NewTaskStatus(status)
	if err != nil {
		return err
	}
	p, err := NewPriority(priority)
	if err != nil {
		return err
	}
	dd, err := buildDueDate(dueDate)
	if err != nil {
		return err
	}
	ids, err := activeAssigneeIDs(assignees)
	if err != nil {
		return err
	}
	t.title = title
	t.description = description
	t.status = s
	t.priority = p
	t.dueDate = dd
	t.assigneeIDs = ids
	return nil
}

// RemoveAssignee は UserDeactivated 副作用で呼ばれ、担当者ID一覧から指定ユーザーを除外する。
// 非活性化済みユーザーの除外が目的のため、活性チェックは行わない。
func (t *Task) RemoveAssignee(userID UserID) {
	remaining := make([]UserID, 0, len(t.assigneeIDs))
	for _, id := range t.assigneeIDs {
		if !id.Equals(userID) {
			remaining = append(remaining, id)
		}
	}
	t.assigneeIDs = remaining
}

// --- TaskRepository ---

// TaskRepository はタスクの永続化・読み込みを担うリポジトリインターフェース。
// 新規作成と更新を別メソッドに分け、ユースケースの意図を明示する。
type TaskRepository interface {
	Insert(task *Task) error
	Update(task *Task) error
	Delete(id TaskID) error
	// FindByID は対象タスクを復元して返す。未存在時は ErrTaskNotFound を返す。
	FindByID(id TaskID) (*Task, error)
	// FindByAssigneeID は指定ユーザーが担当するすべてのタスクを返す。UserDeactivated 副作用で使用する。
	FindByAssigneeID(userID UserID) ([]*Task, error)
}

// --- helpers ---

func validateTitle(value string) error {
	length := utf8.RuneCountInString(value)
	if length < titleMinLen || length > titleMaxLen {
		return apperrors.NewValidationError("validation.title.length", map[string]any{
			"Min": titleMinLen,
			"Max": titleMaxLen,
		})
	}
	return nil
}

func validateDescription(value string) error {
	if utf8.RuneCountInString(value) > descriptionMaxLen {
		return apperrors.NewValidationError("validation.description.length", map[string]any{
			"Max": descriptionMaxLen,
		})
	}
	return nil
}

// activeAssigneeIDs は []*User を受け取り、全員が ACTIVE であることを検証して []UserID を返す。
// BR-T-04 の検証ロジックを NewTask・Update で共用する。
func activeAssigneeIDs(assignees []*User) ([]UserID, error) {
	ids := make([]UserID, 0, len(assignees))
	for _, u := range assignees {
		if !u.Status().IsActive() {
			return nil, apperrors.NewValidationError("validation.assignee.inactive", nil)
		}
		ids = append(ids, u.ID())
	}
	return ids, nil
}

// buildDueDate は *time.Time から *DueDate を構築する。nil の場合は nil を返す。
// NewTask・Update の dueDate 引数処理を共通化する。
func buildDueDate(t *time.Time) (*DueDate, error) {
	if t == nil {
		return nil, nil
	}
	d, err := NewDueDate(*t)
	if err != nil {
		return nil, err
	}
	return &d, nil
}
