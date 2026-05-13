// Package i18n は go-i18n を使った翻訳基盤を提供する。
// pkg/errors.Translator の実装を englishTranslator / requestTranslator として持ち、
// アプリ起動時に *i18n.Bundle を構築する責務を担う。
package i18n

import (
	"fmt"
	"path/filepath"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// supportedLocales はアプリがサポートする言語タグと YAML ファイル名の対応。
// 先頭はデフォルト言語（English）。
var supportedLocales = []struct {
	tag      language.Tag
	filename string
}{
	{language.English, "en.yaml"},
	{language.Japanese, "ja.yaml"},
}

// BuildBundle は localesDir 配下の YAML をロードして *i18n.Bundle を返す。
// 起動時に 1 回だけ呼ばれ、以降は singleton として共有する想定。
func BuildBundle(localesDir string) (*goi18n.Bundle, error) {
	bundle := goi18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)

	for _, loc := range supportedLocales {
		path := filepath.Join(localesDir, loc.filename)
		if _, err := bundle.LoadMessageFile(path); err != nil {
			return nil, fmt.Errorf("i18n: load %s: %w", path, err)
		}
	}
	return bundle, nil
}
