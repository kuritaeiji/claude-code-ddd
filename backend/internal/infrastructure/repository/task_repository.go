package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
)

// taskModel は tasks テーブルへの gorm マッピング。
// domain.Task はフィールドが非公開のため、Repository 境界で平坦化したモデルを介して相互変換する。
// 担当者一覧（assigneeIDs）は taskAssigneeModel（中間テーブル）で別途表現する。
type taskModel struct {
	ID          string     `gorm:"type:uuid;primaryKey"`
	Title       string     `gorm:"type:varchar(200);not null"`
	Description string     `gorm:"type:text"`
	Status      string     `gorm:"type:varchar(20);not null"`
	Priority    string     `gorm:"type:varchar(20);not null"`
	DueDate     *time.Time `gorm:"type:date"`
	CreatedAt   time.Time  `gorm:"not null"`
	UpdatedAt   time.Time  `gorm:"not null"`
}

func (taskModel) TableName() string { return "tasks" }

// taskAssigneeModel は task_assignees 中間テーブルへの gorm マッピング。
// (task_id, user_id) の複合主キーで Task と User の多対多（ID 参照）を表現する。
// Task/User の関連フィールドは schema.md の FK（task_id→tasks.id・user_id→users.id）を
// AutoMigrate で生成させるための宣言であり、行の読み書きには使わない（Create では Omit する）。
type taskAssigneeModel struct {
	TaskID string    `gorm:"type:uuid;primaryKey"`
	UserID string    `gorm:"type:uuid;primaryKey"`
	Task   taskModel `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
	User   userModel `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (taskAssigneeModel) TableName() string { return "task_assignees" }

func taskToModel(t *domain.Task) *taskModel {
	var dueDate *time.Time
	if d := t.DueDate(); d != nil {
		v := d.Value()
		dueDate = &v
	}
	return &taskModel{
		ID:          t.ID().String(),
		Title:       t.Title(),
		Description: t.Description(),
		Status:      string(t.Status()),
		Priority:    string(t.Priority()),
		DueDate:     dueDate,
	}
}

func assigneeModels(t *domain.Task) []taskAssigneeModel {
	ids := t.AssigneeIDs()
	models := make([]taskAssigneeModel, len(ids))
	for i, id := range ids {
		models[i] = taskAssigneeModel{TaskID: t.ID().String(), UserID: id.String()}
	}
	return models
}

// taskToDomain は tasks の 1 行と対応する task_assignees 行を *domain.Task に復元する。
// 業務ルール（BR-T-*）は作成・更新時に検証済みのため、ReconstructTask に DB のプリミティブ値を
// そのまま渡して「無検査復元」する。これは domain パッケージで明示された方針。
func taskToDomain(m *taskModel, assignees []taskAssigneeModel) *domain.Task {
	ids := make([]string, len(assignees))
	for i, a := range assignees {
		ids[i] = a.UserID
	}
	return domain.ReconstructTask(
		m.ID,
		m.Title,
		m.Description,
		domain.TaskStatus(m.Status),
		domain.Priority(m.Priority),
		m.DueDate,
		ids,
	)
}

type taskRepository struct {
	db *gorm.DB
}

// 実装漏れをビルド時に検出するためのアサーション。
var _ domain.TaskRepository = (*taskRepository)(nil)

func NewTaskRepository(db *gorm.DB) domain.TaskRepository {
	return &taskRepository{db: db}
}

// Insert は新規タスクを tasks に INSERT し、担当者を task_assignees に INSERT する。
// tasks 行と中間テーブル行の整合性を保つため、両方を 1 トランザクションで書き込む。
func (r *taskRepository) Insert(task *domain.Task) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(taskToModel(task)).Error; err != nil {
			return err
		}
		if models := assigneeModels(task); len(models) > 0 {
			// Omit で関連（Task/User）の自動 upsert を抑止し、中間テーブルの行だけを INSERT する。
			if err := tx.Omit("Task", "User").Create(&models).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Update は tasks 行を更新し、task_assignees を新しい担当者一覧で全置換する。
// 担当者の追加・削除（T-02 の更新、UserDeactivated による RemoveAssignee の結果）を
// 差分計算せずに「既存を全削除 → 新規を全挿入」で反映する。整合性のため 1 トランザクションで行う。
func (r *taskRepository) Update(task *domain.Task) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(taskToModel(task)).Error; err != nil {
			return err
		}
		if err := tx.Where("task_id = ?", task.ID().String()).Delete(&taskAssigneeModel{}).Error; err != nil {
			return err
		}
		if models := assigneeModels(task); len(models) > 0 {
			// Omit で関連（Task/User）の自動 upsert を抑止し、中間テーブルの行だけを INSERT する。
			if err := tx.Omit("Task", "User").Create(&models).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Delete はタスクと担当者を削除する。FK 整合のため子（task_assignees）から先に削除する。
func (r *taskRepository) Delete(id domain.TaskID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("task_id = ?", id.String()).Delete(&taskAssigneeModel{}).Error; err != nil {
			return err
		}
		return tx.Delete(&taskModel{}, "id = ?", id.String()).Error
	})
}

// FindByID は主キーでタスクを 1 件取得し、担当者一覧とともに復元する。
// gorm の未存在エラー（gorm.ErrRecordNotFound）はドメイン語彙の sentinel
// domain.ErrTaskNotFound に翻訳して返す。HTTP 404 への解釈はユースケース層の責務。
func (r *taskRepository) FindByID(id domain.TaskID) (*domain.Task, error) {
	var m taskModel
	if err := r.db.First(&m, "id = ?", id.String()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTaskNotFound
		}
		return nil, err
	}
	var assignees []taskAssigneeModel
	if err := r.db.Where("task_id = ?", id.String()).Find(&assignees).Error; err != nil {
		return nil, err
	}
	return taskToDomain(&m, assignees), nil
}

// FindByAssigneeID は指定ユーザーが担当するすべてのタスクを復元して返す。UserDeactivated 副作用で使用する。
// まず task_assignees を user_id で引いて対象 task_id 集合を得て、tasks を IN 取得し、
// 各タスクの担当者一覧を引き直して復元する。担当タスクが 0 件なら nil を返す。
func (r *taskRepository) FindByAssigneeID(userID domain.UserID) ([]*domain.Task, error) {
	var taskIDs []string
	if err := r.db.Model(&taskAssigneeModel{}).
		Where("user_id = ?", userID.String()).
		Pluck("task_id", &taskIDs).Error; err != nil {
		return nil, err
	}
	if len(taskIDs) == 0 {
		return nil, nil
	}

	var models []taskModel
	if err := r.db.Where("id IN ?", taskIDs).Find(&models).Error; err != nil {
		return nil, err
	}

	// 対象タスクの担当者を 1 クエリでまとめて取得し、task_id ごとに振り分ける（N+1 回避）。
	var assignees []taskAssigneeModel
	if err := r.db.Where("task_id IN ?", taskIDs).Find(&assignees).Error; err != nil {
		return nil, err
	}
	byTask := make(map[string][]taskAssigneeModel, len(models))
	for _, a := range assignees {
		byTask[a.TaskID] = append(byTask[a.TaskID], a)
	}

	tasks := make([]*domain.Task, len(models))
	for i := range models {
		tasks[i] = taskToDomain(&models[i], byTask[models[i].ID])
	}
	return tasks, nil
}
