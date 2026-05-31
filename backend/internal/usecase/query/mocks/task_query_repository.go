// Package mocks は usecase/query 層のインターフェースに対する testify/mock ベースのモック実装を提供する。
// テストコードから import して使うことを目的とし、プロダクションコードからの参照は禁止する。
package mocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/query"
)

var _ query.TaskQueryRepository = (*TaskQueryRepository)(nil)

type TaskQueryRepository struct {
	mock.Mock
}

func NewTaskQueryRepository() *TaskQueryRepository {
	return &TaskQueryRepository{}
}

func (m *TaskQueryRepository) ListTasks(params query.ListTasksParams) ([]query.TaskListItemDTO, error) {
	args := m.Called(params)
	dtos, _ := args.Get(0).([]query.TaskListItemDTO)
	return dtos, args.Error(1)
}

func (m *TaskQueryRepository) GetTaskByID(id string) (*query.TaskDetailDTO, error) {
	args := m.Called(id)
	dto, _ := args.Get(0).(*query.TaskDetailDTO)
	return dto, args.Error(1)
}
