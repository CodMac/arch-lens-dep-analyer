package resolver

import (
	"strings"

	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// ExpressionSegmenter 将物理表达式树拉平为 Head + Segments 符号链。
// 按 relType + AST kind 确定性分流，保证可复现。
type ExpressionSegmenter struct {
	src []byte
}

// NewExpressionSegmenter 创建拉平器。
func NewExpressionSegmenter(src []byte) *ExpressionSegmenter {
	return &ExpressionSegmenter{src: src}
}

// Segment 将动作节点拉平为符号链。
func (es *ExpressionSegmenter) Segment(node coreast.Node, relType model.DependencyType) *ExpressionChain {
	if node == nil {
		return nil
	}

	kind := node.Kind()

	// 1. Create / 对象创建 -> HeadNewExpr 直接返回。
	if relType == model.Create || kind == "object_creation_expression" {
		return &ExpressionChain{Head: es.headForCreation(node)}
	}

	// 2. Cast / 强转 / instanceof -> HeadCastExpr。
	if relType == model.Cast || kind == "cast_expression" || kind == "instanceof_expression" {
		return &ExpressionChain{Head: es.headForCast(node)}
	}

	// 3. 其他：按 kind 递归拆段。
	head, segments := es.split(node)
	if head == nil {
		return nil
	}
	return &ExpressionChain{Head: *head, Segments: segments}
}

// split 递归拆解表达式，返回头部与段列表。
func (es *ExpressionSegmenter) split(node coreast.Node) (*ExpressionHead, []ExpressionSegment) {
	if node == nil {
		return nil, nil
	}

	switch node.Kind() {
	case "field_access":
		obj := node.ChildByFieldName("object")
		field := node.ChildByFieldName("field")
		if obj == nil || field == nil {
			return nil, nil
		}
		head, segs := es.split(obj)
		return head, append(segs, ExpressionSegment{
			Kind:    SegmentField,
			Name:    field.Utf8Text(es.src),
			ASTNode: field,
			RawText: field.Utf8Text(es.src),
		})

	case "method_invocation":
		name := node.ChildByFieldName("name")
		object := node.ChildByFieldName("object")
		if name == nil {
			return nil, nil
		}
		seg := ExpressionSegment{
			Kind:    SegmentMethod,
			Name:    name.Utf8Text(es.src),
			ASTNode: name,
			RawText: name.Utf8Text(es.src),
		}
		if object == nil {
			// 无接收者 -> 隐式 this 调用
			return &ExpressionHead{
				Type:    HeadImplicitMethod,
				Name:    seg.Name,
				ASTNode: name,
				RawText: node.Utf8Text(es.src),
			}, []ExpressionSegment{}
		}
		head, segs := es.split(object)
		return head, append(segs, seg)

	case "method_reference":
		// 形如 System.out::println / this::foo / Type::new
		receiver := node.ChildByFieldName("object")
		name := node.ChildByFieldName("name")
		if receiver != nil && name != nil {
			head, segs := es.split(receiver)
			return head, append(segs, ExpressionSegment{
				Kind:    SegmentMethod,
				Name:    name.Utf8Text(es.src),
				ASTNode: name,
				RawText: name.Utf8Text(es.src),
			})
		}
		// 无法拆解：整体作为 Ident
		return &ExpressionHead{
			Type:    HeadIdent,
			Name:    node.Utf8Text(es.src),
			ASTNode: node,
			RawText: node.Utf8Text(es.src),
		}, nil

	case "array_access":
		arr := node.ChildByFieldName("array")
		index := node.ChildByFieldName("index")
		if arr == nil {
			return nil, nil
		}
		head, segs := es.split(arr)
		seg := ExpressionSegment{
			Kind:    SegmentArray,
			ASTNode: node,
			RawText: node.Utf8Text(es.src),
		}
		if index != nil {
			seg.Name = index.Utf8Text(es.src)
		}
		return head, append(segs, seg)

	case "parenthesized_expression":
		// 剥离括号
		if node.NamedChildCount() > 0 {
			return es.split(node.NamedChild(0))
		}
		return nil, nil

	case "cast_expression":
		return &ExpressionHead{
			Type:     HeadCastExpr,
			Name:     es.castTypeName(node),
			CastType: es.castTypeName(node),
			ASTNode:  node,
			RawText:  node.Utf8Text(es.src),
		}, nil

	case "instanceof_expression":
		right := node.ChildByFieldName("right")
		if right != nil {
			return &ExpressionHead{
				Type:     HeadCastExpr,
				Name:     right.Utf8Text(es.src),
				CastType: right.Utf8Text(es.src),
				ASTNode:  node,
				RawText:  node.Utf8Text(es.src),
			}, nil
		}
		return nil, nil

	case "object_creation_expression":
		// 作为链中接收者（如 new Runnable(){...}.run()）时，须携带构造类型名供后续解析。
		typeName := ""
		if t := node.ChildByFieldName("type"); t != nil {
			typeName = t.Utf8Text(es.src)
		}
		return &ExpressionHead{Type: HeadNewExpr, Name: typeName, ASTNode: node, RawText: node.Utf8Text(es.src)}, nil

	case "identifier":
		return &ExpressionHead{
			Type:    HeadIdent,
			Name:    node.Utf8Text(es.src),
			ASTNode: node,
			RawText: node.Utf8Text(es.src),
		}, nil

	case "this":
		return &ExpressionHead{Type: HeadThis, Name: "this", ASTNode: node, RawText: "this"}, nil

	case "super":
		return &ExpressionHead{Type: HeadSuper, Name: "super", ASTNode: node, RawText: "super"}, nil

	case "explicit_constructor_invocation":
		// super() / this() / TypeName(...)：目标引用来自 constructor 字段。
		ctor := node.ChildByFieldName("constructor")
		ref := ""
		if ctor != nil {
			ref = ctor.Utf8Text(es.src)
		}
		return &ExpressionHead{
			Type:    HeadCtorInvoke,
			Name:    ref,
			CtorRef: ref,
			ASTNode: node,
			RawText: node.Utf8Text(es.src),
		}, nil
	}

	// 兜底：无法识别时整体作为 Ident
	return &ExpressionHead{
		Type:    HeadIdent,
		Name:    node.Utf8Text(es.src),
		ASTNode: node,
		RawText: node.Utf8Text(es.src),
	}, nil
}

// headForCreation 构造对象创建的头部。
func (es *ExpressionSegmenter) headForCreation(node coreast.Node) ExpressionHead {
	typeNode := node.ChildByFieldName("type")
	typeName := ""
	if typeNode != nil {
		typeName = typeNode.Utf8Text(es.src)
	}
	return ExpressionHead{
		Type:    HeadNewExpr,
		Name:    typeName,
		ASTNode: node,
		RawText: node.Utf8Text(es.src),
	}
}

// headForCast 构造强转的头部。
func (es *ExpressionSegmenter) headForCast(node coreast.Node) ExpressionHead {
	castType := es.castTypeName(node)
	return ExpressionHead{
		Type:     HeadCastExpr,
		Name:     castType,
		CastType: castType,
		ASTNode:  node,
		RawText:  node.Utf8Text(es.src),
	}
}

// castTypeName 提取强转目标类型名。
func (es *ExpressionSegmenter) castTypeName(node coreast.Node) string {
	switch node.Kind() {
	case "cast_expression":
		if t := node.ChildByFieldName("type"); t != nil {
			return t.Utf8Text(es.src)
		}
	case "instanceof_expression":
		if r := node.ChildByFieldName("right"); r != nil {
			return r.Utf8Text(es.src)
		}
	}
	return strings.TrimSpace(node.Utf8Text(es.src))
}
