package errors

// ValidationError は入力値・ビジネスルール違反を表すドメインエラー。
// 人間可読な文字列ではなく翻訳キー（MessageID）とテンプレート変数のみを保持し、
// Error() で初めて Translator 経由で文字列化される。
type ValidationError struct {
	MessageID    string
	TemplateData map[string]any
}

func NewValidationError(messageID string, templateData map[string]any) *ValidationError {
	return &ValidationError{MessageID: messageID, TemplateData: templateData}
}

func (e *ValidationError) Error() string {
	return translate(e.MessageID, e.TemplateData)
}

// NotFoundError は対象リソースが存在しないことを表すドメインエラー。
// ValidationError と同じく翻訳キー（MessageID）とテンプレート変数のみを保持し、
// Error() で初めて Translator 経由で文字列化される。HTTP では 404 にマッピングされる。
type NotFoundError struct {
	MessageID    string
	TemplateData map[string]any
}

func NewNotFoundError(messageID string, templateData map[string]any) *NotFoundError {
	return &NotFoundError{MessageID: messageID, TemplateData: templateData}
}

func (e *NotFoundError) Error() string {
	return translate(e.MessageID, e.TemplateData)
}
