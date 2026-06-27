package resolver

import (
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
func (r *NodeContextResolver) ResolveContext(actionType string, node *sitter.Node) *Result {
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
	case "USE":
		return r.resolveUseContext(node)
	case "ASSIGN":
		return r.resolveAssignContext(node)
	case "CALL":
		return r.resolveCallContext(node)
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
	// 复用helper中的FindNearestKind方法
	assignExpr := helper.FindNearestKind(node, "assignment_expression")
	if assignExpr == nil {
		return r.resolveGenericContext(node)
	}

	leftNode := assignExpr.ChildByFieldName("left")
	if leftNode == nil {
		return r.resolveGenericContext(node)
	}

	ctxNode := r.findOuterChainNode(leftNode)
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
		if targetNode != nil {
			// 复用helper.GetNodeContent
			return helper.GetNodeContent(targetNode, *src)
		}
		return ""
	default:
		if contextNode != nil {
			// 复用helper.GetNodeContent
			return helper.GetNodeContent(contextNode, *src)
		}
		return ""
	}
}
