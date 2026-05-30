package command

import (
	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

// resolveAssignees は担当者 ID 文字列を *domain.User の一覧へ解決する。T-01/T-02 のコマンドが共用する。
// BR-T-04 の ACTIVE 検証は Task 集約に委ね、ここでは「形式」「重複」「存在」のみを担保する:
//   - 各 ID を domain.NewUserID で UUID 形式検証する（不正なら ValidationError）
//   - 重複 ID は除去する（task_assignees の複合主キー違反・FindByIDs の件数照合ずれを防ぐ）
//   - FindByIDs で解決し、存在しない ID が 1 件でもあれば ValidationError を返す
//     （FK 制約違反による 500 を防ぎ、BR-T-04「有効ユーザーの ID のみ許可」の存在面を担保する）
func resolveAssignees(repo domain.UserRepository, idStrings []string) ([]*domain.User, error) {
	seen := make(map[string]struct{}, len(idStrings))
	ids := make([]domain.UserID, 0, len(idStrings))
	for _, s := range idStrings {
		id, err := domain.NewUserID(s)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, nil
	}

	users, err := repo.FindByIDs(ids)
	if err != nil {
		return nil, err
	}
	// ids は重複除去済みで FindByIDs は重複なしに解決するため、件数不足＝未存在 ID あり。
	// 不等号ではなく不足判定にして、万一リポジトリが重複行を返しても誤検知しない安全側に倒す。
	if len(users) < len(ids) {
		return nil, apperrors.NewValidationError("validation.assignee.not_found", nil)
	}
	return users, nil
}
