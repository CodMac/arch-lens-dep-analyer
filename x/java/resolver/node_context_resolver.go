package resolver

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
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

// Result 双轨一元化后的清晰数据容器
type Result struct {
	// Context 级：用于高级类型推断、图拓扑结构分析的宏观最外层表达式/语句边界
	ContextNode *sitter.Node
	ContextKind string

	// Express/Symbol 级：用于符号解析器（Resolver）查找定义的核心连续引用单元（如 a.b.c()）
	ExpressNode *sitter.Node
	ExpressKind string

	// 辅助状态
	IsChain bool
}

// ResolveContext 统一的上下文解析总入口
func (r *NodeContextResolver) ResolveContext(actionType model.DependencyType, node *sitter.Node) *Result {
	if node == nil {
		return nil
	}

	switch actionType {
	case model.Use, model.Assign, model.Call:
		// 🎯 核心重构：将 USE、ASSIGN、CALL 归纳为统一的双轨三步流流水线处理
		return r.resolveTwoTrackPipeline(actionType, node)
	case model.Create:
		return r.resolveForCreate(node)
	case model.Cast:
		return r.resolveForCast(node)
	case model.Throw:
		return r.resolveForThrow(node)
	default:
		return r.buildDefaultResult(node)
	}
}

// =============================================================================
// 一元化流水线核心控制器（The Two-Track Pipeline）
// =============================================================================

func (r *NodeContextResolver) resolveTwoTrackPipeline(actionType model.DependencyType, node *sitter.Node) *Result {
	res := r.buildDefaultResult(node)

	// ---- 第一步：核心连续链条提取（ExpressNode） ----
	if chainNode := r.findCoreChain(node); chainNode != nil {
		res.ExpressNode = chainNode
	}
	res.ExpressKind = res.ExpressNode.Kind()

	// ---- 第二步：特定动作类型的语义微调与锚定（提前，为外溯提供正确的起点） ----
	r.specializeSemanticAction(actionType, node, res)

	// ---- 第三步：宏观语法边界外溯（ContextNode） ----
	if outerBound := r.findOuterBoundary(res.ExpressNode); outerBound != nil {
		res.ContextNode = outerBound
		res.ContextKind = outerBound.Kind()
	} else {
		// 🎯 完美保底：如果外溯没找到更宏观的，且目前的 ContextNode 范围还不如 ExpressNode 大
		// 说明没有被特化层提前锚定（如 Assign 提前锁死了语句），此时无条件对齐 ExpressNode
		if res.ContextNode == nil || res.ContextNode.EndPosition().Row < res.ExpressNode.EndPosition().Row || res.ContextNode.Kind() == "identifier" {
			res.ContextNode = res.ExpressNode
			res.ContextKind = res.ExpressNode.Kind()
		}
	}

	res.IsChain = r.isChainComponent(res.ExpressKind)
	return res
}

// specializeSemanticAction 根据依赖关系动作类别，做无状态、轻量级的语义节点修正
func (r *NodeContextResolver) specializeSemanticAction(actionType model.DependencyType, originNode *sitter.Node, res *Result) {
	switch actionType {
	case model.Assign:
		// 赋值写关系：确保核心链条归属于赋值表达式的左值
		if assignExpr := helper.FindNearestKind(originNode, "assignment_expression"); assignExpr != nil {
			if leftNode := assignExpr.ChildByFieldName("left"); leftNode != nil {
				// 🎯 1. 修正表达链条（ExpressNode）
				if r.isIdentifier(leftNode.Kind()) {
					res.ExpressNode = leftNode
				} else if chainedLeft := r.findChainedAssignLeft(originNode); chainedLeft != nil {
					res.ExpressNode = chainedLeft
				} else {
					res.ExpressNode = leftNode
				}
				res.ExpressKind = res.ExpressNode.Kind()

				// 🎯 2. 核心修复：显式锚定 ContextNode 为 assignment_expression
				// 这样在流水线的第三步外溯时，即使找不到更外层的 macro 结构，也能保底留住赋值语句边界
				res.ContextNode = assignExpr
				res.ContextKind = "assignment_expression"
			}
		}

	case model.Call:
		// 调用关系：如果当前节点是方法的叶子标识符，确保其能顺势覆盖到它所代表的直接 method_invocation 表达单元
		if !r.isInvocationExpression(res.ExpressNode) {
			parent := res.ExpressNode.Parent()
			for parent != nil {
				if r.isInvocationExpression(parent) {
					targetNode := parent
					if r.isPartOfInvocation(parent, originNode) {
						if outer := r.findCoreChain(parent); outer != nil {
							targetNode = outer
						}
					}
					// 🎯 只修正表达链条，不越权干涉外部 Context 边界
					res.ExpressNode = targetNode
					res.ExpressKind = targetNode.Kind()
					break
				}
				parent = parent.Parent()
			}
		}

	case model.Use:
		// 纯读关系：天然契合双轨流的默认提取结果，保持原样即可
	}
}

