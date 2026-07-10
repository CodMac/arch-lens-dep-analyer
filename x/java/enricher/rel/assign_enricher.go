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

func (e *AssignEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	node, _ := rel.Mores[constants.TmpNode].(*sitter.Node)
	exprNode, _ := rel.Mores[constants.TmpExpressNode].(*sitter.Node)
	ctxNode, _ := rel.Mores[constants.TmpCtxNode].(*sitter.Node)

	if node == nil || exprNode == nil || ctxNode == nil {
		return
	}

	src := *e.fCtx.SourceBytes

	// 🚀 【核心修复 1】容器向上回溯机制 (Anchor Recovery)
	// 如果 ctxNode 退化为了普通标识符或表达式，说明上游传入的边界不够大，逆向追溯到真正的语法语义容器
	container := ctxNode
	for container != nil {
		k := container.Kind()
		if k == "variable_declarator" || k == "assignment_expression" || k == "update_expression" {
			break
		}
		container = container.Parent()
	}
	// 如果追溯到了有效容器，用容器纠正 ctxNode；否则维持原样
	if container != nil {
		ctxNode = container
	}

	// 1. 基础左值与 Receiver 提取
	rel.Mores[constants.RelAssignTargetName] = node.Utf8Text(src)
	rel.Mores[constants.RelAssignLeftExpression] = exprNode.Utf8Text(src)

	if p := node.Parent(); p != nil && p.Kind() == "field_access" {
		if obj := p.ChildByFieldName("object"); obj != nil {
			rel.Mores[constants.RelAssignReceiver] = obj.Utf8Text(src)
		}
	} else if rel.Target != nil && rel.Target.Kind == model.Field {
		rel.Mores[constants.RelAssignReceiver] = "this"
	}

	// 2. 闭包边界截断与变量捕获判断
	e.processEnclosingMethod(rel)

	// 3. 实时刷新纠正后的 RawText 和默认 Operator
	rel.Mores[constants.RelRawText] = ctxNode.Utf8Text(src)
	rel.Mores[constants.RelAssignIsInitializer] = false
	rel.Mores[constants.RelAssignOperator] = "="

	// 4. 🚀 【核心修复 2】基于纠正后的稳定容器提取属性
	switch ctxNode.Kind() {
	case "variable_declarator":
		rel.Mores[constants.RelAssignIsInitializer] = true
		if nameNode := ctxNode.ChildByFieldName("name"); nameNode != nil {
			rel.Mores[constants.RelAssignLeftExpression] = nameNode.Utf8Text(src)
		}
		if valNode := ctxNode.ChildByFieldName("value"); valNode != nil {
			rel.Mores[constants.RelAssignRightExpression] = valNode.Utf8Text(src)
		}

	case "assignment_expression":
		if opNode := ctxNode.ChildByFieldName("operator"); opNode != nil {
			rel.Mores[constants.RelAssignOperator] = opNode.Utf8Text(src)
		}
		if rightNode := ctxNode.ChildByFieldName("right"); rightNode != nil {
			rel.Mores[constants.RelAssignRightExpression] = rightNode.Utf8Text(src)
		}

	case "update_expression":
		if argNode := ctxNode.ChildByFieldName("argument"); argNode != nil {
			rel.Mores[constants.RelAssignLeftExpression] = argNode.Utf8Text(src)
		}
		// 根据节点完整文本精准切分操作符
		ctxText := ctxNode.Utf8Text(src)
		if strings.Contains(ctxText, "++") {
			rel.Mores[constants.RelAssignOperator] = "++"
		} else if strings.Contains(ctxText, "--") {
			rel.Mores[constants.RelAssignOperator] = "--"
		}
	}
}

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
