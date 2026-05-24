package controller

import (
	"encoding/json"
	"net/http"

	"github.com/kuritaeiji/claude-code-ddd/internal/infrastructure/i18n"
	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/command"
)

type UserController struct {
	register *command.RegisterUserCommand
}

func NewUserController(register *command.RegisterUserCommand) *UserController {
	return &UserController{register: register}
}

type RegisterUserRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

type RegisterUserResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Status      string `json:"status"`
}

// HandleRegister は POST /users を処理する。
// JSON デコード失敗時は 400、Execute から返るドメインエラーは writeError でステータスを決定する。
func (c *UserController) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		translator := i18n.TranslatorFrom(r.Context())
		writeJSON(w, http.StatusBadRequest, errorResponse{Message: translator.Translate("common.invalid_request_body", nil)})
		return
	}

	dto, err := c.register.Execute(command.RegisterUserParams{
		Email:       req.Email,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, RegisterUserResponse{
		ID:          dto.ID,
		Email:       dto.Email,
		DisplayName: dto.DisplayName,
		Status:      dto.Status,
	})
}
