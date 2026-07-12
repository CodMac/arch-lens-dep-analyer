package resolver

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// ExpressionHeadType 定义链条起点的语法性质
type ExpressionHeadType int

const (
	HeadUnknown        ExpressionHeadType = iota
	HeadThis                              // this 关键字
	HeadSuper                             // super 关键字
	HeadLiteral                           // "str", 123 等字面量起点
	HeadNewExpr                           // new Object() 匿名对象起点
	HeadIdent                             // 标识符（局部变量、类名、实例字段）
	HeadImplicitMethod                    // 隐式方法调用起点（如 simpleMethod()）
)

// ExpressionHead 链式调用的入口节点
type ExpressionHead struct {
	Type    ExpressionHeadType
	Name    string       // 标识符名称（如 "order"），若为 New/Literal 则同 RawText
	ASTNode *sitter.Node // 对应的 Tree-sitter 节点
	RawText string       // 源码文本
}

// SegmentKind 定义后续求值每一步的动作类型
type SegmentKind int

const (
	SegmentField  SegmentKind = iota // .field 字段访问
	SegmentMethod                    // .method(...) 方法调用
	SegmentArray                     // [...] 数组下标读取
)

// ExpressionSegment 链式调用向后递进的每一个切片
type ExpressionSegment struct {
	Kind    SegmentKind
	Name    string       // 动作目标名称。若是 Method 则为方法名；若是 Field 则为字段名；若是 Array 则为空
	ASTNode *sitter.Node // 当前这一层完整的表达式节点（如整个 method_invocation 节点）
	RawText string       // 当前这一步的文本片段
}

// ExpressionChain 被拉平后的标准解析链
type ExpressionChain struct {
	Head     ExpressionHead
	Segments []ExpressionSegment
	RawText  string // 原始完整链条文本
}
