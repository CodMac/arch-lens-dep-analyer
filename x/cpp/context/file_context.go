// Package context 提供 C/C++ 语言专属的 FileContext / GlobalContext 实现。
//
// 通用索引/去重/锁机制复用 core.Base*；本包只承载 C/C++ 的语义差异：
//   - 文件可含多个 namespace（实现 NamespaceAware）
//   - 命名空间按 :: 逐级注册为 NAMESPACE 元素
//   - 重载索引收 Function 族（C 自由函数 + 声明）
package context

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
)

// NamespaceAware 是 C/C++ 专属的可选能力接口：
// 一文件可含多个命名空间，其余语言无需实现。
type NamespaceAware interface {
	core.FileContext

	// Namespaces 命名空间列表（按出现顺序，含嵌套前缀）。
	Namespaces() []string

	// SetNamespaces 设置命名空间列表。
	SetNamespaces(ns []string)
}

// FileContext 是 C/C++ 的文件上下文：在通用实现之上增加命名空间列表。
type FileContext struct {
	*core.BaseFileContext

	namespaces []string
}

// NewFileContext 创建 C/C++ 文件上下文。
func NewFileContext(filePath string, root coreast.Node, sourceBytes *[]byte) *FileContext {
	return &FileContext{
		BaseFileContext: core.NewBaseFileContext(filePath, root, sourceBytes),
	}
}

// Namespaces 返回文件内出现的命名空间（按出现顺序，含嵌套前缀）。
func (fc *FileContext) Namespaces() []string { return fc.namespaces }

// SetNamespaces 设置命名空间列表。
func (fc *FileContext) SetNamespaces(ns []string) { fc.namespaces = ns }
