package command

import (
	"errors"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

type DeactivateUserParams struct {
	ID string
}

type DeactivateUserDTO struct {
	ID          string
	Email       string
	DisplayName string
	Status      string
}

// DeactivateUserCommand は U-02（ユーザー非活性化）のオーケストレーション。
// ID 検証・取得・非活性化・永続化・ドメインイベント発行を順に行う。
// 非活性化の業務ルール（BR-U-03）は User.Deactivate に閉じており、本コマンドは検査しない。
type DeactivateUserCommand struct {
	repo domain.UserRepository
	bus  domain.EventBus
}

func NewDeactivateUserCommand(repo domain.UserRepository, bus domain.EventBus) *DeactivateUserCommand {
	return &DeactivateUserCommand{repo: repo, bus: bus}
}

// Execute は対象ユーザーを非活性化し、発行されたドメインイベントを EventBus に publish する。
// 失敗時は以下のいずれかを返す:
//   - ID 形式不正 → ValidationError（NewUserID 由来）
//   - ユーザー未存在 → NotFoundError（repo の ErrUserNotFound を翻訳）
//   - 既に非活性 → ValidationError（User.Deactivate 由来、BR-U-03）
//   - リポジトリ I/O・イベント発行エラー → そのまま伝播
//
// イベントの publish は永続化成功後に行う（インメモリ同期バス）。
func (c *DeactivateUserCommand) Execute(params DeactivateUserParams) (DeactivateUserDTO, error) {
	id, err := domain.NewUserID(params.ID)
	if err != nil {
		return DeactivateUserDTO{}, err
	}

	user, err := c.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return DeactivateUserDTO{}, apperrors.NewNotFoundError("not_found.user", nil)
		}
		return DeactivateUserDTO{}, err
	}

	if err := user.Deactivate(); err != nil {
		return DeactivateUserDTO{}, err
	}

	if err := c.repo.Update(user); err != nil {
		return DeactivateUserDTO{}, err
	}

	for _, event := range user.PullEvents() {
		if err := c.bus.Publish(event); err != nil {
			return DeactivateUserDTO{}, err
		}
	}

	return DeactivateUserDTO{
		ID:          user.ID().String(),
		Email:       user.Email().String(),
		DisplayName: user.DisplayName(),
		Status:      string(user.Status()),
	}, nil
}
