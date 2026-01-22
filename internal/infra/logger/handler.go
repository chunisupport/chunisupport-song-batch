package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

// CustomHandler はslog.Handlerをラップしてカスタムフォーマットを提供します
type CustomHandler struct {
	slog.Handler
}

// NewCustomHandler は新しいCustomHandlerのインスタンスを生成します
func NewCustomHandler() *CustomHandler {
	return &CustomHandler{}
}

// logColorize はログレベルに応じてメッセージを色付けします
func logColorize(level slog.Level, msg string) string {
	switch level {
	case slog.LevelDebug:
		return fmt.Sprintf("\033[0;38;5;245m%s\033[0m", msg) // Gray
	case slog.LevelInfo:
		return fmt.Sprintf("\033[0;37m%s\033[0m", msg) // White
	case slog.LevelWarn:
		return fmt.Sprintf("\033[0;33m%s\033[0m", msg) // Yellow
	case slog.LevelError:
		return fmt.Sprintf("\033[0;35m%s\033[0m", msg) // Magenta
	default:
		return msg
	}
}

// WithAttrs は属性を持つ新しいHandlerを返します（この実装では何もしません）
func (h *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup はグループを持つ新しいHandlerを返します（この実装では何もしません）
func (h *CustomHandler) WithGroup(name string) slog.Handler {
	return h
}

// Enabled は指定されたレベルのログが有効かどうかを返します
func (h *CustomHandler) Enabled(_ context.Context, level slog.Level) bool {
	return true
}

// Handle はログレコードを処理し、カスタムフォーマットで出力します
func (h *CustomHandler) Handle(_ context.Context, r slog.Record) error {
	t := r.Time.Format("2006/01/02 15:04:05")
	level := r.Level.String()
	dTime := logColorize(r.Level, t)
	dLevelWithBracket := logColorize(r.Level, fmt.Sprintf("[%s]", level))
	dMessage := logColorize(r.Level, r.Message)
	fmt.Fprintf(os.Stdout, "%s %s %s", dTime, dLevelWithBracket, dMessage)

	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(os.Stdout, " %s=%v", a.Key, a.Value)
		return true
	})
	fmt.Fprintln(os.Stdout)
	return nil
}
