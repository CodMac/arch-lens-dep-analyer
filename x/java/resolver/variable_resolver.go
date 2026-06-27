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

// VariableResolver 变量解析器
type VariableResolver struct {
	gCtx *core.GlobalContext
	fCtx *core.FileContext
	src  []byte
	node *sitter.Node
}

// NewVariableResolver 创建变量解析器
func NewVariableResolver(gCtx *core.GlobalContext, fCtx *core.FileContext, node *sitter.Node) *VariableResolver {
	src := *fCtx.SourceBytes
	return &VariableResolver{
		gCtx: gCtx,
		fCtx: fCtx,
		src:  src,
		node: node,
	}
}

// ResolveWithReceiver 使用Receiver解析变量
func (vr *VariableResolver) ResolveWithReceiver(receiver *Receiver, symbol string) *model.CodeElement {
	if receiver == nil || symbol == "" {
		return nil
	}

	symbol = helper.Clean(symbol)

	// 处理不同类型的receiver
	switch receiver.Type {
	case ReceiverChained:
		return vr.resolveChainedReceiver(receiver, symbol)
	case ReceiverThis, ReceiverSuper:
		return vr.resolveThisSuperReceiver(receiver, symbol)
	case ReceiverClassName:
		return vr.resolveClassReceiver(receiver, symbol)
	case ReceiverVariable:
		return vr.resolveVariableReceiver(receiver, symbol)
	case ReceiverField:
		return vr.resolveFieldReceiver(receiver, symbol)
	default:
		// ReceiverNone，在当前作用域查找
		return vr.ResolveInCurrentScope(symbol)
	}
}

// resolveChainedReceiver 处理链式调用中的变量解析
func (vr *VariableResolver) resolveChainedReceiver(receiver *Receiver, symbol string) *model.CodeElement {
	if receiver.Chained == nil || len(receiver.Chained.Steps) == 0 {
		return nil
	}

	// 先解析链式调用本身
	if !receiver.Chained.Resolved {
		if err := vr.resolveChainedContext(receiver.Chained); err != nil {
			return nil
		}
	}

	// 在链式调用的最终结果上访问symbol
	if receiver.Chained.CurrentType != nil {
		return vr.resolveInElementHierarchy(receiver.Chained.CurrentType, symbol, false)
	}

	return nil
}

// resolveThisSuperReceiver 处理this/super receiver
func (vr *VariableResolver) resolveThisSuperReceiver(receiver *Receiver, symbol string) *model.CodeElement {
	container := helper.GetBestElement(vr.fCtx, vr.node, []model.ElementKind{model.Class, model.AnonymousClass})
	if container == nil {
		return nil
	}

	isStatic := slices.Contains(container.Extra.Modifiers, "static")

	if receiver.Type == ReceiverSuper {
		return vr.resolveFromInheritance(container, symbol, isStatic, container)
	}

	return vr.resolveInScopeHierarchy(container.QualifiedName, symbol, isStatic, container)
}

// resolveClassReceiver 处理类名receiver（静态字段访问）
func (vr *VariableResolver) resolveClassReceiver(receiver *Receiver, symbol string) *model.CodeElement {
	// 解析receiver对应的类
	entries := helper.PreciseResolve(vr.gCtx, vr.fCtx, receiver.RawText)
	if len(entries) == 0 {
		return nil
	}

	classEle := entries[0].Element
	if classEle.Kind != model.Class && classEle.Kind != model.Interface {
		return nil
	}

	return vr.resolveInElementHierarchy(classEle, symbol, true)
}

// resolveVariableReceiver 处理变量名receiver
func (vr *VariableResolver) resolveVariableReceiver(receiver *Receiver, symbol string) *model.CodeElement {
	// 先解析出receiver本身的类型
	receiverEle := vr.resolveVariable(receiver.RawText)
	if receiverEle == nil || receiverEle.Extra == nil {
		return nil
	}

	// 从receiver的类型信息中获取实际类型
	var typeQN string
	if typeName, ok := receiverEle.Extra.Mores[constants.VariableTypeWithQN].(string); ok {
		typeQN = typeName
	} else if typeName, ok := receiverEle.Extra.Mores[constants.VariableRawType].(string); ok {
		typeQN = typeName
	}

	if typeQN == "" {
		return nil
	}

	// 解析类型对应的元素
	entries := helper.PreciseResolve(vr.gCtx, vr.fCtx, typeQN)
	if len(entries) == 0 {
		return nil
	}

	typeElement := entries[0].Element
	return vr.resolveInElementHierarchy(typeElement, symbol, false)
}

