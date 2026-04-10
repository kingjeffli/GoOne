package errs

import (
	"fmt"
	"strings"
)

// CfgError 带有上下文（文件/表/字段/行号/类型/根因）的配置错误
type CfgError struct {
	File  string
	Sheet string
	Field string
	Row   int
	Kind  string
	Msg   string
	Cause error
}

func (e *CfgError) Error() string {
	var b strings.Builder
	// 统一一行输出，便于在控制台/CI中查看
	// ❌ [类型] 文件=..., 表=..., 字段=..., 行=..., 详情=..., 根因=...
	if e.Kind != "" {
		b.WriteString("[" + e.Kind + "] ")
	}
	if e.File != "" {
		b.WriteString("类型=" + e.File + " ")
	}
	if e.Sheet != "" {
		b.WriteString("表名=「" + e.Sheet + "」 ")
	}
	if e.Field != "" {
		b.WriteString("字段=[" + e.Field + "] ")
	}
	if e.Row > 0 {
		b.WriteString(fmt.Sprintf("行=%d ", e.Row))
	}
	if e.Msg != "" {
		b.WriteString("详情【" + e.Msg + "】 ")
	}
	if e.Cause != nil {
		b.WriteString("根因-->" + e.Cause.Error() + "")
	}
	return strings.TrimSpace(b.String())
}

func New(file, sheet, field string, row int, kind, format string, args ...interface{}) *CfgError {
	return &CfgError{
		File:  file,
		Sheet: sheet,
		Field: field,
		Row:   row,
		Kind:  kind,
		Msg:   fmt.Sprintf(format, args...),
	}
}

func Wrap(cause error, file, sheet, field string, row int, kind, format string, args ...interface{}) *CfgError {
	return &CfgError{
		File:  file,
		Sheet: sheet,
		Field: field,
		Row:   row,
		Kind:  kind,
		Msg:   fmt.Sprintf(format, args...),
		Cause: cause,
	}
}
