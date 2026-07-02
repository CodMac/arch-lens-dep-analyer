package resolver

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// ExpressionSegmenter 负责将任意 ExpressNode 切割为标准符号推导链
type ExpressionSegmenter struct {
	src *[]byte
}

// NewExpressionSegmenter 创建表达式分段器
func NewExpressionSegmenter(srcBytes *[]byte) *ExpressionSegmenter {
	return &ExpressionSegmenter{src: srcBytes}
}

// Segment 将通过 NodeContextResolver 获取到的 ExpressNode 解析为拉平的拓扑链
func (es *ExpressionSegmenter) Segment(node *sitter.Node) *ExpressionChain {
	if node == nil {
		return nil
	}

	chain := &ExpressionChain{
		RawText:  strings.TrimSpace(node.Utf8Text(*es.src)),
		Segments: make([]ExpressionSegment, 0),
	}

	// 剥离并填充
	es.resolveNode(node, chain)

	return chain
}

// resolveNode 核心递归剥离逻辑
func (es *ExpressionSegmenter) resolveNode(node *sitter.Node, chain *ExpressionChain) {
	kind := node.Kind()

	switch kind {
	case "parenthesized_expression":
		// 遇到括号表达式 (expr)，脱壳穿透
		if child := node.NamedChild(0); child != nil {
			es.resolveNode(child, chain)
		}

	case "field_access":
		// 1. 先递归解析左侧的 object 接收者
		if obj := node.ChildByFieldName("object"); obj != nil {
			es.resolveNode(obj, chain)
		}
		// 2. 将当前字段访问追加到右侧
		fieldNode := node.ChildByFieldName("field")
		if fieldNode != nil {
			chain.Segments = append(chain.Segments, ExpressionSegment{
				Kind:    SegmentField,
				Name:    fieldNode.Utf8Text(*es.src),
				ASTNode: node,
				RawText: node.Utf8Text(*es.src),
			})
		}

	case "method_invocation":
		// 1. 递归解析隐式或显式的接收者 object
		if obj := node.ChildByFieldName("object"); obj != nil {
			es.resolveNode(obj, chain)
		}
		// 2. 追加当前的方法调用段
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			chain.Segments = append(chain.Segments, ExpressionSegment{
				Kind:    SegmentMethod,
				Name:    nameNode.Utf8Text(*es.src),
				ASTNode: node,
				RawText: node.Utf8Text(*es.src),
			})
		}

	case "array_access":
		// 1. 递归解析数组所属的对象（基座）
		if arrayObj := node.ChildByFieldName("array"); arrayObj != nil {
			es.resolveNode(arrayObj, chain)
		}
		// 2. 追加数组读取段
		chain.Segments = append(chain.Segments, ExpressionSegment{
			Kind:    SegmentArray,
			Name:    "",
			ASTNode: node,
			RawText: node.Utf8Text(*es.src),
		})

	default:
		// 终点边界：当无法再剥离出子 object 时，说明触达了链条的起点（Head）
		chain.Head = es.buildHead(node)
	}
}

// buildHead 将基底节点归类为标准的 Head 结构
func (es *ExpressionSegmenter) buildHead(node *sitter.Node) ExpressionHead {
	raw := strings.TrimSpace(node.Utf8Text(*es.src))
	kind := node.Kind()

	head := ExpressionHead{
		ASTNode: node,
		RawText: raw,
		Name:    raw,
	}

	switch kind {
	case "identifier":
		if raw == "this" {
			head.Type = HeadThis
		} else if raw == "super" {
			head.Type = HeadSuper
		} else {
			head.Type = HeadIdent // 可能是局部变量、类名、包名路径起点
		}
	case "object_creation_expression":
		head.Type = HeadNewExpr
	case "string_literal", "decimal_integer_literal", "decimal_floating_point_literal", "boolean_literal":
		head.Type = HeadLiteral
	default:
		head.Type = HeadUnknown
	}

	return head
}
