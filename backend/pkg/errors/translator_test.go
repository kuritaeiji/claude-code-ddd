package errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

// fakeTranslator は受け取った MessageID と TemplateData を可視化して返すスタブ。
type fakeTranslator struct {
	calls []translateCall
}

type translateCall struct {
	messageID    string
	templateData map[string]any
}

func (f *fakeTranslator) Translate(messageID string, templateData map[string]any) string {
	f.calls = append(f.calls, translateCall{messageID: messageID, templateData: templateData})
	if v, ok := templateData["Suffix"]; ok {
		return messageID + ":" + v.(string)
	}
	return messageID
}

// resetDefaultTranslator はテスト後にデフォルトを未注入状態へ戻す。
// 並列実行を避けるため、Translator 系のテストは t.Parallel() を呼ばない。
func resetDefaultTranslator(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { apperrors.SetDefaultTranslator(nil) })
}

func TestValidationError_Error_FallbackWhenTranslatorUnset(t *testing.T) {
	resetDefaultTranslator(t)
	apperrors.SetDefaultTranslator(nil)

	err := apperrors.NewValidationError("validation.email.required", nil)
	assert.Equal(t, "validation.email.required", err.Error())
}

func TestValidationError_Error_UsesInjectedTranslator(t *testing.T) {
	resetDefaultTranslator(t)
	stub := &fakeTranslator{}
	apperrors.SetDefaultTranslator(stub)

	err := apperrors.NewValidationError("validation.display_name.length", map[string]any{
		"Suffix": "1-50",
	})

	assert.Equal(t, "validation.display_name.length:1-50", err.Error())
	require.Len(t, stub.calls, 1)
	assert.Equal(t, "validation.display_name.length", stub.calls[0].messageID)
	assert.Equal(t, "1-50", stub.calls[0].templateData["Suffix"])
}

func TestValidationError_ErrorAs_StillWorks(t *testing.T) {
	resetDefaultTranslator(t)

	var wrapped error = apperrors.NewValidationError("validation.email.invalid", nil)
	var ve *apperrors.ValidationError
	require.True(t, errors.As(wrapped, &ve))
	assert.Equal(t, "validation.email.invalid", ve.MessageID)
}
