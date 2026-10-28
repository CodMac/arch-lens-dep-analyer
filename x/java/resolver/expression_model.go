// Package resolver 实现 Java 动作解析的三段式：Context 提取 -> 链拉平 -> 沿链推导。
// 该包仅依赖 core/ast.Node 接口，与具体 tree-sitter 实现解耦。
package resolver

import (
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
)

// ExpressionHeadType 表示拉平链的头部类型。
type ExpressionHeadType string

const (
	// HeadThis 隐式/显式 this。
	HeadThis ExpressionHeadType = "this"
	// HeadSuper 显式 super。
	HeadSuper ExpressionHeadType = "super"
	// HeadIdent 普通标识符（变量名/类名，需运行时判定）。
	HeadIdent ExpressionHeadType = "ident"
	// HeadNewExpr 对象创建表达式（new X(...)）。
	HeadNewExpr ExpressionHeadType = "new_expr"
	// HeadCastExpr 强转/类型检查表达式 ((T) expr / instanceof T)。
	HeadCastExpr ExpressionHeadType = "cast_expr"
	// HeadImplicitMethod 无接收者的隐式方法调用（this 语义）。
	HeadImplicitMethod ExpressionHeadType = "implicit_method"
	// HeadType 已确认为类型名的标识符（静态访问）。
	HeadType ExpressionHeadType = "type"
	// HeadCtorInvoke 显式构造器调用（super() / this() / 类型构造器）。
	HeadCtorInvoke ExpressionHeadType = "ctor_invoke"
	// HeadUnknown 无法识别的头部。
	HeadUnknown ExpressionHeadType = "unknown"
)

// ExpressionHead 描述拉平链的头部。
type ExpressionHead struct {
	Type     ExpressionHeadType
	Name     string // 标识符名 / 类型名（Cast 为目标类型名，NewExpr 为构造类型名）
	CastType string // 仅 HeadCastExpr 使用
	// CtorRef 仅 HeadCtorInvoke 使用：显式构造器调用的目标引用（super / this / 类型名）。
	CtorRef string
	ASTNode coreast.Node
	RawText string
}

// SegmentKind 表示链中一段的语义。
type SegmentKind string

const (
	SegmentField  SegmentKind = "field"  // 字段访问 .name
	SegmentMethod SegmentKind = "method" // 方法调用 .name(...)
	SegmentArray  SegmentKind = "array"  // 数组下标 [i]
	SegmentClass  SegmentKind = "class"  // 嵌套类 .Inner
)

// ExpressionSegment 描述拉平链中的一段。
type ExpressionSegment struct {
	Kind    SegmentKind
	Name    string
	ASTNode coreast.Node
	RawText string
}

// ExpressionChain 是拉平后的符号链：Head + Segments。
type ExpressionChain struct {
	Head     ExpressionHead
	Segments []ExpressionSegment
}

// ContextResult 是 NodeContextResolver 产出的双轨上下文。
type ContextResult struct {
	// ExpressNode 动作表达式本身（如 method_invocation、identifier）。
	ExpressNode coreast.Node

	// ContextNode 动作发生的语义容器（如 expression_statement、variable_declarator）。
	ContextNode coreast.Node
}
