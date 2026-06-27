package resolver

import (
	"fmt"
	"slices"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// MethodResolver 方法解析器
type MethodResolver struct {
	gCtx *core.GlobalContext
	fCtx *core.FileContext
	src  []byte
	node *sitter.Node
}

// NewMethodResolver 创建方法解析器
func NewMethodResolver(gCtx *core.GlobalContext, fCtx *core.FileContext, node *sitter.Node) *MethodResolver {
	src := *fCtx.SourceBytes
	return &MethodResolver{
		gCtx: gCtx,
		fCtx: fCtx,
		src:  src,
		node: node,
	}
}

// ResolveWithReceiver 使用Receiver解析方法
func (mr *MethodResolver) ResolveWithReceiver(receiver *Receiver, methodName string) *model.CodeElement {
	if receiver == nil || methodName == "" {
		return nil
	}

	methodName = helper.Clean(methodName)

	// 处理不同类型的receiver
	switch receiver.Type {
	case ReceiverChained:
		return mr.resolveChainedReceiver(receiver, methodName)
	case ReceiverThis, ReceiverSuper:
		return mr.resolveThisSuperReceiver(receiver, methodName)
	case ReceiverClassName:
		return mr.resolveClassReceiver(receiver, methodName)
	case ReceiverVariable:
		return mr.resolveVariableReceiver(receiver, methodName)
	case ReceiverField:
		return mr.resolveFieldReceiver(receiver, methodName)
	default:
		// ReceiverNone，从当前类查找
		return mr.ResolveInCurrentClass(methodName)
	}
}

// resolveChainedReceiver 处理链式调用中的方法解析
func (mr *MethodResolver) resolveChainedReceiver(receiver *Receiver, methodName string) *model.CodeElement {
	if receiver.Chained == nil || len(receiver.Chained.Steps) == 0 {
		return nil
	}

	// 先解析链式调用本身
	if !receiver.Chained.Resolved {
		if err := mr.resolveChainedContext(receiver.Chained); err != nil {
			return nil
		}
	}

	// 在链式调用的最终结果上调用方法
	if receiver.Chained.CurrentType != nil {
		return mr.resolveInElementHierarchy(receiver.Chained.CurrentType, methodName, false)
	}

	return nil
}

// resolveThisSuperReceiver 处理this/super receiver的方法调用
func (mr *MethodResolver) resolveThisSuperReceiver(receiver *Receiver, methodName string) *model.CodeElement {
	container := helper.GetBestElement(mr.fCtx, mr.node, []model.ElementKind{model.Class, model.AnonymousClass})
	if container == nil {
		return mr.resolveExternalMethod(methodName)
	}

	var targetContainer *model.CodeElement
	isStaticCall := false

	if receiver.Type == ReceiverSuper {
		// super调用指向父类
		if container.Extra != nil {
			if superClass, ok := container.Extra.Mores[constants.ClassSuperClass].(string); ok && superClass != "" {
				parents := helper.PreciseResolve(mr.gCtx, mr.fCtx, helper.Clean(superClass))
				if len(parents) > 0 {
					targetContainer = parents[0].Element
				}
			}
		}
	} else {
		// this调用在当前类
		targetContainer = container
		isStaticCall = slices.Contains(container.Extra.Modifiers, "static")
	}

	if targetContainer == nil {
		return mr.resolveExternalMethod(methodName)
	}

	return mr.resolveInElementHierarchy(targetContainer, methodName, isStaticCall)
}

// resolveClassReceiver 处理类名receiver（静态方法调用）
func (mr *MethodResolver) resolveClassReceiver(receiver *Receiver, methodName string) *model.CodeElement {
	// 解析receiver对应的类
	entries := helper.PreciseResolve(mr.gCtx, mr.fCtx, receiver.RawText)
	if len(entries) == 0 {
		return mr.resolveExternalMethod(methodName)
	}

	classEle := entries[0].Element
	if classEle.Kind != model.Class && classEle.Kind != model.Interface && classEle.Kind != model.Enum {
		return mr.resolveExternalMethod(methodName)
	}

	return mr.resolveInElementHierarchy(classEle, methodName, true)
}

// resolveVariableReceiver 处理变量名receiver的方法调用
func (mr *MethodResolver) resolveVariableReceiver(receiver *Receiver, methodName string) *model.CodeElement {
	// 先解析出receiver本身的类型
	receiverEle := mr.resolveVariable(receiver.RawText)
	if receiverEle == nil {
		return mr.resolveExternalMethod(methodName)
	}

	// 从receiver的类型信息中获取实际类型
	typeQN := mr.extractTypeQN(receiverEle)
	if typeQN == "" {
		return mr.resolveExternalMethod(methodName)
	}

	// 解析类型对应的元素
	entries := helper.PreciseResolve(mr.gCtx, mr.fCtx, typeQN)
	if len(entries) == 0 {
		return mr.resolveExternalMethod(methodName)
	}

	typeElement := entries[0].Element
	return mr.resolveInElementHierarchy(typeElement, methodName, false)
}

// resolveFieldReceiver 处理字段访问receiver的方法调用
func (mr *MethodResolver) resolveFieldReceiver(receiver *Receiver, methodName string) *model.CodeElement {
	// 先解析字段访问表达式的类型
	fieldType := mr.resolveFieldType(receiver.Node)
	if fieldType == nil {
		return mr.resolveExternalMethod(methodName)
	}

	return mr.resolveInElementHierarchy(fieldType, methodName, false)
}

// ResolveInCurrentClass 在当前类中查找方法
func (mr *MethodResolver) ResolveInCurrentClass(methodName string) *model.CodeElement {
	container := helper.GetBestElement(mr.fCtx, mr.node, []model.ElementKind{model.Class, model.AnonymousClass})
	if container == nil {
		return mr.resolveExternalMethod(methodName)
	}

	isStaticCall := slices.Contains(container.Extra.Modifiers, "static")
	return mr.resolveInElementHierarchy(container, methodName, isStaticCall)
}

// resolveInElementHierarchy 在给定的元素层级中查找方法
func (mr *MethodResolver) resolveInElementHierarchy(element *model.CodeElement, methodName string, isStaticCall bool) *model.CodeElement {
	// 准备参数信息
	argCount, inferredTypes := mr.extractArguments()

	// 在元素中搜索方法
	result := mr.searchMethodInHierarchy(element, methodName, argCount, inferredTypes, isStaticCall, element)
	if result != nil {
		return result
	}

	return mr.resolveExternalMethod(methodName)
}

// searchMethodInHierarchy 递归搜索方法
func (mr *MethodResolver) searchMethodInHierarchy(currContainer *model.CodeElement, methodName string, argCount int, inferredTypes []string, isStaticCall bool, container *model.CodeElement) *model.CodeElement {
	if currContainer == nil {
		return nil
	}

	// 查找当前容器内的所有同名方法
	targetPrefix := currContainer.QualifiedName + "." + methodName
	var candidates []*core.DefinitionEntry

	if entries, ok := mr.gCtx.FindMethodByNoParamsQN(targetPrefix); ok {
		for _, e := range entries {
			if e.Element.Kind != model.Method {
				continue
			}
			// 静态检查
			if isStaticCall && !slices.Contains(e.Element.Extra.Modifiers, "static") {
				continue
			}
			// 可见性检查
			if mr.checkVisibility(container, e) {
				candidates = append(candidates, e)
			}
		}
	}

	// 如果有候选方法，进行重载匹配
	if len(candidates) > 0 {
		return mr.pickBestOverload(candidates, argCount, inferredTypes)
	}

	if currContainer.Extra == nil {
		return nil
	}

	// 搜索父类
	if sc, ok := currContainer.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
		parents := helper.PreciseResolve(mr.gCtx, mr.fCtx, helper.Clean(sc))
		if len(parents) > 0 {
			if result := mr.searchMethodInHierarchy(parents[0].Element, methodName, argCount, inferredTypes, isStaticCall, container); result != nil {
				return result
			}
		}
	}

	// 搜索接口
	if itfs, ok := currContainer.Extra.Mores[constants.ClassImplementedInterfaces].([]string); ok {
		for _, itf := range itfs {
			parents := helper.PreciseResolve(mr.gCtx, mr.fCtx, helper.Clean(itf))
			if len(parents) > 0 {
				if result := mr.searchMethodInHierarchy(parents[0].Element, methodName, argCount, inferredTypes, isStaticCall, container); result != nil {
					return result
				}
			}
		}
	}

	return nil
}

