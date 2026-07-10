package rel

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type AssignEnricher struct {
	fCtx *core.FileContext
	gCtx *core.GlobalContext
}

// ============================================================================
// === 1. 核心生命周期与主入口 ===
// ============================================================================

// EnrichMetadata 负责对 Java 赋值依赖关系的元数据进行多维度、深层次的补全增强
func (e *AssignEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	node, _ := rel.Mores[constants.TmpNode].(*sitter.Node)
	ctxNode, _ := rel.Mores[constants.TmpCtxNode].(*sitter.Node)
	if node == nil || ctxNode == nil {
		return
	}

	src := *e.fCtx.SourceBytes

	// 1. 基础信息补全
	rel.Mores[constants.RelAssignTargetName] = node.Utf8Text(src)

	// 2. 根据不同的上下文语法节点(Context Kind)路由至对应分支提取不同的元信息
	e.extractByContextKind(rel, node, ctxNode, src)

	// 3. 补全 Receiver (调用主体) 信息
	e.extractReceiverInfo(rel, node, src)

	// 4. 计算 EnclosingMethod 闭包边界以及 IsCapture 状态
	e.processEnclosingMethod(rel)
}

// extractByContextKind 识别当前上下文类型并对 AST 树节点做必要的上溯对齐
func (e *AssignEnricher) extractByContextKind(rel *model.DependencyRelation, node, ctx *sitter.Node, src []byte) {
	ctxKind := ctx.Kind()

	// 提前防御：如果目标标识符被包裹在 field_access 里，而外层是具体的赋值/自增/声明节点，
	// 应当优先将上下文指针上溯至最高优先级的表达式节点进行统一处理。
	if ctxKind == "identifier" || ctxKind == "field_access" {
		p := ctx.Parent()
		if p != nil && (p.Kind() == "assignment_expression" || p.Kind() == "update_expression" || p.Kind() == "variable_declarator") {
			ctx = p
			ctxKind = p.Kind()
		}
	}

	// 路由分发
	switch ctxKind {
	case "variable_declarator":
		e.extractFromVariableDeclarator(rel, node, ctx, src)
	case "assignment_expression":
		e.extractFromAssignmentExpression(rel, node, ctx, src)
	case "update_expression":
		e.extractFromUpdateExpression(rel, node, ctx, src)
	default:
		// 当上下文不在预期节点内时，从当前节点往上探测追溯
		e.findAndExtractFromAncestorAssignment(rel, node, ctx, src)
	}
}

// ============================================================================
// === 2. 细粒度语法树分支提取器 ===
// ============================================================================

// extractFromVariableDeclarator 提取变量声明初始化场景: int local = 10;
func (e *AssignEnricher) extractFromVariableDeclarator(rel *model.DependencyRelation, node, ctx *sitter.Node, src []byte) {
	leftExpr := node.Utf8Text(src)
	if nameNode := ctx.ChildByFieldName("name"); nameNode != nil {
		leftExpr = nameNode.Utf8Text(src)
	}

	rightExpr := ""
	if valueNode := ctx.ChildByFieldName("value"); valueNode != nil {
		rightExpr = valueNode.Utf8Text(src)
	}

	e.fillAssignMores(rel, fillParams{
		IsInitializer: true,
		Operator:      "=",
		LeftExpr:      leftExpr,
		RightExpr:     rightExpr,
		RawText:       node.Utf8Text(src),
	})
}

// extractFromAssignmentExpression 提取标准赋值表达式场景: a = b = 50; 或者 this.count = 100;
func (e *AssignEnricher) extractFromAssignmentExpression(rel *model.DependencyRelation, node, ctx *sitter.Node, src []byte) {
	// 强力向上寻找真正的 assignment_expression 顶层节点
	for ctx != nil && ctx.Kind() != "assignment_expression" {
		ctx = ctx.Parent()
	}
	if ctx == nil {
		e.extractGenericAssign(rel, node, ctx, src)
		return
	}

	leftExpr := node.Utf8Text(src)
	if leftNode := ctx.ChildByFieldName("left"); leftNode != nil {
		leftExpr = leftNode.Utf8Text(src)
	}

	operator := "="
	if opNode := ctx.ChildByFieldName("operator"); opNode != nil {
		operator = opNode.Utf8Text(src)
	}

	rightExpr := ""
	if rightNode := ctx.ChildByFieldName("right"); rightNode != nil {
		rightExpr = rightNode.Utf8Text(src)
	}

	e.fillAssignMores(rel, fillParams{
		IsInitializer: false,
		Operator:      operator,
		LeftExpr:      leftExpr,
		RightExpr:     rightExpr,
		RawText:       ctx.Utf8Text(src),
	})
}

