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
	case "object_creation_expression":
		// 遍历 NamedChild, 兼容处理A.B.C()场景
		var typeNode *sitter.Node
		namedCount := int(node.NamedChildCount())
		for i := 0; i < namedCount; i++ {
			child := node.NamedChild(uint(i))
			childKind := child.Kind()
			// 匹配 Java 中 new 关键字后面允许出现的三种核心类型节点
			if childKind == "scoped_type_identifier" || childKind == "type_identifier" || childKind == "generic_type" {
				typeNode = child
				break
			}
		}

		if typeNode != nil {
			es.resolveTypeNode(typeNode, chain)
			// 修正 Head 类型，使其符合 NewExpr 的整体语义特征
			chain.Head.Type = HeadNewExpr
		} else {
			// 降级兜底
			chain.Head = es.buildHead(node)
		}

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

// resolveTypeNode 统一解构 Java 类型节点（支持嵌套内部类、泛型、普通类）
func (es *ExpressionSegmenter) resolveTypeNode(node *sitter.Node, chain *ExpressionChain) {
	if node == nil {
		return
	}

	kind := node.Kind()
	switch kind {
	case "generic_type":
		// 1. 泛型穿透：对 generic_type 的第一个命名子节点（真正的类名节点）展开递归
		if node.NamedChildCount() > 0 {
			es.resolveTypeNode(node.NamedChild(0), chain)
		}

	case "scoped_type_identifier":
		// 2. 嵌套类处理（左结合树规约）：
		// 按照从左到右语义，第一个命名子节点（左子树）向前递归推进
		if node.NamedChildCount() > 0 {
			es.resolveTypeNode(node.NamedChild(0), chain)
		}

		// 第二个命名子节点一定是当前层级的内部类标识符，转换为后置的 SegmentClass 压入链条
		if node.NamedChildCount() > 1 {
			right := node.NamedChild(1)
			rightName := helper.Clean(right.Utf8Text(*es.src))
			chain.Segments = append(chain.Segments, ExpressionSegment{
				Kind:    SegmentClass,
				Name:    rightName,
				ASTNode: right,
				RawText: right.Utf8Text(*es.src),
			})
		}

	case "type_identifier", "identifier":
		// 3. 基本标识符（递归终点/最左起点）：确立为 Head 起点
		raw := strings.TrimSpace(node.Utf8Text(*es.src))
		chain.Head = ExpressionHead{
			Type:    HeadIdent,
			Name:    helper.Clean(raw),
			ASTNode: node,
			RawText: raw,
		}

	default:
		// 异常兜底
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
		// 统一使用过滤命名节点的方式兜底提取简要名称
		namedCount := int(node.NamedChildCount())
		for i := 0; i < namedCount; i++ {
			child := node.NamedChild(uint(i))
			ck := child.Kind()
			if ck == "scoped_type_identifier" || ck == "type_identifier" || ck == "generic_type" {
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
