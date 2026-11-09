// Package context 提供 Java 语言专属的 FileContext / GlobalContext 实现。
//
// 通用索引/去重/锁机制复用 core.Base*；本包只承载 Java 的语义差异：
//   - 包名按点号逐级注册为 PACKAGE 元素
//   - 重载索引只收 Method（Java 无自由函数）
package context

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
)

// FileContext 是 Java 的文件上下文：直接复用 core 通用实现。
type FileContext struct {
	*core.BaseFileContext
}

// NewFileContext 创建 Java 文件上下文。
func NewFileContext(filePath, packageName string, root coreast.Node, sourceBytes *[]byte) *FileContext {
	return &FileContext{
		BaseFileContext: core.NewBaseFileContext(filePath, packageName, root, sourceBytes),
	}
}
