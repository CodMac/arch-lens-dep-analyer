package java

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type RelMetadataEnricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
	gCtx     *core.GlobalContext
}

const TmpNode = "tmp_node"
const TmpRaw = "tmp_raw"
const TmpStmt = "tmp_stmt"

func (e *RelMetadataEnricher) EnrichCoreMetadata(rel *model.DependencyRelation) {
	node, rawText, stmt := e._getRelTmpValue(rel)
	src := *e.fCtx.SourceBytes

	switch rel.Type {
	case model.Call:
		e.enrichCallCore(rel, node, stmt, src)
	case model.Create:
		e.enrichCreateCore(rel, node, stmt, src)
	case model.Assign:
		e.enrichAssignCore(rel, node, stmt, src)
	case model.Use:
		e.enrichUseCore(rel, node, stmt, src)
	case model.Cast:
		e.enrichCastCore(rel, node, stmt, src)
	case model.Throw:
		e.enrichThrowCore(rel, node, stmt, rawText, src)
	case model.Parameter:
		e.enrichParameterCore(rel, rawText)
	case model.Return:
		e.enrichReturnCore(rel, rawText)
	case model.Annotation:
		e.enrichAnnotationCore(rel)
	}
}

func (e *RelMetadataEnricher) enrichCallCore(rel *model.DependencyRelation, node *sitter.Node, ctx *sitter.Node, src []byte) {
	rel.Mores[constants.RelCallIsStatic] = false
	rel.Mores[constants.RelCallIsConstructor] = false
	rel.Mores[constants.RelCallIsChained] = false
	rel.Mores[constants.RelAstKind] = node.Kind()
	rel.Mores[constants.RelRawText] = ctx.Utf8Text(src)
	rel.Mores[constants.RelContext] = ctx.Kind()

	if node == nil {
		return
	}

	// 补全方法名括号，使其符合 collector 规范
	if rel.Target != nil && rel.Target.Kind == model.Method && !strings.HasSuffix(rel.Target.QualifiedName, ")") {
		rel.Target.QualifiedName += "()"
	}

	// 定位调用的真实 AST 容器节点
	callNode := FindNearestKind(node, "method_invocation", "method_reference", "explicit_constructor_invocation", "object_creation_expression")
	if callNode == nil {
		return
	}

	switch callNode.Kind() {
	case "method_invocation":
		if objectNode := callNode.ChildByFieldName("object"); objectNode != nil {
			receiverRaw := objectNode.Utf8Text(src)
			rel.Mores[constants.RelCallReceiverRaw] = receiverRaw
			rel.Mores[constants.RelCallReceiver] = e._normalizeReceiverText(receiverRaw)

			// 【核心修复】判定静态调用，必须排除 getList() 这种带括号的 receiver
			isStatic := IsPotentialClassName(receiverRaw)
			rel.Mores[constants.RelCallIsStatic] = isStatic
			if isStatic {
				// 【新增】统一转换为QN格式：利用 Resolver 解析类名
				typeEle := e.resolver.Resolve(e.gCtx, e.fCtx, nil, "", receiverRaw, model.Class)
				if typeEle != nil {
					rel.Mores[constants.RelCallReceiverType] = typeEle.QualifiedName
				}
			}

			// 识别链式调用并推断 Receiver 类型（链式调用的类型推导交由 ChainedCallResolver 处理）
			if objectNode.Kind() == "method_invocation" {
				rel.Mores[constants.RelCallIsChained] = true
				rel.Mores[constants.RelCallChainDepth] = e._calculateChainDepth(receiverRaw)
				// 推断前一个方法调用的返回类型
				receiverTypeQN := e._inferChainedCallReceiverType(objectNode, src)
				if receiverTypeQN != "" {
					rel.Mores[constants.RelCallReceiverType] = receiverTypeQN
				}
			} else if objectNode.Kind() == "object_creation_expression" {
				// 对象创建表达式也是链式调用的一部分（如 new Builder().name()）
				rel.Mores[constants.RelCallIsChained] = true
				rel.Mores[constants.RelCallChainDepth] = e._calculateChainDepth(receiverRaw)
				// 推断对象创建的类型
				receiverTypeQN := e._inferObjectCreationReceiverType(objectNode, src)
				rel.Mores[constants.RelCallReceiverType] = receiverTypeQN
			} else if objectNode.Kind() == "identifier" {
				// 【新增】实例变量调用：obj1.method()
				// 利用 Resolver 解析变量，获取其类型QN
				varEle := e.resolver.Resolve(e.gCtx, e.fCtx, objectNode, "", receiverRaw, model.Variable)
				if varEle != nil && varEle.Extra != nil {
					// 优先使用带 QN 的类型
					if typeQN, ok := varEle.Extra.Mores[constants.VariableTypeWithQN].(string); ok {
						rel.Mores[constants.RelCallReceiverType] = typeQN
					}
					// 如果没有 QN，尝试从原始类型解析为 QN
					if rel.Mores[constants.RelCallReceiverType] == nil {
						if rawType, ok := varEle.Extra.Mores[constants.VariableRawType].(string); ok {
							typeEle := e.resolver.Resolve(e.gCtx, e.fCtx, nil, "", rawType, model.Class)
							if typeEle != nil {
								rel.Mores[constants.RelCallReceiverType] = typeEle.QualifiedName
							}
						}
					}
				}
			}
		} else {
			rel.Mores[constants.RelCallReceiver] = "this"
			rel.Mores[constants.RelCallIsStatic] = false
		}

	case "object_creation_expression":
		rel.Mores[constants.RelCallIsConstructor] = true

		if typeNode := callNode.ChildByFieldName("type"); typeNode != nil {
			typeName := typeNode.Utf8Text(src)
			rel.Mores[constants.RelCallReceiverRaw] = typeName
			rel.Mores[constants.RelCallReceiver] = typeName

			typeEle := e.resolver.Resolve(e.gCtx, e.fCtx, nil, "", typeName, model.Class)
			if typeEle != nil {
				rel.Mores[constants.RelCallReceiverType] = typeEle.QualifiedName
			}
		}

	case "method_reference":
		rel.Mores[constants.RelCallIsFunctional] = true

		if objectNode := callNode.ChildByFieldName("object"); objectNode != nil {
			receiverRaw := objectNode.Utf8Text(src)
			rel.Mores[constants.RelCallReceiverRaw] = receiverRaw
			rel.Mores[constants.RelCallReceiver] = e._normalizeReceiverText(receiverRaw)
			if IsPotentialClassName(receiverRaw) {
				rel.Mores[constants.RelCallIsStatic] = true
			}
		}

	case "explicit_constructor_invocation":
		rel.Mores[constants.RelCallIsConstructor] = true
		if callNode.ChildCount() > 0 {
			rel.Mores[constants.RelCallReceiver] = callNode.Child(0).Utf8Text(src)
		}
	}

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

	// 【新增】补全 CALL 关系 Target 的 QN（移到最后执行，确保 RelCallReceiverType 已被填充）
	// 当 Resolver 无法找到目标方法时，根据 RelCallReceiverType 和方法名构建完整 QN
	if rel.Target != nil && rel.Target.Kind == model.Method {
		targetQN := rel.Target.QualifiedName
		receiverTypeQN, hasReceiverType := rel.Mores[constants.RelCallReceiverType].(string)

		// 检查是否为不完整的 QN（没有点号或者只有简单的点括号结构）
		isIncompleteQN := !strings.Contains(targetQN, ".") ||
			(strings.Count(targetQN, ".") == 0 && strings.Contains(targetQN, "()"))

		if isIncompleteQN && hasReceiverType && receiverTypeQN != "" {
			// 构建完整的 QN: ReceiverType.methodName()
			methodName := rel.Target.Name
			completeQN := receiverTypeQN + "." + methodName

			// 尝试用完整的 QN 去 gCtx 查找，如果找到则更新 Target
			if entry, ok := e.gCtx.FindByQualifiedName(completeQN); ok && entry.Element.Kind == model.Method {
				rel.Target = entry.Element
			} else {
				// 如果找不到，也更新为完整的 QN（可能参数信息不匹配，但至少类名是正确的）
				rel.Target.QualifiedName = completeQN
				rel.Target.IsFormExternal = false // 不再标记为外部符号
			}
		}
	}
}

