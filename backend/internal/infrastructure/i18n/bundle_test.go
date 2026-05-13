package i18n_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	infrai18n "github.com/kuritaeiji/claude-code-ddd/internal/infrastructure/i18n"
)

// localesDir はテスト実行ディレクトリに依存しないよう、現在のテストファイルを起点に解決する。
func localesDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	// backend/internal/infrastructure/i18n/bundle_test.go から backend/locales/ へ
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "locales"))
}

func TestBuildBundle_LoadsEnAndJa(t *testing.T) {
	t.Parallel()
	bundle, err := infrai18n.BuildBundle(localesDir(t))
	require.NoError(t, err)

	en := infrai18n.NewEnglishTranslator(bundle)
	assert.Equal(t, "validation error: email is required", en.Translate("validation.email.required", nil))
}

func TestBuildBundle_ErrorWhenLocalesMissing(t *testing.T) {
	t.Parallel()
	_, err := infrai18n.BuildBundle(filepath.Join(t.TempDir(), "nonexistent"))
	assert.Error(t, err)
}

func TestRequestTranslator_Japanese(t *testing.T) {
	t.Parallel()
	bundle, err := infrai18n.BuildBundle(localesDir(t))
	require.NoError(t, err)

	tr := infrai18n.NewRequestTranslator(bundle, "ja")
	assert.Equal(t, "バリデーションエラー: メールアドレスは必須です", tr.Translate("validation.email.required", nil))
}

func TestRequestTranslator_FallbackToEnglishOnUnsupported(t *testing.T) {
	t.Parallel()
	bundle, err := infrai18n.BuildBundle(localesDir(t))
	require.NoError(t, err)

	tr := infrai18n.NewRequestTranslator(bundle, "fr-FR")
	assert.Equal(t, "validation error: email is required", tr.Translate("validation.email.required", nil))
}

func TestRequestTranslator_FallbackToEnglishOnEmptyAcceptLanguage(t *testing.T) {
	t.Parallel()
	bundle, err := infrai18n.BuildBundle(localesDir(t))
	require.NoError(t, err)

	tr := infrai18n.NewRequestTranslator(bundle, "")
	assert.Equal(t, "validation error: email is required", tr.Translate("validation.email.required", nil))
}

func TestTranslator_ExpandsTemplateData(t *testing.T) {
	t.Parallel()
	bundle, err := infrai18n.BuildBundle(localesDir(t))
	require.NoError(t, err)

	en := infrai18n.NewEnglishTranslator(bundle)
	assert.Equal(t,
		"validation error: displayName must be 1-50 characters",
		en.Translate("validation.display_name.length", map[string]any{"Min": 1, "Max": 50}),
	)

	ja := infrai18n.NewRequestTranslator(bundle, "ja")
	assert.Equal(t,
		"バリデーションエラー: 表示名は1〜50文字である必要があります",
		ja.Translate("validation.display_name.length", map[string]any{"Min": 1, "Max": 50}),
	)
}

func TestTranslator_UnknownMessageIDReturnsMessageID(t *testing.T) {
	t.Parallel()
	bundle, err := infrai18n.BuildBundle(localesDir(t))
	require.NoError(t, err)

	en := infrai18n.NewEnglishTranslator(bundle)
	assert.Equal(t, "no.such.key", en.Translate("no.such.key", nil))
}
