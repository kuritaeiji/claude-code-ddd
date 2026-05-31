// Command api はタスク管理バックエンドのエントリーポイント。
// 依存組み立て（DI）・スキーマ移行・HTTP サーバー起動のみを担い、
// ビジネスロジックは含まない。
package main

import (
	"fmt"
	"log"
	"net/http"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/kuritaeiji/claude-code-ddd/internal/controller"
	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	"github.com/kuritaeiji/claude-code-ddd/internal/infrastructure/config"
	"github.com/kuritaeiji/claude-code-ddd/internal/infrastructure/event"
	"github.com/kuritaeiji/claude-code-ddd/internal/infrastructure/i18n"
	"github.com/kuritaeiji/claude-code-ddd/internal/infrastructure/repository"
	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/command"
	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/query"
	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("startup failed: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := gorm.Open(postgres.Open(cfg.DB.DSN()), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	if err := repository.MigrateSchema(db); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	bundle, err := i18n.BuildBundle(cfg.LocalesDir)
	if err != nil {
		return fmt.Errorf("load locales: %w", err)
	}
	englishTranslator := i18n.NewEnglishTranslator(bundle)
	apperrors.SetDefaultTranslator(englishTranslator)
	i18n.SetFallbackTranslator(englishTranslator)

	eventBus := event.NewInMemoryBus()

	userRepo := repository.NewUserRepository(db)
	registrar := domain.NewUserRegistrar(userRepo)
	registerUser := command.NewRegisterUserCommand(registrar)
	deactivateUser := command.NewDeactivateUserCommand(userRepo, eventBus)
	userCtrl := controller.NewUserController(registerUser, deactivateUser)

	taskRepo := repository.NewTaskRepository(db)
	// UserDeactivated を購読し、非活性化ユーザーを全担当タスクから除外する（domain.md の副作用）。
	eventBus.Subscribe(command.NewRemoveAssigneeOnUserDeactivated(taskRepo))
	createTask := command.NewCreateTaskCommand(taskRepo, userRepo)
	updateTask := command.NewUpdateTaskCommand(taskRepo, userRepo)
	deleteTask := command.NewDeleteTaskCommand(taskRepo)
	taskQueryRepo := repository.NewTaskQueryRepository(db)
	listTasks := query.NewListTasksQuery(taskQueryRepo)
	getTask := query.NewGetTaskQuery(taskQueryRepo)
	taskCtrl := controller.NewTaskController(createTask, updateTask, deleteTask, listTasks, getTask)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /users", userCtrl.HandleRegister)
	mux.HandleFunc("PATCH /users/{id}/deactivate", userCtrl.HandleDeactivate)
	mux.HandleFunc("POST /tasks", taskCtrl.HandleCreate)
	mux.HandleFunc("PATCH /tasks/{id}", taskCtrl.HandleUpdate)
	mux.HandleFunc("DELETE /tasks/{id}", taskCtrl.HandleDelete)
	mux.HandleFunc("GET /tasks", taskCtrl.HandleList)
	mux.HandleFunc("GET /tasks/{id}", taskCtrl.HandleGet)
	handler := controller.NewI18nMiddleware(bundle)(mux)

	addr := ":" + cfg.Port
	log.Printf("listening on %s", addr)
	return http.ListenAndServe(addr, handler)
}