func (e *RelMetadataEnricher) enrichCreateCore(rel *model.DependencyRelation, node, ctx *sitter.Node, src []byte) {
	if ctx == nil {
		return
	}

	// 1. 通用属性
	rel.Mores[constants.RelAstKind] = ctx.Kind()
	rel.Mores[constants.RelRawText] = ctx.Utf8Text(src)

	// 2. 专用属性提取：变量名 (RelCreateVariableName)
	contextNode := ctx
	if ctx.Kind() == "object_creation_expression" || ctx.Kind() == "array_creation_expression" {
		if p := ctx.Parent(); p != nil && p.Kind() == "variable_declarator" {
			contextNode = p
		}
	}
	if contextNode.Kind() == "variable_declarator" {
		if nameNode := contextNode.ChildByFieldName("name"); nameNode != nil {
			rel.Mores[constants.RelCreateVariableName] = nameNode.Utf8Text(src)
		}
	}

	// 3. 专用属性提取：数组 (RelCreateIsArray)
	if ctx.Kind() == "array_creation_expression" {
		rel.Mores[constants.RelCreateIsArray] = true
	}

	// 4. 特殊处理 super() -> Object 的情况
	if ctx.Kind() == "explicit_constructor_invocation" && strings.Contains(ctx.Utf8Text(src), "super") {
		rel.Target.Name = "Object"
		rel.Target.QualifiedName = "Object"
	}
}

