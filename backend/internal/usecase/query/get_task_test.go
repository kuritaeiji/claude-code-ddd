package query_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/query"
	querymocks "github.com/kuritaeiji/claude-code-ddd/internal/usecase/query/mocks"
	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

func sampleTaskDetail(id string) *query.TaskDetailDTO {
	due := time.Now().Add(48 * time.Hour)
	return &query.TaskDetailDTO{
		ID:            id,
		Title:         "Sample Task",
		Description:   "detail description",
		Status:        "TODO",
		Priority:      "HIGH",
		DueDate:       &due,
		AssigneeNames: []string{"Alice"},
	}
}

func TestGetTaskQuery_Success(t *testing.T) {
	t.Parallel()

	id := uuid.NewString()
	repo := querymocks.NewTaskQueryRepository()
	repo.On("GetTaskByID", id).Return(sampleTaskDetail(id), nil).Once()

	q := query.NewGetTaskQuery(repo)
	dto, err := q.Execute(query.GetTaskParams{ID: id})

	require.NoError(t, err)
	assert.Equal(t, id, dto.ID)
	assert.Equal(t, "Sample Task", dto.Title)
	assert.Equal(t, "detail description", dto.Description)
	assert.Equal(t, []string{"Alice"}, dto.AssigneeNames)
	repo.AssertExpectations(t)
}

func TestGetTaskQuery_InvalidID(t *testing.T) {
	t.Parallel()

	repo := querymocks.NewTaskQueryRepository()
	q := query.NewGetTaskQuery(repo)
	_, err := q.Execute(query.GetTaskParams{ID: "not-a-uuid"})

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	repo.AssertNotCalled(t, "GetTaskByID")
}

func TestGetTaskQuery_NotFound(t *testing.T) {
	t.Parallel()

	id := uuid.NewString()
	repo := querymocks.NewTaskQueryRepository()
	repo.On("GetTaskByID", id).Return(nil, domain.ErrTaskNotFound).Once()

	q := query.NewGetTaskQuery(repo)
	_, err := q.Execute(query.GetTaskParams{ID: id})

	var nfe *apperrors.NotFoundError
	require.ErrorAs(t, err, &nfe)
}

func TestGetTaskQuery_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	id := uuid.NewString()
	repo := querymocks.NewTaskQueryRepository()
	repoErr := errors.New("db down")
	repo.On("GetTaskByID", id).Return(nil, repoErr).Once()

	q := query.NewGetTaskQuery(repo)
	_, err := q.Execute(query.GetTaskParams{ID: id})

	require.ErrorIs(t, err, repoErr)
}