// pickBestOverload 选择最优的重载方法
func (mr *MethodResolver) pickBestOverload(candidates []*core.DefinitionEntry, argCount int, inferredTypes []string) *model.CodeElement {
	var bestMatch *model.CodeElement
	maxScore := -1

	for _, candidate := range candidates {
		score := mr.calculateOverloadScore(candidate, argCount, inferredTypes)
		if score > maxScore {
			maxScore = score
			bestMatch = candidate.Element
		}
	}

	if bestMatch != nil {
		return bestMatch
	}
	return candidates[0].Element
}

// calculateOverloadScore 计算重载方法的匹配分数
func (mr *MethodResolver) calculateOverloadScore(entry *core.DefinitionEntry, argCount int, inferredTypes []string) int {
	score := 0

	// 获取方法参数信息
	params, ok := entry.Element.Extra.Mores[constants.MethodParametersWithQN].([]string)
	if !ok {
		return 0
	}

	definedParamCount := len(params)

	// 参数数量严格匹配
	if definedParamCount == argCount {
		score += 100

		// 参数类型匹配
		for i := 0; i < argCount; i++ {
			if i >= len(inferredTypes) {
				break
			}

			definedTypeQN := helper.Clean(params[i])
			inferredType := inferredTypes[i]

			if inferredType == "unknown" || inferredType == "null" {
				score += 10
				continue
			}

			if definedTypeQN == inferredType || strings.HasSuffix(definedTypeQN, "."+inferredType) {
				score += 50
			}
		}
	}

	return score
}

