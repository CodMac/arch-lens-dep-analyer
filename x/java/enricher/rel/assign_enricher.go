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
	node, ctx := GetRelTmpValue(rel)
	src := *e.fCtx.SourceBytes

	// --- 基础信息补全 ---
	rel.Mores[constants.RelAssignTargetName] = node.Utf8Text(src)

	// --- 根据context节点类型提取不同的元信息 ---
	e.extractByContextKind(rel, node, ctx, src)

	// --- 提取 Receiver 信息 ---
	e.extractReceiverInfo(rel, node, src)

	// --- 处理 EnclosingMethod 和 IsCapture ---
	e.processEnclosingMethod(rel)
}

func (e *AssignEnricher) extractByContextKind(rel *model.DependencyRelation, node, ctx *sitter.Node, src []byte) {
	ctxKind := ctx.Kind()

	switch ctxKind {
	case "variable_declarator":
		e.extractFromVariableDeclarator(rel, node, ctx, src)
	case "assignment_expression":
		e.extractFromAssignmentExpression(rel, node, ctx, src)
	case "update_expression":
		e.extractFromUpdateExpression(rel, node, ctx, src)
	case "identifier":
		e.extractFromIdentifierContext(rel, node, ctx, src)
	default:
		e.extractGenericAssign(rel, node, ctx, src)
	}
}

func (e *AssignEnricher) extractFromVariableDeclarator(rel *model.DependencyRelation, node, ctx *sitter.Node, src []byte) {
	rel.Mores[constants.RelAssignIsInitializer] = true
	rel.Mores[constants.RelAssignOperator] = "="

	leftExpr := node.Utf8Text(src)
	rel.Mores[constants.RelAssignLeftExpression] = leftExpr

	if nameNode := ctx.ChildByFieldName("name"); nameNode != nil {
		rel.Mores[constants.RelAssignLeftExpression] = nameNode.Utf8Text(src)
	}

	value := ctx.ChildByFieldName("value")
	if value != nil {
		rightExpr := value.Utf8Text(src)
		rel.Mores[constants.RelAssignRightExpression] = rightExpr
	} else {
		rel.Mores[constants.RelAssignRightExpression] = ""
	}
}

func (e *AssignEnricher) extractFromAssignmentExpression(rel *model.DependencyRelation, node, ctx *sitter.Node, src []byte) {
	// 如果ctx不是assignment_expression，说明节点选择器可能有问题
	// 尝试从node向上找到真正的assignment_expression
	if ctx.Kind() != "assignment_expression" {
		parent := ctx
		for parent != nil {
			if parent.Kind() == "assignment_expression" {
				ctx = parent
				break
			}
			parent = parent.Parent()
		}
	}

	rel.Mores[constants.RelAssignIsInitializer] = false

	leftNode := ctx.ChildByFieldName("left")
	if leftNode != nil {
		rel.Mores[constants.RelAssignLeftExpression] = leftNode.Utf8Text(src)
	} else {
		rel.Mores[constants.RelAssignLeftExpression] = node.Utf8Text(src)
	}

	if op := ctx.ChildByFieldName("operator"); op != nil {
		rel.Mores[constants.RelAssignOperator] = op.Utf8Text(src)
	} else {
		// 默认使用"="操作符
		rel.Mores[constants.RelAssignOperator] = "="
	}

	rightNode := ctx.ChildByFieldName("right")
	if rightNode != nil {
		rightExpr := rightNode.Utf8Text(src)
		rel.Mores[constants.RelAssignRightExpression] = rightExpr
	} else {
		// 如果找不到right，可能是因为ctx不是真正的assignment_expression
		// 尝试再次向上查找
		parent := ctx.Parent()
		for parent != nil && rightNode == nil {
			if parent.Kind() == "assignment_expression" {
				right := parent.ChildByFieldName("right")
				if right != nil {
					rel.Mores[constants.RelAssignRightExpression] = right.Utf8Text(src)
					break
				}
			}
			parent = parent.Parent()
		}
	}
}

func (e *AssignEnricher) extractFromUpdateExpression(rel *model.DependencyRelation, node, ctx *sitter.Node, src []byte) {
	rel.Mores[constants.RelAssignIsInitializer] = false

	leftExpr := node.Utf8Text(src)
	rel.Mores[constants.RelAssignLeftExpression] = leftExpr

	rawText := ctx.Utf8Text(src)
	if strings.HasSuffix(rawText, "++") {
		rel.Mores[constants.RelAssignOperator] = "++"
	} else if strings.HasSuffix(rawText, "--") {
		rel.Mores[constants.RelAssignOperator] = "--"
	} else if strings.HasPrefix(rawText, "++") {
		rel.Mores[constants.RelAssignOperator] = "++"
	} else if strings.HasPrefix(rawText, "--") {
		rel.Mores[constants.RelAssignOperator] = "--"
	}

	rel.Mores[constants.RelAssignRightExpression] = ""
}

func (e *AssignEnricher) extractFromIdentifierContext(rel *model.DependencyRelation, node, ctx *sitter.Node, src []byte) {
	parent := node.Parent()
	if parent == nil {
		return
	}

	parentKind := parent.Kind()

	switch parentKind {
	case "assignment_expression":
		e.extractFromAssignmentExpression(rel, node, parent, src)
	case "variable_declarator":
		e.extractFromVariableDeclarator(rel, node, parent, src)
	case "update_expression":
		e.extractFromUpdateExpression(rel, node, parent, src)
	default:
		e.findAndExtractFromAncestorAssignment(rel, node, parent, src)
	}
}

func (e *AssignEnricher) findAndExtractFromAncestorAssignment(rel *model.DependencyRelation, node, startNode *sitter.Node, src []byte) {
	current := startNode
	maxDepth := 10 // 防止无限循环

	// 向上查找assignment_expression节点
	for current != nil && maxDepth > 0 {
		kind := current.Kind()

		if kind == "assignment_expression" {
			// 找到赋值表达式节点，提取元信息
			e.extractFromAssignmentExpression(rel, node, current, src)
			return
		}

		if kind == "variable_declarator" {
			e.extractFromVariableDeclarator(rel, node, current, src)
			return
		}

		if kind == "update_expression" {
			e.extractFromUpdateExpression(rel, node, current, src)
			return
		}

		// 避免无限循环，检查一些不需要继续查找的节点类型
		if kind == "method_declaration" || kind == "constructor_declaration" || kind == "class_declaration" || kind == "program" {
			break
		}

		current = current.Parent()
		maxDepth--
	}

	// 如果没有找到赋值表达式，使用默认处理
	rel.Mores[constants.RelAssignLeftExpression] = node.Utf8Text(src)
	rel.Mores[constants.RelAssignIsInitializer] = false
	rel.Mores[constants.RelAssignOperator] = "="
	rel.Mores[constants.RelAssignRightExpression] = "" // 设置空右值
}

func (e *AssignEnricher) extractGenericAssign(rel *model.DependencyRelation, node, ctx *sitter.Node, src []byte) {
	rel.Mores[constants.RelAssignLeftExpression] = node.Utf8Text(src)
	rel.Mores[constants.RelAssignIsInitializer] = false
}

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

func (e *AssignEnricher) processEnclosingMethod(rel *model.DependencyRelation) {
	if rel.Source != nil {
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
}
