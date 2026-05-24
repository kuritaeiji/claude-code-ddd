// Package controller は net/http ハンドラーと共通ミドルウェアを集約する。
package controller

import (
	"net/http"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/kuritaeiji/claude-code-ddd/internal/infrastructure/i18n"
)

// NewI18nMiddleware は Accept-Language を読んでリクエスト用 Translator を context に格納するミドルウェアを返す。
// ハンドラ側は i18n.TranslatorFrom(r.Context()) で取り出す。
// Bundle は起動時に 1 回だけ構築する singleton。
func NewI18nMiddleware(bundle *goi18n.Bundle) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			translator := i18n.NewRequestTranslator(bundle, r.Header.Get("Accept-Language"))
			ctx := i18n.WithTranslator(r.Context(), translator)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
