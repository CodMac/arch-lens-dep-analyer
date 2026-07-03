package rel

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

type CallEnricher struct {
	fCtx *core.FileContext
	gCtx *core.GlobalContext
}

func (e *CallEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	node, _ := GetRelTmpValue(rel)
	//src := *e.fCtx.SourceBytes

	// 基础元数据
	rel.Mores[constants.RelCallIsStatic] = false
	rel.Mores[constants.RelCallIsConstructor] = false
	rel.Mores[constants.RelCallIsChained] = false

	if node == nil {
		return
	}

	// 补全方法名括号，使其符合 collector 规范
	if rel.Target != nil && rel.Target.Kind == model.Method && !strings.HasSuffix(rel.Target.QualifiedName, ")") {
		rel.Target.QualifiedName += "()"
	}

	// 定位调用的真实 AST 容器节点
	callNode := helper.FindNearestKind(node, "method_invocation", "method_reference", "explicit_constructor_invocation", "object_creation_expression")
	if callNode == nil {
		return
	}

	// 分场景填充元数据
	//switch callNode.Kind() {
	//case "method_invocation": // 函数调用
	//	if objectNode := callNode.ChildByFieldName("object"); objectNode != nil {
	//		receiverRaw := objectNode.Utf8Text(src)
	//		rel.Mores[constants.RelCallReceiver] = e._normalizeReceiverText(receiverRaw)
	//
	//		// 判定静态调用，必须排除 getList() 这种带括号的 receiver
	//		isStatic := helper.IsPotentialClassName(receiverRaw)
	//		if isStatic {
	//			rel.Mores[constants.RelCallIsStatic] = isStatic
	//
	//			// 利用 resolver 解析类名
	//			typeEle := e.resolver.ResolveType(e.gCtx, e.fCtx, receiverRaw, model.Class)
	//			if typeEle != nil {
	//				rel.Mores[constants.RelCallReceiverType] = typeEle.QualifiedName
	//			}
	//		}
	//
	//		// 识别链式调用：user.getName().trim(); new User().getName().trim();
	//		if objectNode.Kind() == "method_invocation" || objectNode.Kind() == "object_creation_expression" {
	//			rel.Mores[constants.RelCallIsChained] = true
	//		}
	//
	//		// 实例变量调用：obj1.method()
	//		if objectNode.Kind() == "identifier" {
	//			varEle := e.resolver.ResolveVar(e.gCtx, e.fCtx, objectNode, "", receiverRaw)
	//			if varEle != nil && varEle.Extra != nil {
	//				// 优先使用带 QN 的类型
	//				if typeQN, ok := varEle.Extra.Mores[constants.VariableTypeWithQN].(string); ok {
	//					rel.Mores[constants.RelCallReceiverType] = typeQN
	//				}
	//
	//				// 如果没有 QN，尝试从原始类型解析为 QN
	//				if rel.Mores[constants.RelCallReceiverType] == nil {
	//					if rawType, ok := varEle.Extra.Mores[constants.VariableRawType].(string); ok {
	//						typeEle := e.resolver.ResolveType(e.gCtx, e.fCtx, rawType, model.Class)
	//						if typeEle != nil {
	//							rel.Mores[constants.RelCallReceiverType] = typeEle.QualifiedName
	//						}
	//					}
	//				}
	//			}
	//		}
	//	} else {
	//		rel.Mores[constants.RelCallReceiver] = "this"
	//		rel.Mores[constants.RelCallIsStatic] = false
	//	}
	//
	//case "object_creation_expression": // 构造函数赋值
	//	rel.Mores[constants.RelCallIsConstructor] = true
	//
	//	if typeNode := callNode.ChildByFieldName("type"); typeNode != nil {
	//		typeName := typeNode.Utf8Text(src)
	//		rel.Mores[constants.RelCallReceiver] = e._normalizeReceiverText(typeName)
	//
	//		typeEle := e.resolver.ResolveType(e.gCtx, e.fCtx, typeName, model.Class)
	//		if typeEle != nil {
	//			rel.Mores[constants.RelCallReceiverType] = typeEle.QualifiedName
	//		}
	//	}
	//
	//case "method_reference": // 方法引用
	//	rel.Mores[constants.RelCallIsFunctional] = true
	//
	//	if objectNode := callNode.ChildByFieldName("object"); objectNode != nil {
	//		receiverRaw := objectNode.Utf8Text(src)
	//		rel.Mores[constants.RelCallReceiver] = e._normalizeReceiverText(receiverRaw)
	//
	//		if helper.IsPotentialClassName(receiverRaw) {
	//			rel.Mores[constants.RelCallIsStatic] = true
	//		}
	//	}
	//
	//case "explicit_constructor_invocation": // 显式构造函数调用
	//	rel.Mores[constants.RelCallIsConstructor] = true
	//	if callNode.ChildCount() > 0 {
	//		rel.Mores[constants.RelCallReceiver] = callNode.Child(0).Utf8Text(src)
	//	}
	//}

	// EnclosingMethod 溯源 (Lambda/匿名类溯源到所属方法)
	if rel.Source != nil {
		qn := rel.Source.QualifiedName
		stopMarkers := []string{".lambda", ".anonymousClass", "$", ".block"}
		for _, marker := range stopMarkers {
			if idx := strings.Index(qn, marker); idx != -1 {
				rel.Mores[constants.RelCallEnclosingMethod] = qn[:idx]
				break
			}
		}
	}
}

// _normalizeReceiverText 标准化receiver文本（去除换行符和多余空格）
func (e *CallEnricher) _normalizeReceiverText(raw string) string {
	normalized := ""
	for _, step := range strings.Split(raw, ".") {
		normalized += "." + strings.TrimSpace(step)
	}

	return strings.TrimLeft(normalized, ".")
}