// resolveVariable 解析变量
func (mr *MethodResolver) resolveVariable(varName string) *model.CodeElement {
	container := helper.GetBestElement(mr.fCtx, mr.node, []model.ElementKind{model.Method, model.Class, model.ScopeBlock})
	if container == nil {
		return nil
	}

	targetQN := mr.buildQN(container.QualifiedName, varName)
	if entry, ok := mr.gCtx.FindByQualifiedName(targetQN); ok {
		return entry.Element
	}

	return mr.resolveVariableInParent(container.QualifiedName, varName)
}

// resolveVariableInParent 在父作用域中查找变量
func (mr *MethodResolver) resolveVariableInParent(parentQN, varName string) *model.CodeElement {
	targetQN := mr.buildQN(parentQN, varName)
	if entry, ok := mr.gCtx.FindByQualifiedName(targetQN); ok {
		return entry.Element
	}

	parentEntry, ok := mr.gCtx.FindByQualifiedName(parentQN)
	if !ok {
		return nil
	}

	return mr.resolveVariableInParent(parentEntry.ParentQN, varName)
}

// resolveFieldType 解析字段访问表达式的类型
func (mr *MethodResolver) resolveFieldType(fieldName *sitter.Node) *model.CodeElement {
	var rawText string
	switch fieldName.Kind() {
	case "identifier":
		rawText = fieldName.Utf8Text(mr.src)
	case "field_access":
		objectNode := fieldName.ChildByFieldName("object")
		fieldNode := fieldName.ChildByFieldName("field")

		if objectNode != nil && fieldNode != nil {
			fieldNameStr := fieldNode.Utf8Text(mr.src)
			container := mr.resolveFieldType(objectNode)

			if container != nil {
				return mr.resolveInElementHierarchy(container, fieldNameStr, false)
			}
		}
		return nil
	default:
		return nil
	}

	return mr.resolveVariable(rawText)
}

