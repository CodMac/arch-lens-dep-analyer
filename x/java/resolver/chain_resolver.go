package resolver

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

type ChainResolver struct {
	gCtx           *core.GlobalContext
	fCtx           *core.FileContext
	src            []byte
	memberResolver *MemberResolver
}

func NewChainResolver(gCtx *core.GlobalContext, fCtx *core.FileContext) *ChainResolver {
	return &ChainResolver{
		gCtx:           gCtx,
		fCtx:           fCtx,
		src:            *fCtx.SourceBytes,
		memberResolver: NewMemberResolver(gCtx, fCtx),
	}
}

// ResolveChain 核心入口：专注于链式调用/单调用中 Target 符号的逐层流转与精准对齐
func (cr *ChainResolver) ResolveChain(chain *ExpressionChain) *model.CodeElement {
	if chain == nil {
		return nil
	}

	// 1. 针对方案 A 设计的 HeadNewExpr 特殊预处理：合并可能存在的连续 SegmentClass 作为实例化类终点
	var consumedSegmentsCount int
	var headElem *model.CodeElement
	var currType *model.CodeElement
	var isStaticContext bool

	if chain.Head.Type == HeadNewExpr {
		headElem, currType, consumedSegmentsCount = cr.resolveNewExprHead(chain)
		isStaticContext = false
	} else {
		// 普通 Head 节点流转
		headElem, currType, isStaticContext = cr.resolveHeadWithUnwrap(chain.Head)
	}

	// 截取未被 Head 合并/消耗的后续段落
	remainingSegments := chain.Segments[consumedSegmentsCount:]

	if currType == nil {
		if headElem != nil {
			return headElem
		}
		return nil
	}

	// 如果没有后续剩余段落，直接返回识别出的目标
	if len(remainingSegments) == 0 {
		if headElem != nil {
			return headElem
		}
		return currType
	}

	// 获取调用发生的上下文环境（从哪个方法或类发起的调用）
	fromCtx := helper.GetBestElement(cr.fCtx, chain.Head.ASTNode, []model.ElementKind{model.Method, model.Class, model.ScopeBlock})
	var lastResolvedEntity *model.CodeElement

	// 2. 依次迭代后续的所有 Segments 段落，确保类型链条对齐不坍塌
	for _, seg := range remainingSegments {
		switch seg.Kind {
		case SegmentClass:
			// 🎯 修复场景 7：显式处理中间的内部类节点流转
			lastResolvedEntity = cr.resolveClassSegment(seg, currType, isStaticContext)
			currType = lastResolvedEntity
			// 静态上下文遇到内部类，流转下去依然属于类的静态访问上下文（例如调用其内部静态方法）
			isStaticContext = true

		case SegmentMethod:
			lastResolvedEntity = cr.resolveMethodSegment(seg, currType, isStaticContext, fromCtx)
			currType = cr.extractElementByReturnType(lastResolvedEntity)
			isStaticContext = false

		case SegmentField:
			lastResolvedEntity = cr.resolveFieldSegment(seg, currType, isStaticContext, fromCtx)
			currType = cr.extractElementByFieldType(lastResolvedEntity)
			isStaticContext = false

		case SegmentArray:
			if lastResolvedEntity == nil && headElem != nil {
				lastResolvedEntity = headElem
			}
			isStaticContext = false
		}

		if currType == nil {
			break
		}
	}

	if lastResolvedEntity != nil {
		return lastResolvedEntity
	}
	return currType
}

// ==================== 针对各种 Segment 的定向解析函数 ====================

// resolveClassSegment 解析链条中间的内部类 SegmentClass 节点
func (cr *ChainResolver) resolveClassSegment(seg ExpressionSegment, currType *model.CodeElement, isStatic bool) *model.CodeElement {
	// 在父类作用域/全路径中寻找其定义的内部类
	innerClassQN := currType.QualifiedName + "$" + seg.Name
	if entry, ok := cr.gCtx.FindByQualifiedName(innerClassQN); ok {
		return entry.Element
	}

	// 兼容普通点号分隔命名：Outer.StaticInner
	dotInnerClassQN := currType.QualifiedName + "." + seg.Name
	if entry, ok := cr.gCtx.FindByQualifiedName(dotInnerClassQN); ok {
		return entry.Element
	}

	// 外部依赖保底
	return cr.createExternalClassFallback(currType.QualifiedName + "." + seg.Name)
}

func (cr *ChainResolver) resolveMethodSegment(seg ExpressionSegment, currType *model.CodeElement, isStatic bool, fromCtx *model.CodeElement) *model.CodeElement {
	methodCallNode := helper.GetMethodInvocationNode(seg.ASTNode)
	argTypes := helper.InferMethodArgs(methodCallNode, *cr.fCtx.SourceBytes)

	methodElem := cr.memberResolver.ResolveMethod(currType, seg.Name, argTypes, isStatic, fromCtx)

	// 保底机制：如果项目源码中查无此法（如外部依赖），就地构建合法的外部 METHOD 节点而非降级
	if methodElem == nil {
		fallbackQN := currType.QualifiedName + "." + seg.Name + "()"
		methodElem = &model.CodeElement{
			QualifiedName:  fallbackQN,
			Name:           seg.Name + "()",
			Kind:           model.Method,
			IsFormExternal: true,
		}
	}
	return methodElem
}