// =============================================================================
// 高内聚一元化工具爬升层
// =============================================================================

// findCoreChain 提取专门用于符号引用的局部连续连缀链，杜绝被外部算术/条件运算截断
func (r *NodeContextResolver) findCoreChain(node *sitter.Node) *sitter.Node {
	parent := node.Parent()
	var outerChainNode *sitter.Node

	for parent != nil {
		kind := parent.Kind()
		switch kind {
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
				return outerChainNode // 作为方法名时，其本身的 receiver 连缀至此阻断
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
			return outerChainNode // 撞见这些底层流式运算符号，说明符号连缀链条已达物理终点
		default:
			if r.canChainInKind(kind) {
				node = parent
			} else {
				return outerChainNode
			}
		}
		parent = node.Parent()
	}
	return outerChainNode
}

// findOuterBoundary 向上寻找宏观表达式的最外层合法的独立语义边界
func (r *NodeContextResolver) findOuterBoundary(startNode *sitter.Node) *sitter.Node {
	var bestBound *sitter.Node
	curr := startNode
	parent := curr.Parent()

	for parent != nil {
		kind := parent.Kind()

		// 命中具有高级推断或拓扑价值的宏观表达式
		if r.isMacroExpressionKind(kind) {
			bestBound = parent
		}

		// 根据统一的穿透名单阻断或前行
		if !r.canPenetrateUpward(kind) {
			break
		}
		curr = parent
		parent = curr.Parent()
	}
	return bestBound
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
// 其他独立特化解析层
// =============================================================================

func (r *NodeContextResolver) resolveForCreate(node *sitter.Node) *Result {
	if createStmt := helper.FindNearestKind(node, "object_creation_expression", "array_creation_expression"); createStmt != nil {
		return &Result{
			ContextNode: createStmt,
			ContextKind: createStmt.Kind(),
			ExpressNode: createStmt,
			ExpressKind: createStmt.Kind(),
			IsChain:     false,
		}
	}
	return r.buildDefaultResult(node)
}

func (r *NodeContextResolver) resolveForThrow(node *sitter.Node) *Result {
	if throwStmt := helper.FindNearestKind(node, "throw_statement"); throwStmt != nil {
		return &Result{
			ContextNode: throwStmt,
			ContextKind: "throw_statement",
			ExpressNode: node,
			ExpressKind: node.Kind(),
			IsChain:     false,
		}
	}
	return r.buildDefaultResult(node)
}

func (r *NodeContextResolver) resolveForCast(node *sitter.Node) *Result {
	if castExpr := helper.FindNearestKind(node, "cast_expression", "instanceof_expression"); castExpr != nil {
		return &Result{
			ContextNode: castExpr,
			ContextKind: castExpr.Kind(),
			ExpressNode: node,
			ExpressKind: node.Kind(),
			IsChain:     false,
		}
	}
	return r.buildDefaultResult(node)
}

// =============================================================================
// 无状态规则与微工具判定集
// =============================================================================

func (r *NodeContextResolver) isChainComponent(kind string) bool {
	return kind == "field_access" || kind == "method_invocation" || kind == "array_access"
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

func (r *NodeContextResolver) isIdentifier(kind string) bool {
	return kind == "identifier" || kind == "type_identifier"
}

func (r *NodeContextResolver) isMacroExpressionKind(kind string) bool {
	return kind == "binary_expression" || kind == "ternary_expression" ||
		kind == "cast_expression" || kind == "enhanced_for_statement" ||
		kind == "lambda_expression" || kind == "method_invocation" ||
		kind == "array_access"
}

func (r *NodeContextResolver) canPenetrateUpward(kind string) bool {
	return r.canChainInKind(kind) ||
		kind == "argument_list" || kind == "arguments" ||
		kind == "variable_declarator" || kind == "parenthesized_expression" ||
		kind == "binary_expression" || kind == "array_access" ||
		// 🎯 允许穿透语句和块的语法噪音，从而能够顺利高攀到外部包裹着的 lambda_expression
		kind == "expression_statement" || kind == "block"
}

func (r *NodeContextResolver) canChainInKind(kind string) bool {
	return kind == "parenthesized_expression" || kind == "cast_expression" ||
		kind == "binary_expression" || kind == "ternary_expression" ||
		kind == "unary_expression" || kind == "update_expression"
}

func (r *NodeContextResolver) buildDefaultResult(node *sitter.Node) *Result {
	return &Result{
		ContextNode: node,
		ContextKind: node.Kind(),
		ExpressNode: node,
		ExpressKind: node.Kind(),
		IsChain:     r.isChainComponent(node.Kind()),
	}
}
