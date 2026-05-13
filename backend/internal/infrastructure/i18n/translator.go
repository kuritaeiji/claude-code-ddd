package i18n

import (
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"

	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

// localizerTranslator は *goi18n.Localizer を pkg/errors.Translator として薄くラップする。
// 翻訳失敗時は MessageID をそのまま返し、上位（ロガー等）が落ちないようにする。
type localizerTranslator struct {
	loc *goi18n.Localizer
}

func (t *localizerTranslator) Translate(messageID string, templateData map[string]any) string {
	msg, err := t.loc.Localize(&goi18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	})
	if err != nil {
		return messageID
	}
	return msg
}

// NewEnglishTranslator は err.Error() のデフォルト出力に使う英語固定 Translator を返す。
// 起動時に errors.SetDefaultTranslator へ注入する。
func NewEnglishTranslator(bundle *goi18n.Bundle) apperrors.Translator {
	return &localizerTranslator{loc: goi18n.NewLocalizer(bundle, language.English.String())}
}

// NewRequestTranslator はリクエストの Accept-Language ヘッダから Localizer を生成する。
// 未対応 / 解析失敗時は English にフォールバックする（go-i18n の仕様どおり）。
func NewRequestTranslator(bundle *goi18n.Bundle, acceptLanguage string) apperrors.Translator {
	loc := goi18n.NewLocalizer(bundle, acceptLanguage, language.English.String())
	return &localizerTranslator{loc: loc}
}