func (e *RelMetadataEnricher) enrichAssignCore(rel *model.DependencyRelation, node, ctx *sitter.Node, src []byte) {
	// --- 基础信息补全 ---
	rel.Mores[constants.RelAssignTargetName] = node.Utf8Text(src)
	rel.Mores[constants.RelRawText] = ctx.Utf8Text(src)
	rel.Mores[constants.RelAstKind] = node.Kind() // 记录为 identifier

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

func (e *RelMetadataEnricher) enrichCastCore(rel *model.DependencyRelation, node, ctx *sitter.Node, src []byte) {
	if ctx == nil {
		return
	}
	rel.Mores[constants.RelAstKind] = ctx.Kind()
	rel.Mores[constants.RelRawText] = ctx.Utf8Text(src)
	rel.Mores[constants.RelCastIsInstanceof] = ctx.Kind() == "instanceof_expression"
}

func (e *RelMetadataEnricher) enrichThrowCore(rel *model.DependencyRelation, node, ctx *sitter.Node, rawText string, src []byte) {
	if node != nil {
		rel.Mores[constants.RelAstKind] = "throw_statement"
		rel.Target.Name = Clean(rel.Target.Name)
		rel.Target.QualifiedName = Clean(rel.Target.QualifiedName)
		if node.Kind() == "type_identifier" || (node.Parent() != nil && node.Parent().Kind() == "object_creation_expression") {
			rel.Mores[constants.RelThrowIsRuntime] = true
		} else if node.Kind() == "identifier" {
			rel.Mores[constants.RelThrowIsRethrow] = true
		}
		return
	}
	if rawText != "" && rel.Source != nil && rel.Source.Extra != nil {
		if ths, ok := rel.Source.Extra.Mores[constants.MethodThrowsTypes].([]string); ok {
			for i, ex := range ths {
				if Clean(ex) == rel.Target.Name {
					rel.Mores[constants.RelThrowIndex] = i
					rel.Mores[constants.RelThrowIsSignature] = true
					break
				}
			}
		}
	}
}

func (e *RelMetadataEnricher) enrichParameterCore(rel *model.DependencyRelation, rawText string) {
	if params, ok := rel.Source.Extra.Mores[constants.MethodParameters].([]string); ok {
		for i, p := range params {
			if strings.Contains(p, rel.Target.Name) || strings.Contains(p, rawText) {
				rel.Mores[constants.RelParameterIndex] = i
				parts := strings.Fields(p)
				if len(parts) >= 2 {
					rel.Mores[constants.RelParameterName] = parts[len(parts)-1]
				}
				if strings.Contains(p, "...") {
					rel.Mores[constants.RelParameterIsVarargs] = true
				}
			}
		}
	}
}

func (e *RelMetadataEnricher) enrichReturnCore(rel *model.DependencyRelation, rawText string) {
	rel.Mores[constants.RelReturnIsPrimitive] = e.resolver.IsPrimitive(Clean(rawText))
	rel.Mores[constants.RelReturnIsArray] = strings.Contains(rawText, "[]")
}

func (e *RelMetadataEnricher) enrichAnnotationCore(rel *model.DependencyRelation) {
	target := e._mapElementKindToAnnotationTarget(rel.Source)
	rel.Mores[constants.RelAnnotationTarget] = target
	rel.Target.Name = strings.Split(rel.Target.Name, "(")[0]
	rel.Target.QualifiedName = strings.Split(rel.Target.QualifiedName, "(")[0]
}

func (e *RelMetadataEnricher) enrichUseCore(rel *model.DependencyRelation, node, ctx *sitter.Node, src []byte) {
	if node == nil || ctx == nil {
		return
	}

	rel.Mores[constants.RelUseTargetName] = node.Utf8Text(src)
	rel.Mores[constants.RelRawText] = ctx.Utf8Text(src)
	rel.Mores[constants.RelAstKind] = node.Kind()

	// 1. 设置 Context 类型 (例如 field_access 或 assignment_expression)
	rel.Mores[constants.RelContext] = ctx.Kind()

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
				rel.Mores[constants.RelUseReceiverType] = Clean(rt)
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

// =============================================================================
// 辅助函数
// =============================================================================

func (e *RelMetadataEnricher) _getRelTmpValue(rel *model.DependencyRelation) (*sitter.Node, string, *sitter.Node) {
	node, _ := rel.Mores[TmpNode].(*sitter.Node)
	rawText, _ := rel.Mores[TmpRaw].(string)
	stmt, _ := rel.Mores[TmpStmt].(*sitter.Node)

	delete(rel.Mores, TmpNode)
	delete(rel.Mores, TmpRaw)
	delete(rel.Mores, TmpStmt)

	return node, rawText, stmt
}

func (e *RelMetadataEnricher) _mapElementKindToAnnotationTarget(elem *model.CodeElement) string {
	switch elem.Kind {
	case model.Class, model.Interface, model.Enum:
		return "TYPE"
	case model.Field:
		return "FIELD"
	case model.Method:
		return "METHOD"
	case model.Variable:
		if isParam, _ := elem.Extra.Mores["java.variable.is_param"].(bool); isParam {
			return "PARAMETER"
		}
		return "LOCAL_VARIABLE"
	}
	return "UNKNOWN"
}

func (e *RelMetadataEnricher) _inferChainedCallReceiverType(node *sitter.Node, src []byte) string {
	// 获取方法名
	methodName := ""
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		methodName = nameNode.Utf8Text(src)
	}
	if methodName == "" {
		return ""
	}

	// 获取 receiver（调用链中前一个调用的接收者）
	var receiver string
	if obj := node.ChildByFieldName("object"); obj != nil {
		receiver = obj.Utf8Text(src)
	}

	// 利用 Resolver 解析这个方法调用（支持作用域回溯、继承链查找、重载匹配）
	target := e.resolver.Resolve(e.gCtx, e.fCtx, node, Clean(receiver), methodName, model.Method)
	if target == nil {
		return ""
	}

	// 从 Target 的 Extra 中获取返回类型 QN
	if target.Extra != nil {
		// 优先使用带 QN 的返回类型
		if returnTypeQN, ok := target.Extra.Mores[constants.MethodReturnTypeWithQN].(string); ok {
			return returnTypeQN
		}

		// 如果没有 QN，尝试从原始类型解析为 QN
		if returnType, ok := target.Extra.Mores[constants.MethodReturnType].(string); ok {
			// 尝试将原始类型解析为 QN
			if returnEle := e.resolver.Resolve(e.gCtx, e.fCtx, nil, "", returnType, model.Class); returnEle != nil {
				return returnEle.QualifiedName
			}
		}
	}

	return ""
}

func (e *RelMetadataEnricher) _inferObjectCreationReceiverType(node *sitter.Node, src []byte) string {
	// 获取创建的对象类型
	var typeName string
	if typeNode := node.ChildByFieldName("type"); typeNode != nil {
		typeName = typeNode.Utf8Text(src)
	}
	if typeName == "" {
		return ""
	}

	// 直接使用 Resolver.resolve 解析类型（不需要 clean，因为 Builder 就是类型名）
	typeEle := e.resolver.Resolve(e.gCtx, e.fCtx, nil, "", typeName, model.Class)
	if typeEle != nil {
		return typeEle.QualifiedName
	}

	// 如果找不到，尝试用 clean 后的类型名
	cleanType := Clean(typeName)
	if cleanType != typeName {
		typeEle = e.resolver.Resolve(e.gCtx, e.fCtx, nil, "", cleanType, model.Class)
		if typeEle != nil {
			return typeEle.QualifiedName
		}
	}

	return ""
}

// _normalizeReceiverText 标准化receiver文本（去除换行符和多余空格）

// _normalizeReceiverText 标准化receiver文本（去除换行符和多余空格）
func (e *RelMetadataEnricher) _normalizeReceiverText(raw string) string {
	normalized := strings.ReplaceAll(raw, "\n", " ")
	normalized = strings.ReplaceAll(normalized, "\r", " ")
	return strings.Join(strings.Fields(normalized), " ")
}

// _calculateChainDepth 计算链式调用的深度
func (e *RelMetadataEnricher) _calculateChainDepth(receiverText string) int {
	depth := 0
	for i := 0; i < len(receiverText); i++ {
		if receiverText[i] == '.' {
			depth++
		}
	}
	if depth == 0 {
		return 1
	}
	return depth + 1
}
