package model

// Mores key 分组约定：
//   - 语言无关通用 key 定义于本文件（model 层），使用 "ele." / "rel." 前缀。
//   - 语言特化 key 定义于各语言 constants 包（如 x/java/constants），使用 "java.*" 前缀。
// 禁止在业务代码中直接书写裸字符串 key。

// 通用 CodeElement Mores key（语言无关）。
const (
	// EleRawName 元素的原始名称（可能含泛型等语法，如 List<String>）。
	EleRawName = "ele.raw_name"
)

// 通用 DependencyRelation Mores key（语言无关）。
const (
	// RelRawText 动作的完整源码文本（如 data.name = "Hi"）。
	RelRawText = "rel.raw_text"

	// RelAstKind 触发该关系的 AST 节点类型（如 assignment_expression）。
	RelAstKind = "rel.node_ast_kind"

	// RelContextAstKind 该动作发生的容器 AST 类型（如 method_declaration）。
	RelContextAstKind = "rel.context_ast_kind"
)
