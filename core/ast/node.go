// Package ast 定义语言无关的 AST 薄封装接口，将核心层与具体解析器（如 tree-sitter）解耦。
package ast

// Node 是 AST 节点的语言无关抽象，形状贴近 tree-sitter 的导航能力。
// 各语言实现提供适配器将具体节点（如 *sitter.Node）映射到本接口。
type Node interface {
	// Kind 返回节点的语法类型名（如 "method_invocation"）。
	Kind() string

	// Utf8Text 返回节点覆盖的源码文本。
	Utf8Text(src []byte) string

	// ChildByFieldName 按字段名返回子节点（tree-sitter 字段语义），不存在返回 nil。
	ChildByFieldName(name string) Node

	// ChildByFieldNameRecursive 递归向下查找具名子字段节点，用于跨层取值。
	ChildByFieldNameRecursive(name string) Node

	// NamedChild 返回第 i 个具名子节点。
	NamedChild(i int) Node

	// NamedChildCount 返回具名子节点数量。
	NamedChildCount() int

	// Child 返回第 i 个子节点（含匿名）。
	Child(i int) Node

	// ChildCount 返回子节点总数（含匿名）。
	ChildCount() int

	// Parent 返回父节点，根节点返回 nil。
	Parent() Node

	// PrevSibling 返回前一个具名兄弟节点（用于注释提取等场景），无则返回 nil。
	PrevSibling() Node

	// StartByte / EndByte 返回节点在源码中的字节区间。
	StartByte() int
	EndByte() int

	// StartPoint / EndPoint 返回节点行列位置。
	StartPoint() Point
	EndPoint() Point

	// IsNamed 返回节点是否为具名节点。
	IsNamed() bool

	// IsError 返回节点是否为错误节点。
	IsError() bool

	// Equal 判断两个节点是否指向同一语法节点。
	Equal(other Node) bool

	// String 返回节点描述文本（调试用）。
	String() string
}

// Point 描述源码中的行列位置（0 基）。
type Point struct {
	Row    int // 行号（0 基）
	Column int // 列号（0 基，字节偏移）
}

// SourceFile 是解析后的源文件的语言无关抽象。
type SourceFile interface {
	// Root 返回语法树根节点。
	Root() Node

	// Content 返回源码字节内容。
	Content() []byte

	// Path 返回文件路径。
	Path() string
}