// resolveChainedContext 解析链式调用上下文（方法版）
func (mr *MethodResolver) resolveChainedContext(changedCtx *ChainedContext) error {
	if len(changedCtx.Steps) == 0 {
		return fmt.Errorf("empty chain steps")
	}

	var currentType *model.CodeElement

	// 逐步解析链式调用
	for i, step := range changedCtx.Steps {
		if i == 0 {
			currentType = mr.resolveBaseStep(step)
		} else {
			currentType = mr.resolveNextStep(step, currentType)
		}

		if currentType == nil {
			changedCtx.Error = fmt.Errorf("failed to resolve step %d: %s", i, step.Name)
			return changedCtx.Error
		}
	}

	changedCtx.CurrentType = currentType
	changedCtx.Resolved = true
	return nil
}

// resolveBaseStep 解析基础步骤（方法版）
func (mr *MethodResolver) resolveBaseStep(step ChainStep) *model.CodeElement {
	if step.IsNew {
		typeName := strings.TrimPrefix(step.RawText, "new")
		typeName = strings.TrimSpace(typeName)
		if idx := strings.Index(typeName, "("); idx != -1 {
			typeName = typeName[:idx]
		}

		entries := helper.PreciseResolve(mr.gCtx, mr.fCtx, typeName)
		if len(entries) > 0 {
			return entries[0].Element
		}
		return nil
	}

	if step.Name == "this" {
		return helper.GetBestElement(mr.fCtx, mr.node, []model.ElementKind{model.Class, model.AnonymousClass})
	}
	if step.Name == "super" {
		container := helper.GetBestElement(mr.fCtx, mr.node, []model.ElementKind{model.Class, model.AnonymousClass})
		if container != nil && container.Extra != nil {
			if superClass, ok := container.Extra.Mores[constants.ClassSuperClass].(string); ok && superClass != "" {
				entries := helper.PreciseResolve(mr.gCtx, mr.fCtx, helper.Clean(superClass))
				if len(entries) > 0 {
					return entries[0].Element
				}
			}
		}
		return nil
	}

	return mr.resolveVariable(step.Name)
}

// resolveNextStep 解析后续步骤（方法版）
func (mr *MethodResolver) resolveNextStep(step ChainStep, currentType *model.CodeElement) *model.CodeElement {
	if currentType == nil {
		return nil
	}

	if step.IsCall {
		// 方法调用：需要解析方法返回类型
		argCount, _ := mr.extractArgumentsFromNode(step.ASTNode)
		methodEntry := mr.searchMethodInHierarchy(currentType, step.Name, argCount, []string{}, false, currentType)

		if methodEntry != nil && methodEntry.Extra != nil {
			if returnTypeQN, ok := methodEntry.Extra.Mores[constants.MethodReturnTypeWithQN].(string); ok {
				entries := helper.PreciseResolve(mr.gCtx, mr.fCtx, returnTypeQN)
				if len(entries) > 0 {
					return entries[0].Element
				}
			} else if returnType, ok := methodEntry.Extra.Mores[constants.MethodReturnType].(string); ok {
				entries := helper.PreciseResolve(mr.gCtx, mr.fCtx, returnType)
				if len(entries) > 0 {
					return entries[0].Element
				}
			}
		}
		return nil
	} else if step.IsField {
		// 字段访问：获取字段类型
		return mr.resolveInElementHierarchy(currentType, step.Name, false)
	}

	return nil
}

// extractArguments 从当前节点提取参数信息
func (mr *MethodResolver) extractArguments() (int, []string) {
	return mr.extractArgumentsFromNode(mr.node)
}

