package resolver

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	sitter "github.com/tree-sitter/go-tree-sitter"
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

// ResolveChain 核心入口：驱动链式调用的逐层类型流转
func (cr *ChainResolver) ResolveChain(chain *ExpressionChain) *model.CodeElement {
	if chain == nil {
		return nil
	}

	// 1. 解析 Head 节点（获取当前变量/引用的符号实体，以及解包后的目标类型）
	headElem, currType, isStaticContext := cr.resolveHeadWithUnwrap(chain.Head)
	if currType == nil {
		if headElem != nil {
			return headElem
		}
		return nil
	}

	// 如果没有后续段落（单一代号场景），直接返回
	if len(chain.Segments) == 0 {
		if headElem != nil {
			return headElem
		}
		return currType
	}

	// 获取调用发生的上下文环境（方法、类或作用域块）
	fromCtx := helper.GetBestElement(cr.fCtx, chain.Head.ASTNode, []model.ElementKind{model.Method, model.Class, model.ScopeBlock})
	var lastResolvedEntity *model.CodeElement

	// 2. 依次迭代后续的所有 Segments 段落
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
			// 数组索引操作（如 arr[0]）是对根节点符号/上一层实体的直接操作。
			// 如果在此之前没有产生过字段或方法偏移（lastResolvedEntity == nil），
			// 说明当前依然在操作 Head 变量本身，将其锁定为上一次的符号实体，防止其降级为基础类型 Class。
			if lastResolvedEntity == nil && headElem != nil {
				lastResolvedEntity = headElem
			}
			isStaticContext = false
		}

		// 如果在中途连当前类型都无法追踪了，直接阻断后续解析
		if currType == nil {
			break
		}
	}

	// 如果最终产生了明确的方法、字段或数组操作实体，返回该目标实体
	if lastResolvedEntity != nil {
		return lastResolvedEntity
	}
	return currType
}

// ==================== 针对各种 Segment 的定向解析函数 ====================

func (cr *ChainResolver) resolveMethodSegment(seg ExpressionSegment, currType *model.CodeElement, isStatic bool, fromCtx *model.CodeElement) *model.CodeElement {
	// 防御性安全适配：获取合法的 method_invocation 节点用以推导参数
	methodCallNode := cr.safeGetMethodInvocationNode(seg.ASTNode)
	argTypes := cr.inferArgs(methodCallNode)

	methodElem := cr.memberResolver.ResolveMethod(currType, seg.Name, argTypes, isStatic, fromCtx)

	// Fallback 保底机制：如果项目源码中查无此法（如 JDK 的 String.trim），构建外部虚拟节点
	if methodElem == nil {
		fallbackQN := currType.QualifiedName + "." + seg.Name + "()"
		methodElem = &model.CodeElement{
			QualifiedName:  fallbackQN,
			Name:           seg.Name + "()",
			Kind:           model.Method,
			IsFormExternal: true,
			Extra: &model.Extra{
				Mores: map[string]interface{}{
					"parent_qn": currType.QualifiedName,
				},
			},
		}
	}
	return methodElem
}

func (cr *ChainResolver) resolveFieldSegment(seg ExpressionSegment, currType *model.CodeElement, isStatic bool, fromCtx *model.CodeElement) *model.CodeElement {
	fieldElem := cr.memberResolver.ResolveField(currType, seg.Name, isStatic, fromCtx)

	// Field 虚拟保底
	if fieldElem == nil {
		fallbackQN := currType.QualifiedName + "." + seg.Name
		fieldElem = &model.CodeElement{
			QualifiedName:  fallbackQN,
			Name:           seg.Name,
			Kind:           model.Field,
			IsFormExternal: true,
			Extra: &model.Extra{
				Mores: map[string]interface{}{
					"parent_qn": currType.QualifiedName,
				},
			},
		}
	}
	return fieldElem
}

// ==================== 基础符号与上下文解包 (Head) ====================