func (cr *ChainResolver) resolveFieldSegment(seg ExpressionSegment, currType *model.CodeElement, isStatic bool, fromCtx *model.CodeElement) *model.CodeElement {
	fieldElem := cr.memberResolver.ResolveField(currType, seg.Name, isStatic, fromCtx)

	if fieldElem == nil {
		fallbackQN := currType.QualifiedName + "." + seg.Name
		fieldElem = &model.CodeElement{
			QualifiedName:  fallbackQN,
			Name:           seg.Name,
			Kind:           model.Field,
			IsFormExternal: true,
		}
	}
	return fieldElem
}

// ==================== 基础符号与上下文解包 (Head) ====================

// resolveNewExprHead 针对方案 A 下带有完整内部类路径的 NewExpr 进行就地合并与精确推导
func (cr *ChainResolver) resolveNewExprHead(chain *ExpressionChain) (*model.CodeElement, *model.CodeElement, int) {
	head := chain.Head
	rawName := cr.cleanGenericType(helper.Clean(head.Name))

	// 1. 尝试找到物理起点类（例如 Outer 外部类）
	var currentClass *model.CodeElement
	entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, rawName)
	if len(entries) > 0 {
		currentClass = entries[0].Element
	} else {
		// 起点属于外部类保底
		fullQN := cr.tryResolveExternalFullQN(rawName)
		currentClass = cr.createExternalClassFallback(fullQN)
	}

	// 2. 🎯 核心吞并：顺着 Segments 向右，只要遇到 SegmentClass，就判定为嵌套类实例化的一部分
	consumed := 0
	for i := 0; i < len(chain.Segments); i++ {
		seg := chain.Segments[i]
		if seg.Kind == SegmentClass {
			currentClass = cr.resolveClassSegment(seg, currentClass, true)
			consumed++
		} else {
			// 一旦遇到非 Class 段（例如开始调用方法、访问字段了），立即截断
			break
		}
	}

	// 3. 此时 currentClass 已经锁定为了最内部的那个实例化目标类（例如 StaticInner）
	// 获取构造方法应该具备的参数类型
	argTypes := helper.InferMethodArgs(head.ASTNode, *cr.fCtx.SourceBytes)

	// 4. 尝试在这个锁定的类中寻找到对应的构造方法
	if constructorElem := cr.memberResolver.ResolveMethod(currentClass, currentClass.Name, argTypes, false, nil); constructorElem != nil {
		return constructorElem, currentClass, consumed
	}

	// 无显式构造，虚构对应的隐式默认构造函数 Method 节点
	fallbackConstructor := &model.CodeElement{
		QualifiedName:  currentClass.QualifiedName + "." + currentClass.Name + "()",
		Name:           currentClass.Name + "()",
		Kind:           model.Method,
		IsFormExternal: currentClass.IsFormExternal,
	}
	return fallbackConstructor, currentClass, consumed
}

