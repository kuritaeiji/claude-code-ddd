package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kuritaeiji/claude-code-ddd/internal/infrastructure/i18n"
	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

type errorResponse struct {
	Message string `json:"message"`
}

// writeError はドメインエラーを HTTP レスポンスに変換する。
// 翻訳は context に注入されたリクエスト用 Translator で実施し、err.Error() は使わない。
// err.Error() を直接呼ぶとデフォルト（英語固定）Translator が使われてしまうため、
// Accept-Language が反映されたメッセージにするには明示的に MessageID / TemplateData から訳す必要がある。
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	translator := i18n.TranslatorFrom(r.Context())

	var ve *apperrors.ValidationError
	if errors.As(err, &ve) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Message: translator.Translate(ve.MessageID, ve.TemplateData)})
		return
	}

	writeJSON(w, http.StatusInternalServerError, errorResponse{Message: translator.Translate("common.internal_error", nil)})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
