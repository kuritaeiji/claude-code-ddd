package errors

import "sync"

// Translator は MessageID とテンプレート変数を翻訳済み文字列に変換する。
// pkg/errors は go-i18n に直接依存しないよう、このIFのみを公開し、
// 実装は Infrastructure 層から SetDefaultTranslator で注入される。
type Translator interface {
	Translate(messageID string, templateData map[string]any) string
}

var (
	defaultTranslatorMu sync.RWMutex
	defaultTranslator   Translator
)

// SetDefaultTranslator は err.Error() が参照するパッケージ既定 Translator を設定する。
// アプリ起動時に Infrastructure 層から 1 度だけ呼ぶ想定。
func SetDefaultTranslator(t Translator) {
	defaultTranslatorMu.Lock()
	defer defaultTranslatorMu.Unlock()
	defaultTranslator = t
}

// translate はデフォルト Translator で翻訳する。
// 未注入時は MessageID をそのまま返す（テスト時の panic 回避）。
func translate(messageID string, templateData map[string]any) string {
	defaultTranslatorMu.RLock()
	t := defaultTranslator
	defaultTranslatorMu.RUnlock()
	if t == nil {
		return messageID
	}
	return t.Translate(messageID, templateData)
}
