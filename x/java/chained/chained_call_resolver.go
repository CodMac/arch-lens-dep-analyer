package chained

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// ChainedCallResolver 链式调用处理器
// 职责：利用增强后的元数据，对复杂receiver表达式进行精确解析
type ChainedCallResolver struct {
	resolver core.SymbolResolver
	gCtx     *core.GlobalContext
	fCtx     *core.FileContext

	// 方法返回类型缓存，避免重复计算
	methodReturnTypes map[string]string

	// 变量类型缓存，避免重复解析变量
	variableTypesCache map[string]string
}

// NewChainedCallResolver 创建链式调用处理器
func NewChainedCallResolver(resolver core.SymbolResolver, gCtx *core.GlobalContext, fCtx *core.FileContext) *ChainedCallResolver {
	return &ChainedCallResolver{
		resolver:           resolver,
		gCtx:               gCtx,
		fCtx:               fCtx,
		methodReturnTypes:  make(map[string]string),
		variableTypesCache: make(map[string]string),
	}
}

// ProcessChainRels 处理所有包含复杂receiver的调用关系
// 在 enrichRelMetadata 之后调用，此时已收集了充足的元数据信息用于解析
func (ccr *ChainedCallResolver) ProcessChainRels(rel *model.DependencyRelation) {
	// 检查是否为需要重新解析的调用关系
	if !ccr.needsReResolution(rel) {
		return
	}

	ccr.reResolveCallWithChainedReceiver(rel)
}

// needsReResolution 判断是否需要重新解析调用关系
func (ccr *ChainedCallResolver) needsReResolution(rel *model.DependencyRelation) bool {
	// 目标被标记为外部方法，但我们要尝试在本地重新解析
	if rel.Target.IsFormExternal {
		return true
	}

	// 目标的QN不完整（缺少类名前缀）
	targetQN := rel.Target.QualifiedName
	if !strings.Contains(targetQN, ".") || (strings.Count(targetQN, ".") == 0 && strings.Contains(targetQN, "()")) {
		return true
	}

	// 检查receiver类型是否为空，如果是则尝试重新解析
	if receiverType, ok := rel.Mores[constants.RelCallReceiverType].(string); !ok || receiverType == "" {
		return true
	}

	return false
}

// reResolveCallWithChainedReceiver 使用链式调用信息重新解析目标方法
func (ccr *ChainedCallResolver) reResolveCallWithChainedReceiver(rel *model.DependencyRelation) {
	// 从元数据中获取原始receiver表达式和调用节点信息
	receiverRaw, hasReceiverRaw := rel.Mores[constants.RelCallReceiverRaw].(string)
	tmpNode, hasTmpNode := rel.Mores["tmp_node"].(interface{})

	if !hasReceiverRaw || !hasTmpNode {
		return
	}

	// 尝试将interface{}转换为sitter.Node
	node, ok := tmpNode.(*sitter.Node)
	if !ok {
		return
	}

	// 解析复杂receiver表达式，获取最终的类型QN
	receiverTypeQN := ccr.resolveComplexReceiver(receiverRaw)
	if receiverTypeQN == "" {
		return
	}

	// 使用正确的receiver类型重新解析目标方法
	methodName := rel.Target.Name
	newTarget := ccr.resolver.Resolve(ccr.gCtx, ccr.fCtx, node, receiverTypeQN, methodName, model.Method)

	if newTarget != nil {
		rel.Target = newTarget

		// 更新链式调用元数据
		rel.Mores[constants.RelCallIsChained] = true
		rel.Mores[constants.RelCallReceiverType] = receiverTypeQN

		// 如果是链式调用，记录调用深度
		chainDepth := ccr.calculateChainDepth(receiverRaw)
		if chainDepth > 1 {
			rel.Mores[constants.RelCallChainDepth] = chainDepth
		}
	} else {
		// 如果仍然无法解析，构建完整的QN并更新Target
		completeQN := receiverTypeQN + "." + methodName
		if entry, ok := ccr.gCtx.FindByQualifiedName(completeQN); ok && entry.Element.Kind == model.Method {
			rel.Target = entry.Element
		} else {
			rel.Target.QualifiedName = completeQN
			rel.Target.IsFormExternal = false
		}
	}
}

// resolveComplexReceiver 解析复杂的receiver表达式
func (ccr *ChainedCallResolver) resolveComplexReceiver(receiverExpr string) string {
	receiverExpr = strings.TrimSpace(receiverExpr)

	// 简化版实现：处理常见的链式调用模式
	// 1. 检查是否是方法调用表达式（包含括号）
	if strings.Contains(receiverExpr, "(") && strings.Contains(receiverExpr, ")") {
		return ccr.resolveMethodCallExpression(receiverExpr)
	}

	// 2. 检查是否是对象创建表达式
	if strings.HasPrefix(receiverExpr, "new ") {
		return ccr.resolveObjectCreationExpression(receiverExpr)
	}

	// 3. 检查是否是简单的变量名或类名
	if !strings.Contains(receiverExpr, ".") {
		return ccr.resolveSimpleReceiver(receiverExpr)
	}

	return ""
}

// resolveMethodCallExpression 解析方法调用表达式，如 "obj.method1().method2()"
func (ccr *ChainedCallResolver) resolveMethodCallExpression(expr string) string {
	// 移除换行符和多余空格
	expr = strings.Join(strings.Fields(expr), "")

	// 解析成调用链部分并逐层处理
	return ccr.resolveChainedCallLayerByLayer(expr)
}

