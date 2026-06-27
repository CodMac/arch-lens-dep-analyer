package constants

const (
	RelRawText        = "java.rel.raw_text"         // 完整的赋值语句源码 (eg,: data.name = "Hi")
	RelAstKind        = "java.rel.ast_kind"         // 触发该关系的那个 AST 节点的类型 (eg,: assignment_expression)
	RelContextAstKind = "java.rel.context_ast_kind" // 该动作发生的大环境或语句容器 (eg,: expression_statement 或 method_declaration)
)

const (
	TmpNode    = "tmp_node"
	TmpCtxNode = "tmp_ctx_node"
)
