package resolver

import (
	"strings"

	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// NodeContextResolver 将「关系类型 + 动作节点」统一为双轨 ContextResult{ExpressNode, ContextNode}。
// 消除不同 relType 的分叉处理，让后续 enricher 同时拿到动作发生处与语义容器。
type NodeContextResolver struct{}

// NewNodeContextResolver 创建 Context 提取器。
func NewNodeContextResolver() *NodeContextResolver { return &NodeContextResolver{} }

// Resolve 按 relType 分发到对应提取策略。
func (r *NodeContextResolver) Resolve(node coreast.Node, relType model.DependencyType) ContextResult {
	switch relType {
	case model.Create:
		return r.resolveForCreate(node)
	case model.Throw:
		return r.resolveForThrow(node)
	case model.Cast:
		return r.resolveForCast(node)
	case model.Use, model.Assign, model.Call:
		return r.resolveTwoTrackPipeline(node)
	}
	return ContextResult{ExpressNode: node, ContextNode: node}
}

// resolveTwoTrackPipeline Use/Assign/Call：整链化向上爬升 -> 语义微调 -> 宏观边界。
func (r *NodeContextResolver) resolveTwoTrackPipeline(node coreast.Node) ContextResult {
	if node == nil {
		return ContextResult{}
	}

	express := r.findCoreChain(node)
	express = r.specializeSemanticAction(express)
	ctxNode := r.findOuterBoundary(express)

	return ContextResult{ExpressNode: express, ContextNode: ctxNode}
}

// findCoreChain 向上爬升定位核心表达式链（method_invocation/field_access/identifier 等）。
// 关键：若起点是 method_invocation/field_access 的 name 标识符（如 user.getName() 的 getName），
// 必须继续爬到外层调用/访问节点，否则会丢失接收者（receiver）导致链退化为裸标识符。
func (r *NodeContextResolver) findCoreChain(node coreast.Node) coreast.Node {
	if node == nil {
		return nil
	}
	curr := node
	for curr != nil {
		k := curr.Kind()
		switch k {
		case "method_invocation", "field_access", "object_creation_expression",
			"method_reference", "array_access", "update_expression", "assignment_expression",
			"explicit_constructor_invocation":
			return curr
		case "class_literal":
			// class literal（Xxx.class）是类型引用终点：不被外层 method_invocation 吸收，
			// 否则 SpringUtils.getBean(TaskDaoImpl.class) 的实参会被爬升成外层方法调用，
			// class_lit_atom 解析出的目标变 METHOD 而非类型。
			// 注意：Xxx.class.getCanonicalName() 的 CALL 由 call_target 单独捕获（从 name
			// identifier 爬升到 method_invocation，不经此分支），互不冲突。
			return curr
		case "identifier":
			// identifier 只有在不是 method_invocation/field_access 的 name 时才作为终点；
			// 若作为接收者链的 name（如 getName 之于 user.getName()），继续向上爬到调用节点。
			// 注意：field_access 的字段名是 field 字段而非 name（见 expression_segmenter.go），
			// 必须一并检查，否则 this.field 的 field 标识符会退化为裸标识符，
			// 被作用域链误解析为同名参数/局部变量（字段遮蔽 bug）。
			// 用字节区间比较（Equal 依赖 tree-sitter 指针，不同 Wrap 实例可能不相等）。
			p := curr.Parent()
			if p != nil && (p.Kind() == "method_invocation" || p.Kind() == "field_access" || p.Kind() == "method_reference") {
				nameNode := p.ChildByFieldName("name")
				if nameNode == nil && p.Kind() == "field_access" {
					nameNode = p.ChildByFieldName("field")
				}
				if nameNode != nil &&
					nameNode.StartByte() == curr.StartByte() && nameNode.EndByte() == curr.EndByte() {
					curr = p
					continue
				}
			}
			return curr
		}
		// 遇到语句边界即停。
		if strings.HasSuffix(k, "_statement") || k == "program" {
			break
		}
		curr = curr.Parent()
	}
	return node
}

// specializeSemanticAction 语义微调：若核心是 assignment/update，取其左值标识符。
func (r *NodeContextResolver) specializeSemanticAction(node coreast.Node) coreast.Node {
	if node == nil {
		return nil
	}
	switch node.Kind() {
	case "assignment_expression":
		if left := node.ChildByFieldName("left"); left != nil {
			return r.digLeftValue(left)
		}
	case "update_expression":
		if node.NamedChildCount() > 0 {
			return r.digLeftValue(node.NamedChild(0))
		}
	}
	return node
}

// digLeftValue 从赋值左侧提取最深层的标识符（field_access.field / array_access.array / identifier）。
func (r *NodeContextResolver) digLeftValue(node coreast.Node) coreast.Node {
	if node == nil {
		return nil
	}
	switch node.Kind() {
	case "field_access":
		if f := node.ChildByFieldName("field"); f != nil {
			return f
		}
	case "array_access":
		if a := node.ChildByFieldName("array"); a != nil {
			return a
		}
	}
	return node
}

// findOuterBoundary 找到动作的宏观边界（语句容器）。
func (r *NodeContextResolver) findOuterBoundary(node coreast.Node) coreast.Node {
	if node == nil {
		return nil
	}
	curr := node.Parent()
	for curr != nil {
		k := curr.Kind()
		if strings.HasSuffix(k, "_statement") || k == "variable_declarator" ||
			k == "enhanced_for_statement" || k == "lambda_expression" || k == "field_declaration" ||
			k == "formal_parameter" {
			return curr
		}
		curr = curr.Parent()
	}
	return node
}

// resolveForCreate 定位 new 表达式。
func (r *NodeContextResolver) resolveForCreate(node coreast.Node) ContextResult {
	if node == nil {
		return ContextResult{}
	}
	if node.Kind() == "object_creation_expression" || node.Kind() == "array_creation_expression" {
		return ContextResult{ExpressNode: node, ContextNode: r.findOuterBoundary(node)}
	}
	// 向上查找创建表达式。
	curr := node
	for curr != nil {
		if curr.Kind() == "object_creation_expression" || curr.Kind() == "array_creation_expression" {
			return ContextResult{ExpressNode: curr, ContextNode: r.findOuterBoundary(curr)}
		}
		curr = curr.Parent()
	}
	return ContextResult{ExpressNode: node, ContextNode: r.findOuterBoundary(node)}
}

// resolveForThrow 定位被抛出的异常表达式。
func (r *NodeContextResolver) resolveForThrow(node coreast.Node) ContextResult {
	if node == nil {
		return ContextResult{}
	}
	if node.Kind() == "throw_statement" {
		ctx := node
		if node.NamedChildCount() > 0 {
			return ContextResult{ExpressNode: node.NamedChild(0), ContextNode: ctx}
		}
	}
	return ContextResult{ExpressNode: node, ContextNode: node}
}

// resolveForCast 定位强转表达式。
func (r *NodeContextResolver) resolveForCast(node coreast.Node) ContextResult {
	if node == nil {
		return ContextResult{}
	}
	if node.Kind() == "cast_expression" || node.Kind() == "instanceof_expression" {
		return ContextResult{ExpressNode: node, ContextNode: r.findOuterBoundary(node)}
	}
	return ContextResult{ExpressNode: node, ContextNode: r.findOuterBoundary(node)}
}