// extractFromUpdateExpression 提取一元自增自减表达式场景: ++count; 或者 count--;
func (e *AssignEnricher) extractFromUpdateExpression(rel *model.DependencyRelation, node, ctx *sitter.Node, src []byte) {
	for ctx != nil && ctx.Kind() != "update_expression" {
		ctx = ctx.Parent()
	}
	if ctx == nil {
		return
	}

	leftExpr := node.Utf8Text(src)
	if leftNode := ctx.ChildByFieldName("argument"); leftNode != nil {
		leftExpr = leftNode.Utf8Text(src)
	}

	rawText := ctx.Utf8Text(src)
	operator := ""
	if strings.Contains(rawText, "++") {
		operator = "++"
	} else if strings.Contains(rawText, "--") {
		operator = "--"
	}

	e.fillAssignMores(rel, fillParams{
		IsInitializer: false,
		Operator:      operator,
		LeftExpr:      leftExpr,
		RightExpr:     "",
		RawText:       rawText,
	})
}

// findAndExtractFromAncestorAssignment 动态祖先节点回溯提取器
func (e *AssignEnricher) findAndExtractFromAncestorAssignment(rel *model.DependencyRelation, node, startNode *sitter.Node, src []byte) {
	current := startNode
	maxDepth := 10

	for current != nil && maxDepth > 0 {
		kind := current.Kind()
		switch kind {
		case "assignment_expression":
			e.extractFromAssignmentExpression(rel, node, current, src)
			return
		case "variable_declarator":
			e.extractFromVariableDeclarator(rel, node, current, src)
			return
		case "update_expression":
			e.extractFromUpdateExpression(rel, node, current, src)
			return
		case "method_declaration", "constructor_declaration", "class_declaration", "program":
			// 触碰到作用域边界，提前终止防止跨阻隔检索
			break
		}
		current = current.Parent()
		maxDepth--
	}

	// 降级兜底处理
	e.fillAssignMores(rel, fillParams{
		IsInitializer: false,
		Operator:      "=",
		LeftExpr:      node.Utf8Text(src),
		RightExpr:     "",
		RawText:       rel.Mores[constants.RelRawText].(string),
	})
}

// extractGenericAssign 极端边界兜底赋值函数
func (e *AssignEnricher) extractGenericAssign(rel *model.DependencyRelation, node, ctx *sitter.Node, src []byte) {
	if node != nil {
		rel.Mores[constants.RelAssignLeftExpression] = node.Utf8Text(src)
	}
	rel.Mores[constants.RelAssignIsInitializer] = false
}

// ============================================================================
// === 3. 通用上下文元数据提取与环境推导 ===
// ============================================================================

// extractReceiverInfo 提取当前赋值变量所绑定的接收者(Receiver)主体
func (e *AssignEnricher) extractReceiverInfo(rel *model.DependencyRelation, node *sitter.Node, src []byte) {
	parent := node.Parent()
	if parent != nil && parent.Kind() == "field_access" {
		if obj := parent.ChildByFieldName("object"); obj != nil {
			rel.Mores[constants.RelAssignReceiver] = obj.Utf8Text(src)
		}
	} else if rel.Target != nil && rel.Target.Kind == model.Field {
		rel.Mores[constants.RelAssignReceiver] = "this"
	}
}

// processEnclosingMethod 逆向截断推导所属方法区，并甄别是否属于闭包内变量捕获(Capture)
func (e *AssignEnricher) processEnclosingMethod(rel *model.DependencyRelation) {
	if rel.Source == nil {
		return
	}

	qn := rel.Source.QualifiedName
	stopMarkers := []string{".lambda", ".anonymousClass", "$", ".block"}
	for _, marker := range stopMarkers {
		if idx := strings.Index(qn, marker); idx != -1 {
			rel.Mores[constants.RelAssignEnclosingMethod] = qn[:idx]
			break
		}
	}

	isSubScope := strings.Contains(qn, "lambda$") || strings.Contains(qn, ".anonymousClass")
	isTargetField := rel.Target != nil && rel.Target.Kind == model.Field

	if isSubScope && isTargetField {
		rel.Mores[constants.RelAssignIsCapture] = true
	}
}

// ============================================================================
// === 4. 辅助重构工具方法 ===
// ============================================================================

type fillParams struct {
	IsInitializer bool
	Operator      string
	LeftExpr      string
	RightExpr     string
	RawText       string
}

// fillAssignMores 抽象提取出来的公共结构体映射赋值方法
func (e *AssignEnricher) fillAssignMores(rel *model.DependencyRelation, params fillParams) {
	rel.Mores[constants.RelAssignIsInitializer] = params.IsInitializer
	rel.Mores[constants.RelAssignOperator] = params.Operator
	rel.Mores[constants.RelAssignLeftExpression] = params.LeftExpr
	rel.Mores[constants.RelAssignRightExpression] = params.RightExpr
	rel.Mores[constants.RelRawText] = params.RawText
}
