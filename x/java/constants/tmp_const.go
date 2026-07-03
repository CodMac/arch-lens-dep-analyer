package constants

const (
	RelRawText        = "java.rel.raw_text"         // 完整的赋值语句源码 (eg,: data.name = "Hi")
	RelNodeAstKind    = "java.rel.node_ast_kind"    // 触发该关系的那个节点AST类型 (eg,: assignment_expression)
	RelExpressAstKind = "java.rel.express_ast_kind" // 该动作发生的表达式AST类型 (eg,: expression_statement 或 method_declaration)
	RelContextAstKind = "java.rel.context_ast_kind" // 该动作发生的语句容器AST类型 (eg,: expression_statement 或 method_declaration)

)

const (
	TmpNode        = "tmp_node"
	TmpExpressNode = "tmp_express_node"
	TmpCtxNode     = "tmp_ctx_node"
)
