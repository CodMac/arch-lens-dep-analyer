package resolver

import (
	"fmt"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// NodeContextResolver 节点上下文解析器
type NodeContextResolver struct{}

// NewNodeContextResolver 创建节点上下文解析器
func NewNodeContextResolver() *NodeContextResolver {
	return &NodeContextResolver{}
}

// Result 上下文解析结果
type Result struct {
	ContextNode *sitter.Node
	ContextKind string
	IsChain     bool
}

// ResolveContext 统一的上下文解析入口
func (r *NodeContextResolver) ResolveContext(actionType model.DependencyType, node *sitter.Node) *Result {
	if node == nil {
		return nil
	}

	if r.isChainNode(node) {
		return &Result{
			ContextNode: node,
			ContextKind: node.Kind(),
			IsChain:     true,
		}
	}

	switch actionType {
	case model.Use:
		return r.resolveUseContext(node)
	case model.Assign:
		return r.resolveAssignContext(node)
	case model.Call:
		return r.resolveCallContext(node)
	case model.Create:
		return r.resolveCreateContext(node)
	case model.Cast:
		return r.resolveCastContext(node)
	case model.Throw:
		return r.resolveThrowContext(node)
	default:
		return r.resolveGenericContext(node)
	}
}

// resolveUseContext 解析USE关系的上下文
func (r *NodeContextResolver) resolveUseContext(node *sitter.Node) *Result {
	ctxNode := r.findOuterChainNode(node)
	if ctxNode == nil {
		return &Result{
			ContextNode: node,
			ContextKind: node.Kind(),
			IsChain:     false,
		}
	}

	return &Result{
		ContextNode: ctxNode,
		ContextKind: ctxNode.Kind(),
		IsChain:     r.isChainNode(ctxNode),
	}
}

// resolveAssignContext 解析ASSIGN关系的上下文
func (r *NodeContextResolver) resolveAssignContext(node *sitter.Node) *Result {
	assignExpr := helper.FindNearestKind(node, "assignment_expression")
	if assignExpr == nil {
		ctx := r.resolveGenericContext(node)
		fmt.Printf("[DEBUG ASSIGN] assignExpr == nil, then resolveGenericContext. \n ctx.Kind()=%s, leftNode.Kind()=%s\n", node.Kind(), ctx.ContextKind)
		return ctx
	}

	leftNode := assignExpr.ChildByFieldName("left")
	if leftNode == nil {
		ctx := r.resolveGenericContext(node)
		fmt.Printf("[DEBUG ASSIGN] leftNode == nil, then resolveGenericContext. \n ctx.Kind()=%s, leftNode.Kind()=%s\n", node.Kind(), ctx.ContextKind)
		return ctx
	}

	// DEBUG: 打印调试信息
	fmt.Printf("[DEBUG ASSIGN] capturedNode.Kind()=%s, leftNode.Kind()=%s\n", node.Kind(), leftNode.Kind())
	ctxNode := r.findAssignLeftChainNodeFromIdentifier(node, leftNode)

	// DEBUG: 打印结果
	fmt.Printf("[DEBUG ASSIGN] ctxNode=%v\n", ctxNode != nil)
	if ctxNode != nil {
		fmt.Printf("[DEBUG ASSIGN] ctxNode.Kind()=%s\n", ctxNode.Kind())
	}

	if ctxNode == nil {
		return &Result{
			ContextNode: leftNode,
			ContextKind: leftNode.Kind(),
			IsChain:     false,
		}
	}

	return &Result{
		ContextNode: ctxNode,
		ContextKind: ctxNode.Kind(),
		IsChain:     r.isChainNode(ctxNode),
	}
}

// resolveCallContext 解析CALL关系的上下文
func (r *NodeContextResolver) resolveCallContext(node *sitter.Node) *Result {
	if r.isInvocationNode(node) {
		return &Result{
			ContextNode: node,
			ContextKind: node.Kind(),
			IsChain:     true,
		}
	}

	ctxNode := r.findCallContextNode(node)
	if ctxNode == nil {
		return &Result{
			ContextNode: node,
			ContextKind: node.Kind(),
			IsChain:     false,
		}
	}

	return &Result{
		ContextNode: ctxNode,
		ContextKind: ctxNode.Kind(),
		IsChain:     true,
	}
}

// resolveCreateContext 解析CREATE关系的上下文
func (r *NodeContextResolver) resolveCreateContext(node *sitter.Node) *Result {
	ctxNode := r.findCreateContextNode(node)
	if ctxNode == nil {
		return &Result{
			ContextNode: node,
			ContextKind: node.Kind(),
			IsChain:     false,
		}
	}

	return &Result{
		ContextNode: ctxNode,
		ContextKind: ctxNode.Kind(),
		IsChain:     r.isChainNode(ctxNode),
	}
}

// resolveCastContext 解析CAST关系的上下文
func (r *NodeContextResolver) resolveCastContext(node *sitter.Node) *Result {
	ctxNode := r.findCastContextNode(node)
	if ctxNode == nil {
		return &Result{
			ContextNode: node,
			ContextKind: node.Kind(),
			IsChain:     false,
		}
	}

	return &Result{
		ContextNode: ctxNode,
		ContextKind: ctxNode.Kind(),
		IsChain:     false,
	}
}

// resolveThrowContext 解析THROW关系的上下文
func (r *NodeContextResolver) resolveThrowContext(node *sitter.Node) *Result {
	ctxNode := r.findThrowContextNode(node)
	if ctxNode == nil {
		return &Result{
			ContextNode: node,
			ContextKind: node.Kind(),
			IsChain:     false,
		}
	}

	return &Result{
		ContextNode: ctxNode,
		ContextKind: ctxNode.Kind(),
		IsChain:     false,
	}
}

// resolveGenericContext 通用上下文解析
func (r *NodeContextResolver) resolveGenericContext(node *sitter.Node) *Result {
	ctxNode := r.findOuterChainNode(node)
	if ctxNode == nil {
		return &Result{
			ContextNode: node,
			ContextKind: node.Kind(),
			IsChain:     false,
		}
	}

	return &Result{
		ContextNode: ctxNode,
		ContextKind: ctxNode.Kind(),
		IsChain:     r.isChainNode(ctxNode),
	}
}

// findOuterChainNode 向上查找最外层的链式节点
func (r *NodeContextResolver) findOuterChainNode(node *sitter.Node) *sitter.Node {
	parent := node.Parent()
	var outerChainNode *sitter.Node

	for parent != nil {
		kind := parent.Kind()

		switch kind {
		case "field_access":
			if fieldNode := parent.ChildByFieldName("field"); fieldNode != nil && fieldNode.Id() == node.Id() {
				outerChainNode = parent
				node = parent
			} else if r.isNodeContained(parent, node) {
				node = parent
			} else {
				return outerChainNode
			}

		case "method_invocation":
			if nameNode := parent.ChildByFieldName("name"); nameNode != nil && nameNode.Id() == node.Id() {
				return outerChainNode
			} else if objNode := parent.ChildByFieldName("object"); objNode != nil {
				if r.isNodeContained(objNode, node) {
					outerChainNode = parent
					node = parent
				} else {
					return outerChainNode
				}
			} else {
				return outerChainNode
			}

		case "array_access":
			if objNode := parent.ChildByFieldName("object"); objNode != nil {
				if r.isNodeContained(objNode, node) {
					outerChainNode = parent
					node = parent
				} else {
					return outerChainNode
				}
			} else {
				return outerChainNode
			}

		case "assignment_expression", "binary_expression", "ternary_expression":
			node = parent

		default:
			if r.canContainChain(parent) {
				node = parent
			} else {
				return outerChainNode
			}
		}

		parent = node.Parent()
	}

	return outerChainNode
}

// findCallContextNode 为调用查找上下文
func (r *NodeContextResolver) findCallContextNode(node *sitter.Node) *sitter.Node {
	parent := node.Parent()
	for parent != nil {
		if r.isInvocationNode(parent) {
			if r.isPartOfCallChain(parent, node) {
				return r.findOuterChainNode(parent)
			}
			return parent
		}
		parent = parent.Parent()
	}
	return nil
}

// findCreateContextNode 为创建对象查找上下文
func (r *NodeContextResolver) findCreateContextNode(node *sitter.Node) *sitter.Node {
	parent := node.Parent()
	for parent != nil {
		kind := parent.Kind()

		switch kind {
		case "object_creation_expression", "array_creation_expression":
			return parent
		case "type", "generic_type", "argument_list", "arguments", "inferred_parameters":
			parent = parent.Parent()
		default:
			if r.canContainChain(parent) {
				parent = parent.Parent()
			} else {
				return nil
			}
		}
	}
	return nil
}

// findCastContextNode 为类型转换查找上下文
func (r *NodeContextResolver) findCastContextNode(node *sitter.Node) *sitter.Node {
	parent := node.Parent()
	for parent != nil {
		kind := parent.Kind()

		switch kind {
		case "cast_expression", "instanceof_expression":
			return parent
		case "argument_list", "arguments", "parenthesized_expression":
			parent = parent.Parent()
		default:
			if r.canContainChain(parent) {
				parent = parent.Parent()
			} else {
				return nil
			}
		}
	}
	return nil
}

// findAssignLeftChainNode 专门为ASSIGN左值查找链式上下文
func (r *NodeContextResolver) findAssignLeftChainNode(leftNode *sitter.Node) *sitter.Node {
	if leftNode == nil {
		return nil
	}

	leftValueNode := leftNode
	parent := leftValueNode.Parent()
	var outermostChain *sitter.Node

	for parent != nil {
		kind := parent.Kind()

		switch kind {
		case "assignment_expression":
			return outermostChain

		case "field_access":
			fieldNode := parent.ChildByFieldName("field")
			if fieldNode != nil {
				if fieldNode.Id() == leftValueNode.Id() || outermostChain == fieldNode {
					outermostChain = parent
					leftValueNode = parent
					parent = leftValueNode.Parent()
					continue
				}
			}
			return outermostChain

		case "method_invocation":
			objectNode := parent.ChildByFieldName("object")
			if objectNode != nil && objectNode.Id() == leftValueNode.Id() {
				outermostChain = parent
				leftValueNode = parent
				parent = leftValueNode.Parent()
				continue
			}
			nameNode := parent.ChildByFieldName("name")
			if nameNode != nil && nameNode.Id() == leftValueNode.Id() {
				outermostChain = parent
				leftValueNode = parent
				parent = leftValueNode.Parent()
				continue
			}
			return outermostChain

		case "array_access":
			objectNode := parent.ChildByFieldName("object")
			if objectNode != nil && r.isNodeContained(objectNode, leftValueNode) {
				outermostChain = parent
				leftValueNode = parent
				parent = leftValueNode.Parent()
				continue
			}
			return outermostChain

		default:
			if r.canContainChain(parent) {
				parent = parent.Parent()
			} else {
				return outermostChain
			}
		}
	}

	return outermostChain
}

// findAssignLeftChainNodeFromIdentifier 从捕获的标识符开始查找完整的链式左值
func (r *NodeContextResolver) findAssignLeftChainNodeFromIdentifier(capturedNode, leftNode *sitter.Node) *sitter.Node {
	if capturedNode == nil {
		return nil
	}

	// DEBUG: 初始调试信息
	fmt.Printf("[DEBUG findAssignLeftChainNodeFromIdentifier] START: capturedNode.Kind()=%s, leftNode.Kind()=%s\n", capturedNode.Kind(), leftNode.Kind())

	// 如果leftNode本身就是一个复杂的表达式，直接返回
	if leftNode.Kind() != "identifier" && leftNode.Kind() != "type_identifier" {
		// fmt.Printf("[DEBUG findAssignLeftChainNodeFromIdentifier] leftNode is complex: %s, directly returning\n", leftNode.Kind())
		return leftNode
	}

	// 从捕获的节点开始向上查找
	currentNode := capturedNode
	parent := currentNode.Parent()
	var outermostChain *sitter.Node

	// fmt.Printf("[DEBUG findAssignLeftChainNodeFromIdentifier] Starting traversal from capturedNode\n")

	for parent != nil {
		kind := parent.Kind()

		// fmt.Printf("[DEBUG findAssignLeftChainNodeFromIdentifier] parent.Kind()=%s, currentNode.Kind()=%s\n", kind, currentNode.Kind())

		switch kind {
		case "assignment_expression":
			// fmt.Printf("[DEBUG findAssignLeftChainNodeFromIdentifier] Reached assignment_expression, returning outermostChain=%v\n", outermostChain != nil)
			return outermostChain

		case "field_access":
			fieldNode := parent.ChildByFieldName("field")

			// Check if current node is the field node (most direct case)
			if fieldNode != nil && fieldNode.Id() == currentNode.Id() {
				// fmt.Printf("[DEBUG findAssignLeftChainNodeFromIdentifier] Found field_access where current is field, updating outermostChain\n")
				outermostChain = parent
				currentNode = parent
				parent = currentNode.Parent()
				continue
			}

			// Check if current node is part of a chain (previous field_access)
			if outermostChain != nil && outermostChain.Id() == currentNode.Id() {
				// fmt.Printf("[DEBUG findAssignLeftChainNodeFromIdentifier] Found chain case, updating outermostChain\n")
				outermostChain = parent
				currentNode = parent
				parent = currentNode.Parent()
				continue
			}

			// Check if current node is inside the object part (recursive chain)
			objNode := parent.ChildByFieldName("object")
			if objNode != nil && objNode.Id() == currentNode.Id() {
				// fmt.Printf("[DEBUG findAssignLeftChainNodeFromIdentifier] Found object chain case, updating outermostChain\n")
				outermostChain = parent
				currentNode = parent
				parent = currentNode.Parent()
				continue
			}

			// fmt.Printf("[DEBUG findAssignLeftChainNodeFromIdentifier] Not a field_access chain, returning current outermostChain=%v\n", outermostChain != nil)
			return outermostChain

		case "method_invocation":
			objectNode := parent.ChildByFieldName("object")
			if objectNode != nil && objectNode.Id() == currentNode.Id() {
				outermostChain = parent
				currentNode = parent
				parent = currentNode.Parent()
				continue
			}
			nameNode := parent.ChildByFieldName("name")
			if nameNode != nil && nameNode.Id() == currentNode.Id() {
				outermostChain = parent
				currentNode = parent
				parent = currentNode.Parent()
				continue
			}
			return outermostChain

		case "array_access":
			objectNode := parent.ChildByFieldName("object")
			if objectNode != nil && r.isNodeContained(objectNode, currentNode) {
				outermostChain = parent
				currentNode = parent
				parent = currentNode.Parent()
				continue
			}
			return outermostChain

		default:
			if r.canContainChain(parent) {
				parent = parent.Parent()
			} else {
				return outermostChain
			}
		}
	}

	// fmt.Printf("[DEBUG findAssignLeftChainNodeFromIdentifier] Reached end of traversal, returning outermostChain=%v\n", outermostChain != nil)
	return outermostChain
}

// isPartOfAssignLeft 判断节点是否是赋值左值链的一部分
func (r *NodeContextResolver) isPartOfAssignLeft(chainNode, targetNode *sitter.Node) bool {
	if chainNode == nil || targetNode == nil {
		return false
	}

	switch chainNode.Kind() {
	case "field_access":
		fieldNode := chainNode.ChildByFieldName("field")
		return fieldNode != nil && (fieldNode.Id() == targetNode.Id() || r.isNodeContained(fieldNode, targetNode))

	case "method_invocation":
		objectNode := chainNode.ChildByFieldName("object")
		if objectNode != nil && objectNode.Id() == targetNode.Id() {
			return true
		}
		nameNode := chainNode.ChildByFieldName("name")
		return nameNode != nil && nameNode.Id() == targetNode.Id()

	case "array_access":
		objectNode := chainNode.ChildByFieldName("object")
		return objectNode != nil && r.isNodeContained(objectNode, targetNode)

	default:
		return false
	}
}

// findThrowContextNode 为抛出异常查找上下文
func (r *NodeContextResolver) findThrowContextNode(node *sitter.Node) *sitter.Node {
	parent := node.Parent()
	for parent != nil {
		kind := parent.Kind()

		if kind == "throw_statement" {
			return parent
		} else if r.canContainChain(parent) || kind == "object_creation_expression" || kind == "argument_list" || kind == "arguments" {
			parent = parent.Parent()
		} else {
			return nil
		}
	}
	return nil
}

// isChainNode 判断是否为链式节点
func (r *NodeContextResolver) isChainNode(node *sitter.Node) bool {
	switch node.Kind() {
	case "field_access", "method_invocation", "array_access":
		return true
	default:
		return false
	}
}

// isInvocationNode 判断是否为调用节点
func (r *NodeContextResolver) isInvocationNode(node *sitter.Node) bool {
	switch node.Kind() {
	case "method_invocation", "explicit_constructor_invocation", "object_creation_expression", "method_reference":
		return true
	default:
		return false
	}
}

// isPartOfCallChain 判断node是否是调用链的一部分
func (r *NodeContextResolver) isPartOfCallChain(invokeNode, node *sitter.Node) bool {
	objNode := invokeNode.ChildByFieldName("object")
	if objNode != nil && r.isNodeContained(objNode, node) {
		return true
	}

	args := invokeNode.ChildByFieldName("arguments")
	if args != nil && r.isNodeContained(args, node) {
		return true
	}

	typeArgs := invokeNode.ChildByFieldName("type_arguments")
	if typeArgs != nil && r.isNodeContained(typeArgs, node) {
		return true
	}

	return false
}

// canContainChain 判断节点是否可以包含链式调用
func (r *NodeContextResolver) canContainChain(node *sitter.Node) bool {
	switch node.Kind() {
	case "parenthesized_expression", "cast_expression", "binary_expression",
		"ternary_expression", "unary_expression", "update_expression":
		return true
	default:
		return false
	}
}

// isNodeContained 检查node是否被container包含
// 使用helper中的FindNamedChildOfType改进字段查找
func (r *NodeContextResolver) isNodeContained(container, node *sitter.Node) bool {
	if container == nil || node == nil {
		return false
	}

	if container.Id() == node.Id() {
		return true
	}

	// 检查直接父关系
	if node.Parent().Id() == container.Id() {
		return true
	}

	// 使用helper中的方法检查常见字段
	if r.isInCommonFields(container, node) {
		return true
	}

	return false
}

// isInCommonFields 检查node是否在container的常见字段中
// 集成helper.FindNamedChildOfType来改进查找
func (r *NodeContextResolver) isInCommonFields(container, node *sitter.Node) bool {
	// 使用helper方法查找命名字段类型
	identifiers := r.findNamedChildOfType(container, "identifier")
	typeIdentifiers := r.findNamedChildOfType(container, "type_identifier")

	// 合并所有可能的字段节点
	allNodes := append(identifiers, typeIdentifiers...)

	for _, fieldNode := range allNodes {
		if fieldNode != nil && (fieldNode.Id() == node.Id() || node.Parent().Id() == fieldNode.Id()) {
			return true
		}
	}

	// 还需要检查重要的字段对象
	fields := []string{"object", "field", "value", "left", "right",
		"condition", "consequence", "alternative", "arguments"}
	for _, fieldName := range fields {
		fieldNode := container.ChildByFieldName(fieldName)
		if fieldNode != nil && (fieldNode.Id() == node.Id() || node.Parent().Id() == fieldNode.Id()) {
			return true
		}
	}

	return false
}

// findNamedChildOfType 批量查找指定类型的命名子节点
// 优化了helper.FindNamedChildOfType的批量查找功能
func (r *NodeContextResolver) findNamedChildOfType(container *sitter.Node, nodeType string) []*sitter.Node {
	var result []*sitter.Node
	count := container.NamedChildCount()
	for i := uint(0); i < count; i++ {
		child := container.NamedChild(i)
		if child != nil && child.Kind() == nodeType {
			result = append(result, child)
		}
	}
	return result
}

// GetRawTextForAction 获取Action关系的原始文本（包级函数）
// 复用helper.GetNodeContent方法
func GetRawTextForAction(actionType string, targetNode, contextNode *sitter.Node, src *[]byte) string {
	if targetNode == nil && contextNode == nil {
		return ""
	}

	switch actionType {
	case "ASSIGN":
		if contextNode != nil {
			return helper.GetNodeContent(contextNode, *src)
		}
		if targetNode != nil {
			return helper.GetNodeContent(targetNode, *src)
		}
		return ""
	case "CREATE", "CALL", "THROW":
		if contextNode != nil {
			return helper.GetNodeContent(contextNode, *src)
		}
		if targetNode != nil {
			return helper.GetNodeContent(targetNode, *src)
		}
		return ""
	case "CAST":
		if contextNode != nil {
			return helper.GetNodeContent(contextNode, *src)
		}
		return ""
	default:
		if contextNode != nil {
			return helper.GetNodeContent(contextNode, *src)
		}
		return ""
	}
}
