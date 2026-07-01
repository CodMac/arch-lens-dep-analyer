package resolver

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper" // 引入你的 helper 包
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type NodeContextResolver struct {
	srcs *[]byte
}

func NewNodeContextResolver(fCtx *core.FileContext) *NodeContextResolver {
	return &NodeContextResolver{
		srcs: fCtx.SourceBytes,
	}
}

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

	if r.isChainExpression(node) {
		return r.buildResult(node)
	}

	switch actionType {
	case model.Use:
		return r.resolveForUse(node)
	case model.Assign:
		return r.resolveForAssign(node)
	case model.Call:
		return r.resolveForCall(node)
	case model.Create:
		return r.resolveForCreate(node)
	case model.Cast:
		return r.resolveForCast(node)
	case model.Throw:
		return r.resolveForThrow(node)
	default:
		return r.resolveForGeneric(node)
	}
}

// =============================================================================
// 解析层
// =============================================================================

func (r *NodeContextResolver) resolveForUse(node *sitter.Node) *Result {
	// 1. 优先通过特定的链式查找器向上爬到整串流式/字段调用的最外层边界 (如 this.fieldVar)
	if ctxNode := r.findOutermostChainExpression(node); ctxNode != nil {
		node = ctxNode
	}

	// 2. 向上追溯，收集具有更高统计和图依赖价值的“最外层非语句级表达式”
	var bestExpressionContext *sitter.Node
	parent := node.Parent()

	for parent != nil {
		kind := parent.Kind()

		// 命中具有图拓扑价值的表达式或特定容器白名单
		if kind == "binary_expression" ||
			kind == "ternary_expression" ||
			kind == "cast_expression" ||
			kind == "enhanced_for_statement" ||
			kind == "lambda_expression" ||
			kind == "method_invocation" ||
			kind == "array_access" {

			// 暂存当前找到的表达式上下文，并允许其继续往上探索更大的合法父容器
			bestExpressionContext = parent
		}

		// 显式允许穿透的语法噪音层（包含流式参数结构、修饰括号、变量声明槽）
		if r.canChainInKind(kind) ||
			kind == "argument_list" ||
			kind == "arguments" ||
			kind == "variable_declarator" ||
			kind == "parenthesized_expression" ||
			kind == "binary_expression" || // 允许二元运算向上穿透进入三元、方法等更大表达式
			kind == "array_access" { // 允许数组读取穿透到调用容器
			parent = parent.Parent()
		} else {
			// 撞上语句级或作用域边界（如 expression_statement, block, class_body），及时阻断
			break
		}
	}

	// 如果向上探索到了更宏观、更有意义的表达式上下文，优先返回它
	if bestExpressionContext != nil {
		return r.buildResult(bestExpressionContext)
	}

	// 降级保护
	return r.buildResult(node)
}

func (r *NodeContextResolver) resolveForAssign(node *sitter.Node) *Result {
	// 【复用 1】直接通过 helper 寻找最近的赋值表达式，代替原先繁琐的向上手写判断
	assignExpr := helper.FindNearestKind(node, "assignment_expression")
	if assignExpr == nil {
		return r.resolveForGeneric(node)
	}

	leftNode := assignExpr.ChildByFieldName("left")
	if leftNode == nil {
		return r.resolveForGeneric(node)
	}

	if leftNode.Kind() != "identifier" && leftNode.Kind() != "type_identifier" {
		if ctxNode := r.findChainedAssignLeft(node); ctxNode != nil {
			return r.buildResult(ctxNode)
		}
		return r.buildResult(leftNode)
	}
	return r.buildResult(leftNode)
}

func (r *NodeContextResolver) resolveForCall(node *sitter.Node) *Result {
	if r.isInvocationExpression(node) {
		return r.buildResult(node)
	}

	parent := node.Parent()
	for parent != nil {
		if r.isInvocationExpression(parent) {
			if r.isPartOfInvocation(parent, node) {
				if outer := r.findOutermostChainExpression(parent); outer != nil {
					return r.buildResult(outer)
				}
			}
			return r.buildResult(parent)
		}
		parent = parent.Parent()
	}
	return r.buildResult(node)
}

func (r *NodeContextResolver) resolveForCreate(node *sitter.Node) *Result {
	if createStmt := helper.FindNearestKind(node, "object_creation_expression", "array_creation_expression"); createStmt != nil {
		return r.buildResult(createStmt)
	}
	return r.buildResult(node)
}

func (r *NodeContextResolver) resolveForThrow(node *sitter.Node) *Result {
	if throwStmt := helper.FindNearestKind(node, "throw_statement"); throwStmt != nil {
		return r.buildResult(throwStmt)
	}
	return r.buildResult(node)
}

func (r *NodeContextResolver) resolveForCast(node *sitter.Node) *Result {
	if throwStmt := helper.FindNearestKind(node, "cast_expression", "instanceof_expression"); throwStmt != nil {
		return r.buildResult(throwStmt)
	}
	return r.buildResult(node)
}

