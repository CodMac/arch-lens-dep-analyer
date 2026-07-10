package rel

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

type UseEnricher struct {
	fCtx *core.FileContext
	gCtx *core.GlobalContext
}

func (e *UseEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	node, _ := rel.Mores[constants.TmpNode].(*sitter.Node)
	exprNode, _ := rel.Mores[constants.TmpExpressNode].(*sitter.Node)
	ctxNode, _ := rel.Mores[constants.TmpCtxNode].(*sitter.Node)

	if node == nil || ctxNode == nil {
		return
	}
	if exprNode == nil {
		exprNode = node
	}

	src := *e.fCtx.SourceBytes

	// 1. 基础依赖目标名称
	rel.Mores[constants.RelUseTargetName] = node.Utf8Text(src)

	// 2. 提取并填充 Receiver 文本（只看 field_access 级别的亲属节点，不盲目向上扩大到 method_invocation）
	// TODO: Remember the previous request to convert Receiver strings to TypeSymbols as a to-do item for java_collector.
	if p := exprNode.Parent(); p != nil && p.Kind() == "field_access" {
		if obj := p.ChildByFieldName("object"); obj != nil {
			rel.Mores[constants.RelUseReceiver] = obj.Utf8Text(src)
		}
	} else if node.Parent() != nil && node.Parent().Kind() == "field_access" {
		if obj := node.Parent().ChildByFieldName("object"); obj != nil {
			rel.Mores[constants.RelUseReceiver] = obj.Utf8Text(src)
		}
	} else if rel.Target != nil && rel.Target.Kind == model.Field {
		// 如果解析目标是 Field 且无显式前缀，标记为隐式 this
		rel.Mores[constants.RelUseReceiver] = "this"
	}

	// 3. 填充 ReceiverType
	// 区分 Field（所属类全路径）和 Variable（变量自身的原始类型）
	if rel.Target != nil {
		if rel.Target.Kind == model.Field {
			qn := rel.Target.QualifiedName
			if idx := strings.LastIndex(qn, "."); idx != -1 {
				rel.Mores[constants.RelUseReceiverType] = qn[:idx]
			}
		} else if rel.Target.Kind == model.Variable && rel.Target.Extra != nil {
			keys := []string{constants.VariableRawType, constants.FieldRawType}
			for _, k := range keys {
				if rt, ok := rel.Target.Extra.Mores[k].(string); ok {
					rel.Mores[constants.RelUseReceiverType] = helper.Clean(rt)
					break
				}
			}
		}
	}

	// 4. 处理 EnclosingMethod 和跨作用域捕获 (IsCapture)
	if rel.Source != nil {
		qn := rel.Source.QualifiedName

		// 溯源 EnclosingMethod
		stopMarkers := []string{".lambda", ".anonymousClass", "$", ".block"}
		for _, marker := range stopMarkers {
			if idx := strings.Index(qn, "lambda$"); idx != -1 {
				// 特殊处理 lambda$ 格式
				rel.Mores[constants.RelUseEnclosingMethod] = qn[:idx] + "lambda"
				break
			}
			if idx := strings.Index(qn, marker); idx != -1 {
				rel.Mores[constants.RelUseEnclosingMethod] = qn[:idx]
				break
			}
		}

		// 识别跨作用域捕获 (IsCapture)
		isSubScope := strings.Contains(qn, "lambda$") || strings.Contains(qn, ".anonymousClass")
		if isSubScope && rel.Target != nil {
			if rel.Target.Kind == model.Field {
				rel.Mores[constants.RelUseIsCapture] = true
			}
			if rel.Target.Kind == model.Variable && rel.Source.Location != nil && rel.Target.Location != nil {
				if rel.Source.Location.FilePath == rel.Target.Location.FilePath {
					srcStart := rel.Source.Location.StartLine
					srcEnd := rel.Source.Location.EndLine
					defLine := rel.Target.Location.StartLine
					if defLine < srcStart || defLine > srcEnd {
						rel.Mores[constants.RelUseIsCapture] = true
					}
				}
			}
		}
	}
}
