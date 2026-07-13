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

	// 1. 解析 Head 节点：获取当前起点变量的符号实体（headElem）以及它解包后的类型（currType）
	headElem, currType, isStaticContext := cr.resolveHeadWithUnwrap(chain.Head)
	if currType == nil {
		if headElem != nil {
			return headElem
		}
		return nil
	}

	// 如果没有后续段落（例如纯粹的单代号隐式调用方法或局部变量），直接返回 Head 识别出的目标
	if len(chain.Segments) == 0 {
		if headElem != nil {
			return headElem
		}
		return currType
	}

	// 获取调用发生的上下文环境（从哪个方法或类发起的调用）
	fromCtx := helper.GetBestElement(cr.fCtx, chain.Head.ASTNode, []model.ElementKind{model.Method, model.Class, model.ScopeBlock})
	var lastResolvedEntity *model.CodeElement

	// 2. 依次迭代后续的所有 Segments 段落，确保类型链条对齐不坍塌
	for _, seg := range chain.Segments {
		switch seg.Kind {
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

func (cr *ChainResolver) resolveHeadWithUnwrap(head ExpressionHead) (*model.CodeElement, *model.CodeElement, bool) {
	switch head.Type {
	case HeadNewExpr:
		// 🎯 核心重构：New 表达式对应的生成物是构造函数（Method）
		rawName := cr.cleanGenericType(helper.Clean(head.Name))

		// 1. 获取构造方法应该具备的参数类型
		argTypes := helper.InferMethodArgs(head.ASTNode, *cr.fCtx.SourceBytes)

		// 2. 先尝试精准判定关联的类实体
		entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, rawName)
		if len(entries) > 0 {
			classElem := entries[0].Element
			// 通过 memberResolver 在该类中寻找到对应的构造方法
			if constructorElem := cr.memberResolver.ResolveMethod(classElem, classElem.Name, argTypes, false, nil); constructorElem != nil {
				return constructorElem, classElem, false
			}

			// 如果类存在但没找到匹配的显式构造函数，虚构一个对应的隐式默认构造函数 Method 节点
			fallbackConstructor := &model.CodeElement{
				QualifiedName:  classElem.QualifiedName + "." + classElem.Name + "()",
				Name:           classElem.Name + "()",
				Kind:           model.Method,
				IsFormExternal: classElem.IsFormExternal,
			}
			return fallbackConstructor, classElem, false
		}

		// 3. 兜底判定：如果属于外部依赖类（Imports 映射或保持原样短类名）
		fullQN := cr.tryResolveExternalFullQN(rawName)
		extClass := cr.createExternalClassFallback(fullQN)

		extConstructor := &model.CodeElement{
			QualifiedName:  extClass.QualifiedName + "." + extClass.Name + "()",
			Name:           extClass.Name + "()",
			Kind:           model.Method,
			IsFormExternal: true,
		}
		return extConstructor, extClass, false

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
