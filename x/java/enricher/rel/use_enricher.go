package rel

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

type UseEnricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
	gCtx     *core.GlobalContext
}

func (e *UseEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	node, ctx := GetRelTmpValue(rel)
	src := *e.fCtx.SourceBytes

	if node == nil || ctx == nil {
		return
	}

	rel.Mores[constants.RelUseTargetName] = node.Utf8Text(src)
	rel.Mores[constants.RelRawText] = ctx.Utf8Text(src)
	rel.Mores[constants.RelAstKind] = node.Kind()

	// 1. 设置 Context 类型 (例如 field_access 或 assignment_expression)
	rel.Mores[constants.RelContextAstKind] = ctx.Kind()

	// 2. 提取并填充 Receiver 文本
	parent := node.Parent()
	if parent != nil && parent.Kind() == "field_access" {
		if obj := parent.ChildByFieldName("object"); obj != nil {
			rel.Mores[constants.RelUseReceiver] = obj.Utf8Text(src)
		}
	} else if rel.Target != nil && rel.Target.Kind == model.Field {
		// 如果解析目标是 Field 且无显式前缀，标记为隐式 this
		rel.Mores[constants.RelUseReceiver] = "this"
	}

	// 3. 填充 ReceiverType
	// 逻辑：如果 Target 是一个 Field，其 ReceiverType 通常是该 Field 所属类的 QualifiedName
	if rel.Target != nil && rel.Target.Kind == model.Field {
		qn := rel.Target.QualifiedName
		if idx := strings.LastIndex(qn, "."); idx != -1 {
			// 截取掉最后的字段名，保留类全路径
			rel.Mores[constants.RelUseReceiverType] = qn[:idx]
		}
	}

	// --- 提取接收者类型 QN ---
	// 这里利用你之前从 Target.Extra 中收集到的 RawType
	if rel.Target != nil && rel.Target.Extra != nil {
		keys := []string{constants.FieldRawType, constants.VariableRawType}
		for _, k := range keys {
			if rt, ok := rel.Target.Extra.Mores[k].(string); ok {
				// e.clean 会去掉泛型和修饰符，保留纯粹的类型名
				rel.Mores[constants.RelUseReceiverType] = helper.Clean(rt)
				break
			}
		}
	}

	// 处理 EnclosingMethod 和 IsCapture
	if rel.Source != nil {
		qn := rel.Source.QualifiedName

		// 1. 溯源 EnclosingMethod
		stopMarkers := []string{".lambda", ".anonymousClass", "$", ".block"}
		for _, marker := range stopMarkers {
			if idx := strings.Index(qn, marker); idx != -1 {
				rel.Mores[constants.RelUseEnclosingMethod] = qn[:idx]
				break
			}
		}

		// 2. 识别跨作用域捕获 (IsCapture)
		isSubScope := strings.Contains(qn, "lambda$") || strings.Contains(qn, ".anonymousClass")
		if isSubScope {
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
