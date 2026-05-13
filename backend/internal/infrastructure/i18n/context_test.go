package i18n_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	infrai18n "github.com/kuritaeiji/claude-code-ddd/internal/infrastructure/i18n"
)

// resetFallbackTranslator は他テストへの影響を避けるためフォールバックを未注入状態に戻す。
// SetFallbackTranslator がプロセスグローバルなので、状態を触るテストは t.Parallel() しない。
func resetFallbackTranslator(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { infrai18n.SetFallbackTranslator(nil) })
}

func TestTranslatorFrom_ReturnsInjectedTranslator(t *testing.T) {
	resetFallbackTranslator(t)
	bundle, err := infrai18n.BuildBundle(localesDir(t))
	require.NoError(t, err)

	ja := infrai18n.NewRequestTranslator(bundle, "ja")
	ctx := infrai18n.WithTranslator(context.Background(), ja)

	got := infrai18n.TranslatorFrom(ctx)
	assert.Equal(t,
		"バリデーションエラー: メールアドレスは必須です",
		got.Translate("validation.email.required", nil),
	)
}

func TestTranslatorFrom_FallbackToEnglishWhenUnset(t *testing.T) {
	resetFallbackTranslator(t)
	bundle, err := infrai18n.BuildBundle(localesDir(t))
	require.NoError(t, err)

	infrai18n.SetFallbackTranslator(infrai18n.NewEnglishTranslator(bundle))

	got := infrai18n.TranslatorFrom(context.Background())
	assert.Equal(t,
		"validation error: email is required",
		got.Translate("validation.email.required", nil),
	)
}

func TestTranslatorFrom_PassthroughWhenNoFallbackSet(t *testing.T) {
	resetFallbackTranslator(t)
	// SetFallbackTranslator 未呼び出し / context にも未格納の場合は MessageID をそのまま返す。
	got := infrai18n.TranslatorFrom(context.Background())
	assert.Equal(t, "validation.email.required", got.Translate("validation.email.required", nil))
}