// resolveFieldReceiver 处理字段访问receiver
func (vr *VariableResolver) resolveFieldReceiver(receiver *Receiver, symbol string) *model.CodeElement {
	// 先解析字段访问表达式的类型
	fieldType := vr.resolveFieldType(receiver.Node)
	if fieldType == nil {
		return nil
	}

	return vr.resolveInElementHierarchy(fieldType, symbol, false)
}

// ResolveInCurrentScope 在当前作用域查找变量
func (vr *VariableResolver) ResolveInCurrentScope(symbol string) *model.CodeElement {
	container := helper.GetBestElement(vr.fCtx, vr.node, []model.ElementKind{model.Method, model.Class, model.ScopeBlock})
	if container == nil {
		return nil
	}

	isStatic := slices.Contains(container.Extra.Modifiers, "static")
	return vr.resolveInScopeHierarchy(container.QualifiedName, symbol, isStatic, container)
}

// resolveInScopeHierarchy 递归查找变量
func (vr *VariableResolver) resolveInScopeHierarchy(previousQN, symbol string, isStatic bool, container *model.CodeElement) *model.CodeElement {
	if previousQN == "" {
		return nil
	}

	targetQN := vr.buildQN(previousQN, symbol)
	if entry, ok := vr.gCtx.FindByQualifiedName(targetQN); ok {
		if vr.checkVisibility(container, entry) {
			isIllegalStatic := isStatic && entry.Element.Kind == model.Field && !slices.Contains(entry.Element.Extra.Modifiers, "static")
			if !isIllegalStatic {
				return entry.Element
			}
		}
	}

	previousEntry, ok := vr.gCtx.FindByQualifiedName(previousQN)
	if !ok {
		return nil
	}

	// 搜索继承链
	previousEleKind := previousEntry.Element.Kind
	if previousEleKind == model.Class || previousEleKind == model.Interface || previousEleKind == model.AnonymousClass {
		if inherited := vr.resolveFromInheritance(previousEntry.Element, symbol, isStatic, container); inherited != nil {
			return inherited
		}
	}

	// 递归到上一层作用域
	return vr.resolveInScopeHierarchy(previousEntry.ParentQN, symbol, isStatic, container)
}

// resolveFromInheritance 从继承链中查找
func (vr *VariableResolver) resolveFromInheritance(elem *model.CodeElement, symbol string, isStatic bool, sourceElem *model.CodeElement) *model.CodeElement {
	if elem.Extra == nil {
		return nil
	}

	var superTargets []string
	if sc, ok := elem.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
		superTargets = append(superTargets, sc)
	}
	if itfs, ok := elem.Extra.Mores[constants.ClassImplementedInterfaces].([]string); ok {
		superTargets = append(superTargets, itfs...)
	}

	for _, rawSuperName := range superTargets {
		cleanSuperName := strings.Split(rawSuperName, "<")[0]
		parentEntries := helper.PreciseResolve(vr.gCtx, vr.fCtx, cleanSuperName)

		if len(parentEntries) > 0 {
			parentElem := parentEntries[0].Element
			targetQN := vr.buildQN(parentElem.QualifiedName, symbol)

			if fieldEntry, ok := vr.gCtx.FindByQualifiedName(targetQN); ok {
				if vr.checkVisibility(sourceElem, fieldEntry) {
					if !isStatic || slices.Contains(fieldEntry.Element.Extra.Modifiers, "static") {
						return fieldEntry.Element
					}
				}
			}

			if found := vr.resolveFromInheritance(parentElem, symbol, isStatic, sourceElem); found != nil {
				return found
			}
		}
	}
	return nil
}

// resolveInElementHierarchy 在给定的元素层级中查找
func (vr *VariableResolver) resolveInElementHierarchy(element *model.CodeElement, symbol string, isStatic bool) *model.CodeElement {
	targetQN := vr.buildQN(element.QualifiedName, symbol)
	if entry, ok := vr.gCtx.FindByQualifiedName(targetQN); ok {
		if vr.checkVisibility(element, entry) {
			return entry.Element
		}
	}

	return vr.resolveFromInheritance(element, symbol, isStatic, element)
}

// resolveVariable 解析变量类型
func (vr *VariableResolver) resolveVariable(varName string) *model.CodeElement {
	// 先在当前作用域查找变量定义
	container := helper.GetBestElement(vr.fCtx, vr.node, []model.ElementKind{model.Method, model.Class, model.ScopeBlock})
	if container == nil {
		return nil
	}

	targetQN := vr.buildQN(container.QualifiedName, varName)
	if entry, ok := vr.gCtx.FindByQualifiedName(targetQN); ok {
		return entry.Element
	}

	// 递归查找父作用域
	return vr.resolveVariableInParent(container.QualifiedName, varName)
}