func (r *NodeContextResolver) resolveUpstream(node *sitter.Node, targetKinds []string, allowChainOuter bool) *Result {
	parent := node.Parent()
	for parent != nil {
		kind := parent.Kind()
		if helper.Contains(targetKinds, kind) {
			if allowChainOuter {
				if outer := r.findOutermostChainExpression(parent); outer != nil {
					return r.buildResult(outer)
				}
			}
			return r.buildResult(parent)
		}

		if r.canChainInKind(kind) || kind == "argument_list" || kind == "arguments" ||
			kind == "type" || kind == "generic_type" || kind == "parenthesized_expression" {
			parent = parent.Parent()
		} else {
			break
		}
	}
	return r.buildResult(node)
}

func (r *NodeContextResolver) resolveForGeneric(node *sitter.Node) *Result {
	if ctxNode := r.findOutermostChainExpression(node); ctxNode != nil {
		return r.buildResult(ctxNode)
	}
	return r.buildResult(node)
}

// =============================================================================
// 查找层
// =============================================================================

func (r *NodeContextResolver) findOutermostChainExpression(node *sitter.Node) *sitter.Node {
	parent := node.Parent()
	var outerChainNode *sitter.Node

	for parent != nil {
		switch parent.Kind() {
		case "field_access":
			fieldNode := parent.ChildByFieldName("field")
			if fieldNode != nil && fieldNode.Id() == node.Id() {
				outerChainNode = parent
				node = parent
			} else if helper.IsNodeContained(parent.ChildByFieldName("object"), node) {
				node = parent
			} else {
				return outerChainNode
			}

		case "method_invocation":
			nameNode := parent.ChildByFieldName("name")
			if nameNode != nil && nameNode.Id() == node.Id() {
				return outerChainNode
			}
			objNode := parent.ChildByFieldName("object")
			if objNode != nil && helper.IsNodeContained(objNode, node) {
				outerChainNode = parent
				node = parent
			} else {
				return outerChainNode
			}

		case "array_access":
			if helper.IsNodeContained(parent.ChildByFieldName("object"), node) {
				outerChainNode = parent
				node = parent
			} else {
				return outerChainNode
			}

		case "assignment_expression", "binary_expression", "ternary_expression":
			node = parent

		default:
			if r.canChainInKind(parent.Kind()) {
				node = parent
			} else {
				return outerChainNode
			}
		}
		parent = node.Parent()
	}
	return outerChainNode
}

func (r *NodeContextResolver) findChainedAssignLeft(capturedNode *sitter.Node) *sitter.Node {
	curr := capturedNode
	parent := curr.Parent()
	var outermostChain *sitter.Node

	for parent != nil {
		kind := parent.Kind()
		if kind == "assignment_expression" {
			return outermostChain
		}

		switch kind {
		case "field_access":
			fieldNode := parent.ChildByFieldName("field")
			objNode := parent.ChildByFieldName("object")

			if (fieldNode != nil && fieldNode.Id() == curr.Id()) ||
				(outermostChain != nil && outermostChain.Id() == curr.Id()) ||
				(objNode != nil && objNode.Id() == curr.Id()) {
				outermostChain = parent
				curr = parent
			} else {
				return outermostChain
			}

		case "method_invocation":
			objNode := parent.ChildByFieldName("object")
			nameNode := parent.ChildByFieldName("name")
			if (objNode != nil && objNode.Id() == curr.Id()) || (nameNode != nil && nameNode.Id() == curr.Id()) {
				outermostChain = parent
				curr = parent
			} else {
				return outermostChain
			}

		case "array_access":
			if helper.IsNodeContained(parent.ChildByFieldName("object"), curr) {
				outermostChain = parent
				curr = parent
			} else {
				return outermostChain
			}

		default:
			// 【优化清洁】这里现在直接使用重构后的 canChainInKind 字符串匹配，安全拿掉临时补丁
			if r.canChainInKind(kind) {
				curr = parent
			} else {
				return outermostChain
			}
		}
		parent = curr.Parent()
	}
	return outermostChain
}

// =============================================================================
// 底层判定工具
// =============================================================================

func (r *NodeContextResolver) isChainExpression(node *sitter.Node) bool {
	k := node.Kind()
	return k == "field_access" || k == "method_invocation" || k == "array_access"
}

func (r *NodeContextResolver) isInvocationExpression(node *sitter.Node) bool {
	k := node.Kind()
	return k == "method_invocation" || k == "explicit_constructor_invocation" ||
		k == "object_creation_expression" || k == "method_reference"
}

func (r *NodeContextResolver) isPartOfInvocation(invokeNode, node *sitter.Node) bool {
	return helper.IsNodeContained(invokeNode.ChildByFieldName("object"), node) ||
		helper.IsNodeContained(invokeNode.ChildByFieldName("arguments"), node) ||
		helper.IsNodeContained(invokeNode.ChildByFieldName("type_arguments"), node)
}

// canChainInKind 统一改为接收 string 类型，消除了多处由于 Node 和 Kind 文本不匹配带来的强转和假节点补丁
func (r *NodeContextResolver) canChainInKind(kind string) bool {
	return kind == "parenthesized_expression" || kind == "cast_expression" ||
		kind == "binary_expression" || kind == "ternary_expression" ||
		kind == "unary_expression" || kind == "update_expression"
}

func (r *NodeContextResolver) buildResult(node *sitter.Node) *Result {
	return &Result{
		ContextNode: node,
		ContextKind: node.Kind(),
		IsChain:     r.isChainExpression(node),
	}
}
