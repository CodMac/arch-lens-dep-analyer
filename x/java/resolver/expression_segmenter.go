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
	// 1. 如果传入的直接就是 object_creation_expression 或 array_creation_expression
	case "object_creation_expression", "array_creation_expression":
		var typeNode *sitter.Node
		namedCount := int(node.NamedChildCount())
		for i := 0; i < namedCount; i++ {
			child := node.NamedChild(uint(i))
			childKind := child.Kind()
			if childKind == "scoped_type_identifier" || childKind == "type_identifier" || childKind == "generic_type" || helper.IsPrimitiveType(childKind) {
				typeNode = child
				break
			}
		}

		if typeNode != nil {
			rawTypeName := strings.TrimSpace(typeNode.Utf8Text(*es.src))
			chain.Head = ExpressionHead{
				Type:    HeadNewExpr,
				Name:    helper.Clean(rawTypeName),
				ASTNode: typeNode,
				RawText: rawTypeName,
			}
		} else {
			chain.Head = es.buildHead(node)
		}

	// 2. 🎯 新增防滑漏拦截：如果 NodeContextResolver 已经帮我们把 ExpressNode 精简成了类型的子节点
	// （例如：scoped_type_identifier 或 generic_type），且它的父级是 new 表达式，我们也必须直接将其作为 HeadNewExpr 处理！
	case "scoped_type_identifier", "generic_type":
		if parent := node.Parent(); parent != nil && (parent.Kind() == "object_creation_expression" || parent.Kind() == "array_creation_expression") {
			rawTypeName := strings.TrimSpace(node.Utf8Text(*es.src))
			chain.Head = ExpressionHead{
				Type:    HeadNewExpr,
				Name:    helper.Clean(rawTypeName),
				ASTNode: node,
				RawText: rawTypeName,
			}
			return
		}
		// 如果不是 new 出来的（只是普通的静态方法调用或外部类引用等），继续走 field_access 式的递归解构
		es.resolveFieldLikeChain(node, chain)

	case "field_access":
		if obj := node.ChildByFieldName("object"); obj != nil {
			es.resolveNode(obj, chain)
		}

		fieldNode := node.ChildByFieldName("field")
		if fieldNode != nil {
			fieldName := helper.Clean(fieldNode.Utf8Text(*es.src))

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

// 辅助函数：处理非 new 场景下的 type 链条
func (es *ExpressionSegmenter) resolveFieldLikeChain(node *sitter.Node, chain *ExpressionChain) {
	// 如果是普通的 scoped_type_identifier（如 A.B），按照 field_access 逻辑解开
	// A 作为 HeadIdent，.B 作为 SegmentClass 塞入链条
	raw := strings.TrimSpace(node.Utf8Text(*es.src))
	parts := strings.Split(raw, ".")
	if len(parts) > 1 {
		chain.Head = ExpressionHead{
			Type:    HeadIdent,
			Name:    parts[0],
			ASTNode: node,
			RawText: parts[0],
		}
		for _, part := range parts[1:] {
			chain.Segments = append(chain.Segments, ExpressionSegment{
				Kind:    SegmentClass,
				Name:    part,
				ASTNode: node,
				RawText: part,
			})
		}
	} else {
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
		// 🎯 增加判定：如果这个普通的 type_identifier 父级是 new 表达式，强制修正为 HeadNewExpr
		if parent := node.Parent(); parent != nil && (parent.Kind() == "object_creation_expression" || parent.Kind() == "array_creation_expression") {
			head.Type = HeadNewExpr
		} else {
			head.Type = HeadIdent
		}
	case "scoped_type_identifier", "generic_type":
		if parent := node.Parent(); parent != nil && (parent.Kind() == "object_creation_expression" || parent.Kind() == "array_creation_expression") {
			head.Type = HeadNewExpr
		} else {
			head.Type = HeadIdent
		}
	case "object_creation_expression", "array_creation_expression":
		head.Type = HeadNewExpr
		namedCount := int(node.NamedChildCount())
		for i := 0; i < namedCount; i++ {
			child := node.NamedChild(uint(i))
			ck := child.Kind()
			if ck == "scoped_type_identifier" || ck == "type_identifier" || ck == "generic_type" || helper.IsPrimitiveType(ck) {
				head.Name = helper.Clean(child.Utf8Text(*es.src))
				break
			}
		}
	case "string_literal", "decimal_integer_literal", "decimal_floating_point_literal", "boolean_literal":
		head.Type = HeadLiteral
	default:
		head.Type = HeadUnknown
	}

	return head
}

// skipParentheses 穿透多层嵌套的括号表达式
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