// resolveVariableInParent 在父作用域中查找变量
func (vr *VariableResolver) resolveVariableInParent(parentQN, varName string) *model.CodeElement {
	targetQN := vr.buildQN(parentQN, varName)
	if entry, ok := vr.gCtx.FindByQualifiedName(targetQN); ok {
		return entry.Element
	}

	parentEntry, ok := vr.gCtx.FindByQualifiedName(parentQN)
	if !ok {
		return nil
	}

	return vr.resolveVariableInParent(parentEntry.ParentQN, varName)
}

// resolveFieldType 解析字段访问表达式的类型
func (vr *VariableResolver) resolveFieldType(fieldName *sitter.Node) *model.CodeElement {
	var rawText string
	switch fieldName.Kind() {
	case "identifier":
		rawText = fieldName.Utf8Text(vr.src)
	case "field_access":
		objectNode := fieldName.ChildByFieldName("object")
		fieldNode := fieldName.ChildByFieldName("field")

		if objectNode != nil && fieldNode != nil {
			fieldNameStr := fieldNode.Utf8Text(vr.src)
			container := vr.resolveFieldType(objectNode)

			if container != nil {
				return vr.resolveInElementHierarchy(container, fieldNameStr, false)
			}
		}
		return nil
	default:
		return nil
	}

	return vr.resolveVariable(rawText)
}

// resolveChainedContext 解析链式调用上下文
func (vr *VariableResolver) resolveChainedContext(chainedCtx *ChainedContext) error {
	if len(chainedCtx.Steps) == 0 {
		return fmt.Errorf("empty chain steps")
	}

	var currentType *model.CodeElement

	// 逐步解析链式调用
	for i, step := range chainedCtx.Steps {
		if i == 0 {
			// 第一步：解析基础表达式
			currentType = vr.resolveBaseStep(step)
		} else {
			// 后续步骤：在当前类型上继续解析
			currentType = vr.resolveNextStep(step, currentType)
		}

		if currentType == nil {
			chainedCtx.Error = fmt.Errorf("failed to resolve step %d: %s", i, step.Name)
			return chainedCtx.Error
		}
	}

	chainedCtx.CurrentType = currentType
	chainedCtx.Resolved = true
	return nil
}

// resolveBaseStep 解析基础步骤
func (vr *VariableResolver) resolveBaseStep(step ChainStep) *model.CodeElement {
	if step.IsNew {
		// 处理 new ClassName()
		typeName := strings.TrimPrefix(step.RawText, "new")
		typeName = strings.TrimSpace(typeName)
		if idx := strings.Index(typeName, "("); idx != -1 {
			typeName = typeName[:idx]
		}

		entries := helper.PreciseResolve(vr.gCtx, vr.fCtx, typeName)
		if len(entries) > 0 {
			return entries[0].Element
		}
		return nil
	}

	// 处理普通标识符
	if step.Name == "this" {
		return helper.GetBestElement(vr.fCtx, vr.node, []model.ElementKind{model.Class, model.AnonymousClass})
	}
	if step.Name == "super" {
		container := helper.GetBestElement(vr.fCtx, vr.node, []model.ElementKind{model.Class, model.AnonymousClass})
		if container != nil && container.Extra != nil {
			if superClass, ok := container.Extra.Mores[constants.ClassSuperClass].(string); ok && superClass != "" {
				entries := helper.PreciseResolve(vr.gCtx, vr.fCtx, helper.Clean(superClass))
				if len(entries) > 0 {
					return entries[0].Element
				}
			}
		}
		return nil
	}

	// 普通变量
	return vr.resolveVariable(step.Name)
}

// resolveNextStep 解析后续步骤
func (vr *VariableResolver) resolveNextStep(step ChainStep, currentType *model.CodeElement) *model.CodeElement {
	if currentType == nil {
		return nil
	}

	if step.IsCall || step.IsField {
		// 访问成员：在当前类型中查找
		return vr.resolveInElementHierarchy(currentType, step.Name, false)
	}

	return nil
}

// checkVisibility 检查可见性
func (vr *VariableResolver) checkVisibility(container *model.CodeElement, target *core.DefinitionEntry) bool {
	if target.Element.Kind == model.Variable {
		return true
	}

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

	targetPkg := helper.GetRealPackage(vr.gCtx, target.Element)
	if targetPkg == vr.fCtx.PackageName {
		return true
	}

	if slices.Contains(mods, "protected") {
		sourceClass := helper.GetOwnerClassQN(vr.gCtx, container)
		return helper.IsSubClassOf(vr.gCtx, vr.fCtx, sourceClass, target.ParentQN)
	}

	return false
}

// buildQN 构建限定名
func (vr *VariableResolver) buildQN(parentQN, name string) string {
	if parentQN == "" || parentQN == "." {
		return name
	}
	return parentQN + "." + name
}
