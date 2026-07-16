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

// ResolveChain 核心入口：专注于链式调用中已知 Target 符号的逐层流转。查无此符号则返回 nil，不进行任何就地虚拟保底。
func (cr *ChainResolver) ResolveChain(chain *ExpressionChain) *model.CodeElement {
	if chain == nil {
		return nil
	}

	var consumedSegmentsCount int
	var headElem *model.CodeElement
	var currType *model.CodeElement
	var isStaticContext bool

	// 1. 处理 Head 节点
	if chain.Head.Type == HeadNewExpr {
		headElem, currType, consumedSegmentsCount = cr.resolveNewExprHead(chain)
		isStaticContext = false
	} else {
		headElem, currType, isStaticContext = cr.resolveHeadWithUnwrap(chain.Head)
	}

	remainingSegments := chain.Segments[consumedSegmentsCount:]

	if currType == nil {
		return headElem // 可能是局部变量或直接识别出的方法/类型本身
	}

	if len(remainingSegments) == 0 {
		if headElem != nil {
			return headElem
		}
		return currType
	}

	fromCtx := helper.GetBestElement(cr.fCtx, chain.Head.ASTNode, []model.ElementKind{model.Method, model.Class, model.ScopeBlock})
	var lastResolvedEntity *model.CodeElement = headElem

	// 2. 迭代链条后续段落
	for _, seg := range remainingSegments {
		switch seg.Kind {
		case SegmentClass:
			// 内部类或嵌套静态类
			lastResolvedEntity = cr.resolveClassSegment(seg, currType)
			currType = lastResolvedEntity
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
			// 数组访问：如果是 head 衍生出来的，保留其引用。类型信息在此处若降维则返回 nil，交由外层降维或保留原本类型。
			isStaticContext = false
		}

		if lastResolvedEntity == nil {
			return nil // 中途断链，无法再提供更精准的符号，立即返回 nil 触发外部保底
		}
	}

	return lastResolvedEntity
}

// resolveClassSegment 精准寻找内部类
func (cr *ChainResolver) resolveClassSegment(seg ExpressionSegment, currType *model.CodeElement) *model.CodeElement {
	if currType == nil {
		return nil
	}
	// 嵌套类 A$B
	innerClassQN := currType.QualifiedName + "$" + seg.Name
	if entry, ok := cr.gCtx.FindByQualifiedName(innerClassQN); ok {
		return entry.Element
	}
	// 点号分隔 A.B
	dotInnerClassQN := currType.QualifiedName + "." + seg.Name
	if entry, ok := cr.gCtx.FindByQualifiedName(dotInnerClassQN); ok {
		return entry.Element
	}
	return nil
}

func (cr *ChainResolver) resolveMethodSegment(seg ExpressionSegment, currType *model.CodeElement, isStatic bool, fromCtx *model.CodeElement) *model.CodeElement {
	methodCallNode := helper.GetMethodInvocationNode(seg.ASTNode)
	argTypes := helper.InferMethodArgs(methodCallNode, cr.src)
	return cr.memberResolver.ResolveMethod(currType, seg.Name, argTypes, isStatic, fromCtx)
}

func (cr *ChainResolver) resolveFieldSegment(seg ExpressionSegment, currType *model.CodeElement, isStatic bool, fromCtx *model.CodeElement) *model.CodeElement {
	return cr.memberResolver.ResolveField(currType, seg.Name, isStatic, fromCtx)
}

// resolveNewExprHead 合并可能存在的连续嵌套内部类（如 new A.B.C()）实例化路径并定位构造函数
func (cr *ChainResolver) resolveNewExprHead(chain *ExpressionChain) (*model.CodeElement, *model.CodeElement, int) {
	head := chain.Head
	rawName := cr.cleanGenericType(helper.Clean(head.Name))

	var currentClass *model.CodeElement
	entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, rawName)
	if len(entries) > 0 {
		currentClass = entries[0].Element
	}

	consumed := 0
	if currentClass != nil {
		// 顺着 Segments 向右合并所有的内部类段落
		for i := 0; i < len(chain.Segments); i++ {
			seg := chain.Segments[i]
			if seg.Kind == SegmentClass {
				if nextClass := cr.resolveClassSegment(seg, currentClass); nextClass != nil {
					currentClass = nextClass
					consumed++
				} else {
					break
				}
			} else {
				break
			}
		}
	}

	if currentClass == nil {
		return nil, nil, 0
	}

	// 尝试寻找匹配的显式构造函数
	argTypes := helper.InferMethodArgs(head.ASTNode, cr.src)
	if constructorElem := cr.memberResolver.ResolveMethod(currentClass, currentClass.Name, argTypes, false, nil); constructorElem != nil {
		return constructorElem, currentClass, consumed
	}

	// 若类存在但查无显式构造，为调用处返回当前类作为类型支撑
	return nil, currentClass, consumed
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
		currentClass := helper.GetBestElement(cr.fCtx, head.ASTNode, []model.ElementKind{model.Class, model.AnonymousClass})
		container := helper.GetBestElement(cr.fCtx, head.ASTNode, []model.ElementKind{model.Method, model.Class, model.ScopeBlock})

		if currentClass != nil {
			argTypes := helper.InferMethodArgs(head.ASTNode, cr.src)
			if methodElem := cr.memberResolver.ResolveMethod(currentClass, head.Name, argTypes, false, container); methodElem != nil {
				return methodElem, cr.extractElementByReturnType(methodElem), false
			}
		}
		return nil, nil, false

	case HeadIdent:
		// 1. 是否是类名
		if helper.IsPotentialClassName(helper.Clean(head.Name)) {
			entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, head.Name)
			if len(entries) > 0 {
				k := entries[0].Element.Kind
				isStatic := k == model.Class || k == model.Interface || k == model.Enum
				return entries[0].Element, entries[0].Element, isStatic
			}
		}

		// 2. 是否是局部变量/参数/Lambda 参数
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

		// 3. 是否是当前类的隐式字段
		if currentClass := helper.GetBestElement(cr.fCtx, head.ASTNode, []model.ElementKind{model.Class, model.AnonymousClass}); currentClass != nil {
			if fieldElem := cr.memberResolver.ResolveField(currentClass, head.Name, false, container); fieldElem != nil {
				return fieldElem, cr.extractElementByFieldType(fieldElem), false
			}
		}
	}
	return nil, nil, false
}

// ==================== AST 辅助及类型还原工具 ====================

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
		return nil
	}

	var typeQN string
	if qn, ok := element.Extra.Mores[constants.VariableTypeWithQN].(string); ok && qn != "" {
		typeQN = qn
	} else if raw, ok := element.Extra.Mores[constants.VariableRawType].(string); ok {
		typeQN = raw
	}

	if typeQN == "" || typeQN == "void" {
		return nil
	}

	typeQN = cr.cleanGenericType(typeQN)
	entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, typeQN)
	if len(entries) > 0 {
		return entries[0].Element
	}

	// 仅返回全限定名对应的临时占位（用于下一层链条寻找），不生成真正的 External 节点
	return &model.CodeElement{
		QualifiedName: cr.tryResolveExternalFullQN(typeQN),
		Name:          typeQN,
		Kind:          model.Class,
	}
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

	return &model.CodeElement{
		QualifiedName: cr.tryResolveExternalFullQN(returnQN),
		Name:          returnQN,
		Kind:          model.Class,
	}
}
