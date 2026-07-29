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

	var headElem *model.CodeElement
	var currType *model.CodeElement
	var isStaticContext bool

	// 1. 处理 Head 节点
	if chain.Head.Type == HeadNewExpr {
		headElem, currType = cr.resolveNewExprHead(chain)
		isStaticContext = false
	} else if chain.Head.Type == HeadCastExpr {
		headElem, currType = cr.resolveCastExprHead(chain)
		isStaticContext = false
	} else {
		headElem, currType, isStaticContext = cr.resolveHeadWithUnwrap(chain.Head)
	}

	if len(chain.Segments) == 0 {
		return headElem
	}

	fromCtx := helper.GetBestElement(cr.fCtx, chain.Head.ASTNode, []model.ElementKind{model.Method, model.Class, model.ScopeBlock})
	var lastResolvedEntity = headElem

	// 2. 迭代链条后续段落
	for _, seg := range chain.Segments {
		switch seg.Kind {
		case SegmentClass:
			// 内部类或嵌套静态类
			lastResolvedEntity = cr.resolveClassSegment(seg, currType)
			currType = lastResolvedEntity
			isStaticContext = true

		case SegmentMethod:
			lastResolvedEntity = cr.resolveMethodSegment(seg, currType, isStaticContext, fromCtx)
			// 适配 helper 的双返回值，ChainResolver 只流转已知符号 currType
			_, currType = helper.ExtractElementByReturnType(cr.gCtx, cr.fCtx, lastResolvedEntity)
			isStaticContext = false

		case SegmentField:
			lastResolvedEntity = cr.resolveFieldSegment(seg, currType, isStaticContext, fromCtx)
			// 适配 helper 的双返回值，ChainResolver 只流转已知符号 currType
			_, currType = helper.ExtractElementByFieldType(cr.gCtx, cr.fCtx, lastResolvedEntity)
			isStaticContext = false

		case SegmentArray:
			isStaticContext = false
		}

		if lastResolvedEntity == nil {
			return nil
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
func (cr *ChainResolver) resolveNewExprHead(chain *ExpressionChain) (*model.CodeElement, *model.CodeElement) {
	head := chain.Head
	rawName := cr.cleanGenericType(helper.Clean(head.Name))

	var currentClass *model.CodeElement
	entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, rawName)
	if len(entries) > 0 {
		currentClass = entries[0].Element
	}

	if currentClass == nil {
		return nil, nil
	}

	// 尝试寻找匹配的显式构造函数
	argTypes := helper.InferMethodArgs(head.ASTNode, cr.src)
	if constructorElem := cr.memberResolver.ResolveMethod(currentClass, currentClass.Name, argTypes, false, currentClass); constructorElem != nil {
		return constructorElem, currentClass
	}

	return nil, nil
}

// resolveCastExprHead 处理强制类型转换起点
func (cr *ChainResolver) resolveCastExprHead(chain *ExpressionChain) (*model.CodeElement, *model.CodeElement) {
	head := chain.Head
	castType := cr.cleanGenericType(helper.Clean(head.CastType))
	if castType == "" {
		castType = cr.cleanGenericType(helper.Clean(head.Name))
	}

	// 仅在项目已知的符号库中精准查找
	entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, castType)
	if len(entries) > 0 {
		return entries[0].Element, entries[0].Element
	}

	// 查不到说明是外部类（如 String / java.util.List），直接返回 nil！
	return nil, nil
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
				_, retType := helper.ExtractElementByReturnType(cr.gCtx, cr.fCtx, methodElem)
				return methodElem, retType, false
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
					_, fieldType := helper.ExtractElementByFieldType(cr.gCtx, cr.fCtx, entry.Element)
					return entry.Element, fieldType, false
				}
			}
			// 作用域链向上爬升
			curr := container.QualifiedName
			for curr != "" {
				targetQN := curr + "." + head.Name
				if entry, ok := cr.gCtx.FindByQualifiedName(targetQN); ok {
					_, fieldType := helper.ExtractElementByFieldType(cr.gCtx, cr.fCtx, entry.Element)
					return entry.Element, fieldType, false
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
				_, fieldType := helper.ExtractElementByFieldType(cr.gCtx, cr.fCtx, fieldElem)
				return fieldElem, fieldType, false
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
