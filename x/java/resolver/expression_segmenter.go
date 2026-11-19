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
		// C-7 限定 this：X.this 是「类型名 X 限定的 this 引用」，不是字段访问。
		// 常见于匿名类/内部类内调用外层实例成员（InstanceSupplier.this.get()）。
		// 直接产出 HeadThis（Name 携带限定类型名 X），使后续方法/字段段在 X 的
		// 实例成员上解析（instanceCtx=true），避免虚增构造器 CALL + 丢失真实方法 CALL。
		if field.Utf8Text(es.src) == "this" && obj.Kind() == "identifier" {
			return &ExpressionHead{
				Type:    HeadThis,
				Name:    obj.Utf8Text(es.src),
				ASTNode: node,
				RawText: node.Utf8Text(es.src),
			}, nil
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
		// 注意：tree-sitter-java 的 method_reference 不提供 object/name 字段
		// （子节点是 [接收者, "::", 可选的 type_arguments, 名称]），必须按子节点手工切分。
		// 否则整串 "b::getSomeField" 会退化为裸标识符，被解析成同名变量/字段，
		// 产出 target 为 VARIABLE/FIELD 的伪 CALL。
		if receiver, name := es.splitMethodRef(node); receiver != nil && name != nil {
			head, segs := es.split(receiver)
			return head, append(segs, ExpressionSegment{
				Kind:        SegmentMethod,
				Name:        name.Utf8Text(es.src),
				ASTNode:     name,
				RawText:     name.Utf8Text(es.src),
				IsMethodRef: true,
			})
		}
		// 构造引用 Type::new：tree-sitter 不为 `::new` 提供 object/name 字段
		// （子节点是 identifier / :: / new），需手工取 `::` 前的接收者作为类型头。
		// 否则会退化成裸标识符 "Type::new"，解析失败后被 align 补成 "类QN()" 悬空。
		if refType := es.methodRefCtorType(node); refType != nil {
			// 用 HeadNewExpr 而非 HeadType：构造引用的 CALL 目标应是真实构造器元素
			// （`类QN.类名(参数)`），与 `new Type(...)` 走同一条构造器消解路径。
			return &ExpressionHead{
				Type:    HeadNewExpr,
				Name:    refType.Utf8Text(es.src),
				ASTNode: refType,
				RawText: refType.Utf8Text(es.src),
			}, nil
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

	case "class_literal":
		// Xxx.class 类字面量：头部解析为类型元素（HeadClassLit），
		// 但后续段（.getCanonicalName() 等）应在 java.lang.Class 上解析成员（见 resolveHead）。
		// 目标为 _unannotated_type 子节点类型名。
		if node.NamedChildCount() > 0 {
			typeNode := node.NamedChild(0)
			return &ExpressionHead{
				Type:    HeadClassLit,
				Name:    typeNode.Utf8Text(es.src),
				ASTNode: typeNode,
				RawText: typeNode.Utf8Text(es.src),
			}, nil
		}
		return nil, nil

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

// splitMethodRef 按子节点切分方法引用 `receiver::name`（可带 type_arguments）。
// tree-sitter-java 的 method_reference 无 object/name 字段，子节点为
// [接收者, "::", 可选 type_arguments, 名称]。非 `::new` 且结构完整时返回接收者与名称节点。
func (es *ExpressionSegmenter) splitMethodRef(node coreast.Node) (receiver, name coreast.Node) {
	var sepIdx = -1
	for i := 0; i < node.ChildCount(); i++ {
		if node.Child(i).Kind() == "::" {
			sepIdx = i
			break
		}
	}
	if sepIdx <= 0 {
		return nil, nil
	}
	// `::new` 是构造引用，走 methodRefCtorType 分支。
	for i := sepIdx + 1; i < node.ChildCount(); i++ {
		if node.Child(i).Kind() == "new" {
			return nil, nil
		}
	}
	// 接收者：`::` 之前最后一个具名子节点（可能是 identifier / field_access / this / super / 类型名）。
	for i := sepIdx - 1; i >= 0; i-- {
		if c := node.Child(i); c.IsNamed() {
			receiver = c
			break
		}
	}
	// 名称：`::` 之后跳过 type_arguments/type_identifier 等泛型前缀，取第一个具名节点。
	for i := sepIdx + 1; i < node.ChildCount(); i++ {
		c := node.Child(i)
		switch c.Kind() {
		case "type_arguments", "type_identifier", "scoped_type_identifier":
			continue
		}
		if c.IsNamed() {
			name = c
			break
		}
	}
	if receiver == nil || name == nil {
		return nil, nil
	}
	return receiver, name
}

// methodRefCtorType 提取构造引用 `Type::new` 的接收者类型节点。
// tree-sitter 对 `::new` 不提供 object/name 字段，子节点为 identifier / :: / new，
// 因此取 `::` 之前的第一个具名子节点作为类型。非 `::new` 形态返回 nil。
func (es *ExpressionSegmenter) methodRefCtorType(node coreast.Node) coreast.Node {
	isCtorRef := false
	for i := 0; i < node.ChildCount(); i++ {
		if node.Child(i).Kind() == "new" {
			isCtorRef = true
			break
		}
	}
	if !isCtorRef {
		return nil
	}
	for i := 0; i < node.ChildCount(); i++ {
		c := node.Child(i)
		if c.Kind() == "::" {
			break
		}
		if c.IsNamed() {
			return c
		}
	}
	return nil
}