func (cr *ChainResolver) resolveHeadWithUnwrap(head ExpressionHead) (*model.CodeElement, *model.CodeElement, bool) {
	switch head.Type {
	case HeadNewExpr:
		typeName := helper.Clean(head.Name)
		entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, typeName)
		if len(entries) > 0 {
			return entries[0].Element, entries[0].Element, false
		}
		return nil, nil, false

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

	case HeadIdent:
		// 1. 局部变量/方法参数查找
		container := helper.GetBestElement(cr.fCtx, head.ASTNode, []model.ElementKind{model.Method, model.Class, model.ScopeBlock})
		if container != nil {
			if container.Kind == model.Method {
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

		// 2. 当前类及父类中的成员字段/方法隐式调用查找
		if currentClass := helper.GetBestElement(cr.fCtx, head.ASTNode, []model.ElementKind{model.Class, model.AnonymousClass}); currentClass != nil {
			if fieldElem := cr.memberResolver.ResolveField(currentClass, head.Name, false, container); fieldElem != nil {
				return fieldElem, cr.extractElementByFieldType(fieldElem), false
			}
			if methodElem := cr.memberResolver.ResolveMethod(currentClass, head.Name, []string{}, false, container); methodElem != nil {
				return methodElem, cr.extractElementByReturnType(methodElem), false
			}
		}

		// 3. 静态类/接口/枚举类型直接访问查找
		entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, head.Name)
		if len(entries) > 0 {
			k := entries[0].Element.Kind
			if k == model.Class || k == model.Interface || k == model.Enum {
				return entries[0].Element, entries[0].Element, true
			}
			return entries[0].Element, entries[0].Element, false
		}
	}
	return nil, nil, false
}

// ==================== AST 辅助及参数推导工具 ====================

func (cr *ChainResolver) safeGetMethodInvocationNode(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Kind() == "identifier" {
		if parent := node.Parent(); parent != nil && parent.Kind() == "method_invocation" {
			return parent
		}
	}
	return node
}

func (cr *ChainResolver) inferArgs(methodInvocationNode *sitter.Node) []string {
	var types []string
	if methodInvocationNode == nil || methodInvocationNode.Kind() != "method_invocation" {
		return types
	}

	argsNode := methodInvocationNode.ChildByFieldName("arguments")
	if argsNode == nil {
		return types
	}

	count := argsNode.NamedChildCount()
	for i := uint(0); i < count; i++ {
		arg := argsNode.NamedChild(i)
		if arg == nil {
			continue
		}
		switch arg.Kind() {
		case "string_literal":
			types = append(types, "java.lang.String")
		case "decimal_integer_literal", "hex_integer_literal":
			types = append(types, "int")
		case "decimal_floating_point_literal":
			types = append(types, "double")
		case "true", "false":
			types = append(types, "boolean")
		case "null_literal":
			types = append(types, "null")
		case "object_creation_expression", "cast_expression":
			if typeNode := arg.ChildByFieldName("type"); typeNode != nil {
				types = append(types, helper.GetNodeContent(typeNode, cr.src))
			} else {
				types = append(types, "unknown")
			}
		default:
			types = append(types, "unknown")
		}
	}
	return types
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

	if typeQN != "" {
		entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, typeQN)
		if len(entries) > 0 {
			return entries[0].Element
		}

		// 🎯 核心兜底：如果是外部依赖类/JDK 类（源码中查无此项），就地构建虚拟 Class 节点保证链条延续
		return cr.createExternalClassFallback(typeQN)
	}
	return element
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

	if returnQN != "" {
		if returnQN == "void" {
			return nil
		}

		entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, returnQN)
		if len(entries) > 0 {
			return entries[0].Element
		}

		// 🎯 核心兜底：针对外部返回类型（如 String 等）做虚拟 Class 节点保底
		return cr.createExternalClassFallback(returnQN)
	}
	return nil
}

// 辅助函数：统一收拢外部依赖/JDK 类的虚拟 Class 节点构造逻辑
func (cr *ChainResolver) createExternalClassFallback(qualifiedName string) *model.CodeElement {
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
