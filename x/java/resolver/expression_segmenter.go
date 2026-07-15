package resolver

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
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
// 🎯 引入 relType 明确上下文语义，驱动确定性分流
func (es *ExpressionSegmenter) Segment(node *sitter.Node, relType model.DependencyType) *ExpressionChain {
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

	// 填充起点和后续分段
	es.resolveNode(node, chain, relType)

	return chain
}

// resolveNode 核心递归剥离与分流逻辑
func (es *ExpressionSegmenter) resolveNode(node *sitter.Node, chain *ExpressionChain, relType model.DependencyType) {
	if node == nil {
		return
	}

	node = es.skipParentheses(node)
	kind := node.Kind()

	// ==========================================
	// 🎯 1. 确定性上下文分流：CREATE 关系
	// ==========================================
	if relType == model.Create || kind == "object_creation_expression" || kind == "array_creation_expression" {
		rawTypeName := strings.TrimSpace(node.Utf8Text(*es.src))
		chain.Head = ExpressionHead{
			Type:    HeadNewExpr,
			Name:    helper.Clean(rawTypeName),
			ASTNode: node,
			RawText: rawTypeName,
		}
		return // 直接返回，无需向下递归
	}

	// ==========================================
	// 🎯 2. 确定性上下文分流：CAST 关系
	// ==========================================
	if relType == model.Cast || kind == "cast_expression" || kind == "instanceof_expression" {
		es.resolveCastNode(node, chain)
		return
	}

	// ==========================================
	// 🎯 3. 其他关系（如 Call, Use 等）进行常规链式剥离
	// ==========================================
	switch kind {
	case "field_access":
		if obj := node.ChildByFieldName("object"); obj != nil {
			es.resolveNode(obj, chain, relType)
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
			es.resolveNode(obj, chain, relType)

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
			chain.Head = es.buildHead(node, relType)
		}

	case "method_reference":
		if obj := node.ChildByFieldName("object"); obj != nil {
			es.resolveNode(obj, chain, relType)
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
			es.resolveNode(arrayObj, chain, relType)
		}
		chain.Segments = append(chain.Segments, ExpressionSegment{
			Kind:    SegmentArray,
			Name:    "",
			ASTNode: node,
			RawText: node.Utf8Text(*es.src),
		})

	case "scoped_type_identifier", "generic_type":
		es.resolveFieldLikeChain(node, chain)

	default:
		chain.Head = es.buildHead(node, relType)
	}
}

// resolveCastNode 专门负责提取强转与模式匹配信息
func (es *ExpressionSegmenter) resolveCastNode(node *sitter.Node, chain *ExpressionChain) {
	kind := node.Kind()

	if kind == "cast_expression" {
		typeNode := node.ChildByFieldName("type")
		valueNode := node.ChildByFieldName("value")

		var castType string
		if typeNode != nil {
			castType = helper.Clean(typeNode.Utf8Text(*es.src))
		}

		// 应对多重强转，向下钻取至底层实际的数据源节点
		finalValueNode := valueNode
		for finalValueNode != nil && finalValueNode.Kind() == "cast_expression" {
			finalValueNode = finalValueNode.ChildByFieldName("value")
		}

		var castValue string
		if finalValueNode != nil {
			castValue = helper.Clean(finalValueNode.Utf8Text(*es.src))
		}

		chain.Head = ExpressionHead{
			Type:     HeadCastExpr,
			Name:     castValue,
			CastType: castType,
			ASTNode:  node,
			RawText:  node.Utf8Text(*es.src),
		}
	} else if kind == "instanceof_expression" {
		var leftNode, typeNode *sitter.Node
		namedCount := node.NamedChildCount()
		if namedCount >= 2 {
			leftNode = node.NamedChild(0)
			typeNode = node.NamedChild(1)
		}

		var castType, castValue string
		if typeNode != nil {
			castType = helper.Clean(typeNode.Utf8Text(*es.src))
		}
		if leftNode != nil {
			castValue = helper.Clean(leftNode.Utf8Text(*es.src))
		}

		chain.Head = ExpressionHead{
			Type:     HeadCastExpr,
			Name:     castValue,
			CastType: castType,
			ASTNode:  node,
			RawText:  node.Utf8Text(*es.src),
		}
	}
}

// resolveFieldLikeChain 处理 field 样式声明的 type 链条 (例如 A.B.C)
func (es *ExpressionSegmenter) resolveFieldLikeChain(node *sitter.Node, chain *ExpressionChain) {
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
		chain.Head = es.buildHead(node, model.Use)
	}
}

// buildHead 构造链条头部节点性质
func (es *ExpressionSegmenter) buildHead(node *sitter.Node, relType model.DependencyType) ExpressionHead {
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
	case "identifier", "type_identifier", "scoped_type_identifier", "generic_type":
		head.Type = HeadIdent
	case "string_literal", "decimal_integer_literal", "decimal_floating_point_literal", "boolean_literal":
		head.Type = HeadLiteral
	default:
		if helper.IsPrimitiveType(kind) {
			head.Type = HeadIdent
		} else {
			head.Type = HeadUnknown
		}
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
