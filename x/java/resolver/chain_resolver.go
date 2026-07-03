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

// ResolveChain 接收精准解构对齐后的 ExpressionChain 指针
func (cr *ChainResolver) ResolveChain(chain *ExpressionChain) *model.CodeElement {
	if chain == nil {
		return nil
	}

	// 1. 消费 Head 节点，推导基础上下文状态
	currType, isStaticContext := cr.resolveHead(chain.Head)
	if currType == nil {
		return nil
	}

	fromCtx := helper.GetBestElement(cr.fCtx, chain.Head.ASTNode, []model.ElementKind{model.Method, model.Class, model.ScopeBlock})

	// 2. 向前串联驱动所有的 Segments 后续层级
	for _, seg := range chain.Segments {
		if seg.Kind == SegmentMethod {
			// 解析当前层的实参节点推导
			argTypes := cr.inferArgs(seg.ASTNode)
			methodElem := cr.memberResolver.ResolveMethod(currType, seg.Name, argTypes, isStaticContext, fromCtx)
			if methodElem == nil {
				return nil
			}
			currType = cr.extractElementByReturnType(methodElem)
			isStaticContext = false
		} else if seg.Kind == SegmentField {
			fieldElem := cr.memberResolver.ResolveField(currType, seg.Name, isStaticContext, fromCtx)
			if fieldElem == nil {
				return nil
			}
			currType = cr.extractElementByFieldType(fieldElem)
			isStaticContext = false
		} else if seg.Kind == SegmentArray {
			// 数组访问保持当前元素底层类型或降级处理
			isStaticContext = false
		}

		if currType == nil {
			return nil
		}
	}

	return currType
}

func (cr *ChainResolver) resolveHead(head ExpressionHead) (*model.CodeElement, bool) {
	switch head.Type {
	case HeadNewExpr:
		typeName := helper.Clean(head.Name)
		entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, typeName)
		if len(entries) > 0 {
			return entries[0].Element, false
		}
		return nil, false

	case HeadThis:
		return helper.GetBestElement(cr.fCtx, head.ASTNode, []model.ElementKind{model.Class, model.AnonymousClass}), false

	case HeadSuper:
		container := helper.GetBestElement(cr.fCtx, head.ASTNode, []model.ElementKind{model.Class, model.AnonymousClass})
		if container != nil && container.Extra != nil {
			if sc, ok := container.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
				entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, helper.Clean(sc))
				if len(entries) > 0 {
					return entries[0].Element, false
				}
			}
		}
		return nil, false

	case HeadLiteral:
		// 字面量不具备向后点号引用的实际结构体或返回空
		return nil, false

	case HeadIdent:
		// 本地变量或形参作用域浮动检索
		container := helper.GetBestElement(cr.fCtx, head.ASTNode, []model.ElementKind{model.Method, model.Class, model.ScopeBlock})
		if container != nil {
			curr := container.QualifiedName
			for curr != "" {
				targetQN := curr + "." + head.Name
				if entry, ok := cr.gCtx.FindByQualifiedName(targetQN); ok {
					return cr.extractElementByFieldType(entry.Element), false
				}
				if pEntry, ok := cr.gCtx.FindByQualifiedName(curr); ok {
					curr = pEntry.ParentQN
				} else {
					break
				}
			}
		}

		// 静态类上下文访问查找
		entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, head.Name)
		if len(entries) > 0 {
			k := entries[0].Element.Kind
			if k == model.Class || k == model.Interface || k == model.Enum {
				return entries[0].Element, true
			}
			return entries[0].Element, false
		}
	}

	return nil, false
}

func (cr *ChainResolver) inferArgs(methodInvocationNode *sitter.Node) []string {
	var types []string
	if methodInvocationNode == nil {
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
		kind := arg.Kind()
		switch kind {
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
		entries := helper.PreciseResolve(cr.gCtx, cr.fCtx, returnQN)
		if len(entries) > 0 {
			return entries[0].Element
		}
	}
	return nil
}
