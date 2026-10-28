package core

import (
	"github.com/CodMac/arch-lens-dep-analyer/core/ast"
)

// Collector 负责将单个源文件解析为 FileContext（内含该文件的全部代码元素定义）。
// 各阶段中该接口在文件级并行执行。
type Collector interface {
	// CollectDefinitions 从源文件收集符号定义并产出 FileContext。
	CollectDefinitions(src ast.SourceFile) (*FileContext, error)
}
