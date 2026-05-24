// Package command は CQRS の書き込み側ユースケースを集約する。
package command

import (
	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
)

type RegisterUserParams struct {
	Email       string
	DisplayName string
}

type RegisterUserDTO struct {
	ID          string
	Email       string
	DisplayName string
	Status      string
}

// RegisterUserCommand は U-01（ユーザー登録）のオーケストレーション。
// 入力検証・一意性確認・永続化はすべて domain.UserRegistrar に委譲し、
// このコマンド自身は「params 受領 → Registrar 呼び出し → DTO 変換」のみを担う。
type RegisterUserCommand struct {
	registrar *domain.UserRegistrar
}

func NewRegisterUserCommand(registrar *domain.UserRegistrar) *RegisterUserCommand {
	return &RegisterUserCommand{registrar: registrar}
}

func (c *RegisterUserCommand) Execute(params RegisterUserParams) (RegisterUserDTO, error) {
	user, err := c.registrar.Register(params.Email, params.DisplayName)
	if err != nil {
		return RegisterUserDTO{}, err
	}
	return RegisterUserDTO{
		ID:          user.ID().String(),
		Email:       user.Email().String(),
		DisplayName: user.DisplayName(),
		Status:      string(user.Status()),
	}, nil
}
