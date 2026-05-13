package i18n

import (
	"context"
	"sync"

	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

// ctxKey は外部パッケージから値が偽装されないよう非公開型を採用する。
type ctxKey struct{}

var translatorKey ctxKey

var (
	fallbackTranslatorMu sync.RWMutex
	fallbackTranslator   apperrors.Translator
)

// SetFallbackTranslator は TranslatorFrom(ctx) の ctx 未設定時フォールバックを登録する。
// 起動時に英語固定 Translator を渡しておくことで、Controller 層は ctx だけで翻訳できる
// （ハンドラごとに bundle を引き回す必要がない）。
// nil を渡すと未注入状態に戻る（テスト用途）。
func SetFallbackTranslator(t apperrors.Translator) {
	fallbackTranslatorMu.Lock()
	defer fallbackTranslatorMu.Unlock()
	fallbackTranslator = t
}

// WithTranslator はリクエスト用 Translator を context に格納する。
// Controller 層の i18n ミドルウェアから呼ばれる想定。
func WithTranslator(parent context.Context, t apperrors.Translator) context.Context {
	return context.WithValue(parent, translatorKey, t)
}

// TranslatorFrom は context から Translator を取り出す。
// 未設定（context にない / 型違い）の場合は SetFallbackTranslator で登録された
// 英語固定 Translator を返し、呼び出し側で nil チェックを書かなくて済むようにする。
// フォールバックも未設定の場合は MessageID をそのまま返す passthrough Translator を返す。
func TranslatorFrom(ctx context.Context) apperrors.Translator {
	if t, ok := ctx.Value(translatorKey).(apperrors.Translator); ok && t != nil {
		return t
	}
	fallbackTranslatorMu.RLock()
	t := fallbackTranslator
	fallbackTranslatorMu.RUnlock()
	if t != nil {
		return t
	}
	return passthroughTranslator{}
}

// passthroughTranslator は SetFallbackTranslator 未注入時の最終フォールバック。
// MessageID をそのまま返すため、テスト時の nil panic を避けつつ、設定漏れを目視で検出できる。
type passthroughTranslator struct{}

func (passthroughTranslator) Translate(messageID string, _ map[string]any) string {
	return messageID
}
