package core

import (
	"github.com/CodMac/arch-lens-dep-analyer/core/ast"
)

// ParserFactory 负责创建语言解析器。
type ParserFactory interface {
	// NewParser 创建一个解析器实例。
	NewParser() Parser
}

// Parser 是语言解析器的抽象：将源文件解析为 AST。
type Parser interface {
	// ParseFile 解析指定路径的源文件为 SourceFile。
	ParseFile(path string) (ast.SourceFile, error)

	// Close 释放解析器底层资源。
	Close()
}