// extractArgumentsFromNode 从指定节点提取参数信息
func (mr *MethodResolver) extractArgumentsFromNode(node *sitter.Node) (int, []string) {
	argCount := 0
	var inferredTypes []string

	if node == nil {
		return 0, inferredTypes
	}

	invNode := helper.FindNearestKind(node, "method_invocation", "object_creation_expression", "explicit_constructor_invocation")
	if invNode == nil {
		return 0, inferredTypes
	}

	if args := invNode.ChildByFieldName("arguments"); args != nil {
		argCount = int(args.NamedChildCount())
		inferredTypes = mr.inferArgumentTypes(args)
	}

	return argCount, inferredTypes
}

// inferArgumentTypes 推断参数类型
func (mr *MethodResolver) inferArgumentTypes(argsNode *sitter.Node) []string {
	var types []string

	for i := 0; i < int(argsNode.NamedChildCount()); i++ {
		arg := argsNode.NamedChild(uint(i))
		kind := arg.Kind()

		switch kind {
		case "string_literal":
			types = append(types, "String")
		case "decimal_integer_literal", "hex_integer_literal":
			types = append(types, "int")
		case "decimal_floating_point_literal":
			types = append(types, "double")
		case "true", "false", "boolean_type":
			types = append(types, "boolean")
		case "null_literal":
			types = append(types, "null")
		case "object_creation_expression", "cast_expression":
			if typeNode := arg.ChildByFieldName("type"); typeNode != nil {
				types = append(types, helper.GetNodeContent(typeNode, mr.src))
			} else {
				types = append(types, "unknown")
			}
		case "array_creation_expression":
			if typeNode := arg.ChildByFieldName("type"); typeNode != nil {
				types = append(types, helper.GetNodeContent(typeNode, mr.src)+"[]")
			} else {
				types = append(types, "unknown")
			}
		default:
			types = append(types, "unknown")
		}
	}

	return types
}

// resolveExternalMethod 解析外部方法
func (mr *MethodResolver) resolveExternalMethod(methodName string) *model.CodeElement {
	return &model.CodeElement{
		Name:           methodName,
		QualifiedName:  methodName,
		Kind:           model.Method,
		IsFormExternal: true,
	}
}

// checkVisibility 检查可见性
func (mr *MethodResolver) checkVisibility(container *model.CodeElement, target *core.DefinitionEntry) bool {
	if container == nil || target.Element == nil {
		return false
	}

	containerOutermost := helper.GetOutermostClassQN(container.QualifiedName)
	targetOutermost := helper.GetOutermostClassQN(target.Element.QualifiedName)
	if containerOutermost != "" && containerOutermost == targetOutermost {
		return true
	}

	if target.Element.Extra == nil || target.Element.Extra.Modifiers == nil {
		return false
	}

	mods := target.Element.Extra.Modifiers
	if slices.Contains(mods, "public") {
		return true
	}

	targetPkg := helper.GetRealPackage(mr.gCtx, target.Element)
	if targetPkg == mr.fCtx.PackageName {
		return true
	}

	if slices.Contains(mods, "protected") {
		sourceClass := helper.GetOwnerClassQN(mr.gCtx, container)
		return helper.IsSubClassOf(mr.gCtx, mr.fCtx, sourceClass, target.ParentQN)
	}

	return false
}

// extractTypeQN 从代码元素中提取类型QN
func (mr *MethodResolver) extractTypeQN(element *model.CodeElement) string {
	if element == nil || element.Extra == nil {
		return ""
	}

	if typeQN, ok := element.Extra.Mores[constants.VariableTypeWithQN].(string); ok {
		return typeQN
	} else if rawType, ok := element.Extra.Mores[constants.VariableRawType].(string); ok {
		return rawType
	}

	return ""
}

// buildQN 构建限定名
func (mr *MethodResolver) buildQN(parentQN, name string) string {
	if parentQN == "" || parentQN == "." {
		return name
	}
	return parentQN + "." + name
}
