// Package mocks は Domain 層のインターフェースに対する testify/mock ベースのモック実装を提供する。
// テストコードから import して使うことを目的とし、プロダクションコードからの参照は禁止する。
package mocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
)

// TaskRepository インターフェースの実装漏れをビルド時に検出するためのアサーション。
var _ domain.TaskRepository = (*TaskRepository)(nil)

type TaskRepository struct {
	mock.Mock
}

func NewTaskRepository() *TaskRepository {
	return &TaskRepository{}
}

func (m *TaskRepository) Insert(task *domain.Task) error {
	args := m.Called(task)
	return args.Error(0)
}

func (m *TaskRepository) Update(task *domain.Task) error {
	args := m.Called(task)
	return args.Error(0)
}

func (m *TaskRepository) Delete(id domain.TaskID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *TaskRepository) FindByID(id domain.TaskID) (*domain.Task, error) {
	args := m.Called(id)
	task, _ := args.Get(0).(*domain.Task)
	return task, args.Error(1)
}

func (m *TaskRepository) FindByAssigneeID(userID domain.UserID) ([]*domain.Task, error) {
	args := m.Called(userID)
	tasks, _ := args.Get(0).([]*domain.Task)
	return tasks, args.Error(1)
}