func (cr *ChainResolver) resolveHeadWithUnwrap(head ExpressionHead) (*model.CodeElement, *model.CodeElement, bool) {
	switch head.Type {
	case HeadThis:
		elem := helper.GetBestElement(cr.fCtx, head.ASTNode, []model.ElementKind{model.Class, model.AnonymousClass})
		return elem, elem, false

	case HeadSuper:
		container := helper.GetBestElement(cr.fCtx, head.ASTNode, []model.ElementKind{model.Class, model.AnonymousClass})
		if container != nil && container.Extra != nil {
			if sc, ok := container.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
				entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, helper.Clean(sc))
				if len(entries) > 0 {
					return entries[0].Element, entries[0].Element, false
				}
			}
		}
		return nil, nil, false

	case HeadLiteral:
		return nil, nil, false

	case HeadImplicitMethod:
		// 🎯 100% 确定是当前类或父类的隐式方法调用
		currentClass := helper.GetBestElement(cr.fCtx, head.ASTNode, []model.ElementKind{model.Class, model.AnonymousClass})
		container := helper.GetBestElement(cr.fCtx, head.ASTNode, []model.ElementKind{model.Method, model.Class, model.ScopeBlock})

		if currentClass != nil {
			// 推导参数类型
			argTypes := helper.InferMethodArgs(head.ASTNode, *cr.fCtx.SourceBytes)
			if methodElem := cr.memberResolver.ResolveMethod(currentClass, head.Name, argTypes, false, container); methodElem != nil {
				return methodElem, cr.extractElementByReturnType(methodElem), false
			}
		}

		// 保底：如果是外部或未识别的隐式方法，不降级为类，直接构建方法虚拟节点
		fallbackQN := ""
		if currentClass != nil {
			fallbackQN = currentClass.QualifiedName + "." + head.Name + "()"
		} else {
			fallbackQN = head.Name + "()"
		}
		return &model.CodeElement{
			QualifiedName:  fallbackQN,
			Name:           head.Name + "()",
			Kind:           model.Method,
			IsFormExternal: true,
		}, nil, false

	case HeadIdent:
		// 1. 类
		if helper.IsPotentialClassName(helper.Clean(head.Name)) {
			entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, head.Name)
			if len(entries) > 0 {
				k := entries[0].Element.Kind
				if k == model.Class || k == model.Interface || k == model.Enum {
					return entries[0].Element, entries[0].Element, true
				}
				return entries[0].Element, entries[0].Element, false
			}
		}

		// 2. 变量/方法参数/lambda表达式查找
		container := helper.GetBestElement(cr.fCtx, head.ASTNode, []model.ElementKind{model.Method, model.Class, model.ScopeBlock, model.Lambda})
		if container != nil {
			if container.Kind == model.Method || container.Kind == model.ScopeBlock {
				localVariableQN := container.QualifiedName + "." + head.Name
				if entry, ok := cr.gCtx.FindByQualifiedName(localVariableQN); ok {
					return entry.Element, cr.extractElementByFieldType(entry.Element), false
				}
			}
			// 作用域链向上爬升
			curr := container.QualifiedName
			for curr != "" {
				targetQN := curr + "." + head.Name
				if entry, ok := cr.gCtx.FindByQualifiedName(targetQN); ok {
					return entry.Element, cr.extractElementByFieldType(entry.Element), false
				}
				if pEntry, ok := cr.gCtx.FindByQualifiedName(curr); ok {
					curr = pEntry.ParentQN
				} else {
					break
				}
			}
		}

		// 3. 隐式字段查找
		if currentClass := helper.GetBestElement(cr.fCtx, head.ASTNode, []model.ElementKind{model.Class, model.AnonymousClass}); currentClass != nil {
			if fieldElem := cr.memberResolver.ResolveField(currentClass, head.Name, false, container); fieldElem != nil {
				return fieldElem, cr.extractElementByFieldType(fieldElem), false
			}
		}

	}
	return nil, nil, false
}

// ==================== AST 辅助及参数推导工具 ====================

func (cr *ChainResolver) cleanGenericType(typeName string) string {
	if idx := strings.Index(typeName, "<"); idx != -1 {
		return strings.TrimSpace(typeName[:idx])
	}
	return typeName
}

func (cr *ChainResolver) tryResolveExternalFullQN(shortName string) string {
	shortName = cr.cleanGenericType(shortName)
	if cr.fCtx != nil && cr.fCtx.Imports != nil {
		if imps, ok := cr.fCtx.Imports[shortName]; ok && len(imps) > 0 {
			return imps[0].RawImportPath
		}
	}
	return shortName
}

func (cr *ChainResolver) extractElementByFieldType(element *model.CodeElement) *model.CodeElement {
	if element == nil || element.Extra == nil {
		return element
	}

	var typeQN string
	if qn, ok := element.Extra.Mores[constants.VariableTypeWithQN].(string); ok && qn != "" {
		typeQN = qn
	} else if raw, ok := element.Extra.Mores[constants.VariableRawType].(string); ok {
		typeQN = raw
	}

	if typeQN == "" || typeQN == "void" {
		return element
	}

	typeQN = cr.cleanGenericType(typeQN)
	entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, typeQN)
	if len(entries) > 0 {
		return entries[0].Element
	}

	return cr.createExternalClassFallback(cr.tryResolveExternalFullQN(typeQN))
}

func (cr *ChainResolver) extractElementByReturnType(methodElement *model.CodeElement) *model.CodeElement {
	if methodElement == nil || methodElement.Extra == nil {
		return nil
	}

	var returnQN string
	if qn, ok := methodElement.Extra.Mores[constants.MethodReturnTypeWithQN].(string); ok && qn != "" {
		returnQN = qn
	} else if raw, ok := methodElement.Extra.Mores[constants.MethodReturnType].(string); ok {
		returnQN = raw
	}

	if returnQN == "" || returnQN == "void" {
		return nil
	}

	returnQN = cr.cleanGenericType(returnQN)
	entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, returnQN)
	if len(entries) > 0 {
		return entries[0].Element
	}

	return cr.createExternalClassFallback(cr.tryResolveExternalFullQN(returnQN))
}

func (cr *ChainResolver) createExternalClassFallback(qualifiedName string) *model.CodeElement {
	qualifiedName = cr.cleanGenericType(qualifiedName)
	shortName := qualifiedName
	for i := len(qualifiedName) - 1; i >= 0; i-- {
		if qualifiedName[i] == '.' {
			shortName = qualifiedName[i+1:]
			break
		}
	}

	return &model.CodeElement{
		QualifiedName:  qualifiedName,
		Name:           shortName,
		Kind:           model.Class,
		IsFormExternal: true,
		Extra: &model.Extra{
			Mores: make(map[string]interface{}),
		},
	}
}
