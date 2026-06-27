package rel

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"strings"
)

type AssignEnricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
	gCtx     *core.GlobalContext
}

func (e *AssignEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	node, ctx := GetRelTmpValue(rel)
	src := *e.fCtx.SourceBytes

	// --- 基础信息补全 ---
	rel.Mores[constants.RelRawText] = ctx.Utf8Text(src)
	rel.Mores[constants.RelAstKind] = node.Kind() // 记录为 identifier
	rel.Mores[constants.RelAssignTargetName] = node.Utf8Text(src)

	// --- 提取并填充 Receiver 属性 ---
	parent := node.Parent()
	if parent != nil && parent.Kind() == "field_access" {
		if obj := parent.ChildByFieldName("object"); obj != nil {
			rel.Mores[constants.RelAssignReceiver] = obj.Utf8Text(src)
		}
	} else if rel.Target != nil && rel.Target.Kind == model.Field {
		// 如果解析出来的目标是 Field，且没有显式前缀，则标记为隐式 this
		rel.Mores[constants.RelAssignReceiver] = "this"
	}

	// --- 提取 Operator 和 Value ---
	switch ctx.Kind() {
	case "variable_declarator":
		rel.Mores[constants.RelAssignIsInitializer] = true
		rel.Mores[constants.RelAssignOperator] = "="
		if val := ctx.ChildByFieldName("value"); val != nil {
			rel.Mores[constants.RelAssignValueExpression] = val.Utf8Text(src)
		}
	case "assignment_expression":
		rel.Mores[constants.RelAssignIsInitializer] = false
		if op := ctx.ChildByFieldName("operator"); op != nil {
			rel.Mores[constants.RelAssignOperator] = op.Utf8Text(src)
		}
		if right := ctx.ChildByFieldName("right"); right != nil {
			rel.Mores[constants.RelAssignValueExpression] = right.Utf8Text(src)
		}
	case "update_expression":
		rel.Mores[constants.RelAssignIsInitializer] = false
		// 处理 ++ / --
		txt := ctx.Utf8Text(src)
		if strings.Contains(txt, "++") {
			rel.Mores[constants.RelAssignOperator] = "++"
		} else {
			rel.Mores[constants.RelAssignOperator] = "--"
		}
	}

	// --- 处理 EnclosingMethod 和 IsCapture ---
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