// resolveChainedCallLayerByLayer 逐层解析链式调用
func (ccr *ChainedCallResolver) resolveChainedCallLayerByLayer(expr string) string {
	// 将表达式拆分为调用链的各个部分
	// 例如 "obj.method1().method2().method3()" → ["obj", "method1()", "method2()", "method3()"]

	// 解析成调用链部分
	chainParts := ccr._parseCallChain(expr)
	if len(chainParts) == 0 {
		return ""
	}

	// 逐步处理调用链
	currentType := ""

	for i := 0; i < len(chainParts); i++ {
		part := chainParts[i]

		if i == 0 {
			// 第一部分是起始者（变量或表达式）
			currentType = ccr.resolveComplexReceiver(part)
		} else {
			// 后续部分是方法调用
			methodName := ccr._extractMethodName(part)

			// 查找方法的返回类型
			if currentType != "" && methodName != "" {
				methodQN := currentType + "." + methodName
				if returnType, ok := ccr.methodReturnTypes[methodQN]; ok {
					currentType = returnType
				} else {
					// 如果缓存中没有，尝试重新解析
					if entry, ok := ccr.gCtx.FindByQualifiedName(methodQN); ok && entry.Element.Kind == model.Method {
						if entry.Element.Extra != nil {
							if returnTypeQN, ok := entry.Element.Extra.Mores[constants.MethodReturnTypeWithQN].(string); ok {
								currentType = returnTypeQN
							} else if returnType, ok := entry.Element.Extra.Mores[constants.MethodReturnType].(string); ok {
								currentType = returnType
							}
						}
					}
				}
			}
		}

		// 如果任何一个环节解析失败，返回当前已解析的类型（可能不完整但有用）
		if currentType == "" && i > 0 {
			break
		}
	}

	return currentType
}

// _extractMethodName 从方法调用部分提取方法名
func (ccr *ChainedCallResolver) _extractMethodName(methodCall string) string {
	methodCall = strings.TrimSpace(methodCall)

	// 移除括号和参数
	if idx := strings.Index(methodCall, "("); idx != -1 {
		return methodCall[:idx]
	}

	return methodCall
}

// _parseCallChain 将链式调用表达式解析为各个部分
func (ccr *ChainedCallResolver) _parseCallChain(expr string) []string {
	var parts []string
	current := ""
	depth := 0

	for _, ch := range expr {
		switch ch {
		case '.':
			if depth == 0 && current != "" {
				parts = append(parts, current)
				current = ""
			} else {
				current += string(ch)
			}
		case '(':
			current += string(ch)
			depth++
		case ')':
			current += string(ch)
			depth--
		default:
			current += string(ch)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// resolveObjectCreationExpression 解析对象创建表达式，如 "new Builder()"
func (ccr *ChainedCallResolver) resolveObjectCreationExpression(expr string) string {
	// 去掉 "new " 前缀
	typeName := strings.TrimSpace(strings.TrimPrefix(expr, "new"))

	// 移除可能的参数部分
	if idx := strings.Index(typeName, "("); idx != -1 {
		typeName = typeName[:idx]
	}

	// 解析类型
	if typeEle := ccr.resolver.Resolve(ccr.gCtx, ccr.fCtx, nil, "", typeName, model.Class); typeEle != nil {
		return typeEle.QualifiedName
	}

	return typeName // 返回原始类型名作为fallback
}

// resolveSimpleReceiver 解析简单的receiver项（变量名或类名）
func (ccr *ChainedCallResolver) resolveSimpleReceiver(receiverExpr string) string {
	// 移除换行符和空格
	receiverExpr = strings.TrimSpace(receiverExpr)

	// 特殊关键字处理
	if receiverExpr == "this" || receiverExpr == "super" {
		return ccr.getCurrentClassType()
	}

	// 尝试解析为变量
	if varType, ok := ccr.variableTypesCache[receiverExpr]; ok {
		return varType
	}

	varType := ccr._resolveVariableType(receiverExpr)
	if varType != "" {
		ccr.variableTypesCache[receiverExpr] = varType
		return varType
	}

	// 尝试解析为类名（静态调用）
	if typeEle := ccr.resolver.Resolve(ccr.gCtx, ccr.fCtx, nil, "", receiverExpr, model.Class); typeEle != nil {
		return typeEle.QualifiedName
	}

	return ""
}

// _resolveVariableType 解析变量的类型
func (ccr *ChainedCallResolver) _resolveVariableType(varName string) string {
	// 尝试从上下文中查找变量定义
	if entries, ok := ccr.fCtx.FindByShortName(varName); ok && len(entries) > 0 {
		for _, entry := range entries {
			if entry.Element.Kind == model.Variable || entry.Element.Kind == model.Field {
				if entry.Element.Extra != nil {
					if typeQN, ok := entry.Element.Extra.Mores[constants.VariableTypeWithQN].(string); ok {
						return typeQN
					}
					if rawType, ok := entry.Element.Extra.Mores[constants.VariableRawType].(string); ok {
						if typeEle := ccr.resolver.Resolve(ccr.gCtx, ccr.fCtx, nil, "", rawType, model.Class); typeEle != nil {
							return typeEle.QualifiedName
						}
						return rawType
					}
				}
			}
		}
	}

	return ""
}

// getCurrentClassType 获取当前类的类型
func (ccr *ChainedCallResolver) getCurrentClassType() string {
	// 简化实现：返回文件中找到的第一个类
	for _, entry := range ccr.fCtx.Definitions {
		if entry.Element.Kind == model.Class || entry.Element.Kind == model.Interface {
			return entry.Element.QualifiedName
		}
	}
	return ""
}

// calculateChainDepth 计算链式调用的深度
func (ccr *ChainedCallResolver) calculateChainDepth(receiverExpr string) int {
	depth := 0
	for i := 0; i < len(receiverExpr); i++ {
		if receiverExpr[i] == '.' {
			depth++
		}
	}
	return depth + 1 // 至少是1层调用
}
