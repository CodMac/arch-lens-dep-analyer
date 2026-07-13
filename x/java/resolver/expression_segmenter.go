package resolver

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// ExpressionSegmenter 负责将任意 ExpressNode 切割为标准符号推导链
type ExpressionSegmenter struct {
	src *[]byte
}

// NewExpressionSegmenter 创建表达式分段器
func NewExpressionSegmenter(fCtx *core.FileContext) *ExpressionSegmenter {
	return &ExpressionSegmenter{src: fCtx.SourceBytes}
}

// Segment 将通过 NodeContextResolver 获取到的 ExpressNode 解析为拉平的拓扑链
func (es *ExpressionSegmenter) Segment(node *sitter.Node) *ExpressionChain {
	if node == nil {
		return nil
	}

	node = es.skipParentheses(node)
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
	if node == nil {
		return
	}

	node = es.skipParentheses(node)
	kind := node.Kind()

	switch kind {
	case "field_access":
		if obj := node.ChildByFieldName("object"); obj != nil {
			es.resolveNode(obj, chain)
		}

		fieldNode := node.ChildByFieldName("field")
		if fieldNode != nil {
			fieldName := helper.Clean(fieldNode.Utf8Text(*es.src))
			// 🎯 动态判定 Segment 类型：如果该段落符合类名潜质（如大写开头），则识别为 SegmentClass 辅助精细解析
			segKind := SegmentField
			if helper.IsPotentialClassName(fieldName) {
				segKind = SegmentClass
			}
			chain.Segments = append(chain.Segments, ExpressionSegment{
				Kind:    segKind,
				Name:    fieldName,
				ASTNode: node,
				RawText: node.Utf8Text(*es.src),
			})
		}

	case "method_invocation":
		if obj := node.ChildByFieldName("object"); obj != nil {
			// 1. 有 object：说明是显式链式调用（如 obj.method()），正常向前递归
			es.resolveNode(obj, chain)

			nameNode := node.ChildByFieldName("name")
			if nameNode != nil {
				chain.Segments = append(chain.Segments, ExpressionSegment{
					Kind:    SegmentMethod,
					Name:    helper.Clean(nameNode.Utf8Text(*es.src)),
					ASTNode: node,
					RawText: node.Utf8Text(*es.src),
				})
			}
		} else {
			// 2. 没有 object：说明是独立的隐式方法调用（如 simpleMethod()）
			// 它本身就是链条的起点（Head），直接在此处截断，不进 default
			chain.Head = es.buildHead(node)
		}

	case "method_reference":
		if obj := node.ChildByFieldName("object"); obj != nil {
			es.resolveNode(obj, chain)
		}

		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			chain.Segments = append(chain.Segments, ExpressionSegment{
				Kind:    SegmentMethod,
				Name:    helper.Clean(nameNode.Utf8Text(*es.src)),
				ASTNode: node,
				RawText: node.Utf8Text(*es.src),
			})
		}

	case "array_access":
		if arrayObj := node.ChildByFieldName("array"); arrayObj != nil {
			es.resolveNode(arrayObj, chain)
		}
		chain.Segments = append(chain.Segments, ExpressionSegment{
			Kind:    SegmentArray,
			Name:    "",
			ASTNode: node,
			RawText: node.Utf8Text(*es.src),
		})

	default:
		chain.Head = es.buildHead(node)
	}
}

func (es *ExpressionSegmenter) buildHead(node *sitter.Node) ExpressionHead {
	node = es.skipParentheses(node)
	raw := strings.TrimSpace(node.Utf8Text(*es.src))
	kind := node.Kind()

	head := ExpressionHead{
		ASTNode: node,
		RawText: raw,
		Name:    helper.Clean(raw),
	}

	switch kind {
	case "this":
		head.Type = HeadThis
	case "super":
		head.Type = HeadSuper
	case "explicit_constructor_invocation":
		if strings.HasPrefix(raw, "super") {
			head.Type = HeadSuperConstructor
			head.Name = "super"
		} else if strings.HasPrefix(raw, "this") {
			head.Type = HeadThisConstructor
			head.Name = "this"
		} else {
			head.Type = HeadUnknown
		}
	case "method_invocation":
		head.Type = HeadImplicitMethod
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			head.Name = helper.Clean(nameNode.Utf8Text(*es.src))
		}
	case "identifier", "type_identifier":
		head.Type = HeadIdent
	case "object_creation_expression":
		head.Type = HeadNewExpr
	case "string_literal", "decimal_integer_literal", "decimal_floating_point_literal", "boolean_literal":
		head.Type = HeadLiteral
	default:
		head.Type = HeadUnknown
	}

	return head
}

// skipParentheses 穿透多层嵌套的括号表达式，直达内部的核心语义节点
func (es *ExpressionSegmenter) skipParentheses(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && curr.Kind() == "parenthesized_expression" {
		if curr.NamedChildCount() > 0 {
			curr = curr.NamedChild(0)
		} else {
			break
		}
	}
	return curr
}
