// Package ele 提供 Java 代码元素的元数据增强能力：
// 在 Collector 的 enrichMetadata 阶段，为每个定义填充签名、修饰符、注解、注释与语言特化指标。
package ele

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

// IEnricher 是单个元素类型的元数据增强契约。
type IEnricher interface {
	EnrichMetadata(elem *model.CodeElement, node coreast.Node)
}

// Enricher 按 ElementKind 分派到具体的元素增强器。
type Enricher struct {
	fCtx        *core.FileContext
	enricherMap map[model.ElementKind]IEnricher
}

// NewEnricher 创建元素元数据增强器（注册全部 Java 元素类型）。
func NewEnricher(fCtx *core.FileContext) *Enricher {
	enricherMap := make(map[model.ElementKind]IEnricher)
	enricherMap[model.File] = &FileEnricher{fCtx: fCtx}
	enricherMap[model.Class] = &ClassEnricher{fCtx: fCtx}
	enricherMap[model.Method] = &MethodEnricher{fCtx: fCtx}
	enricherMap[model.Method+model.KAnnotation] = &AnnotationMemberEnricher{fCtx: fCtx}
	enricherMap[model.Field] = &FieldEnricher{fCtx: fCtx}
	enricherMap[model.Variable] = &VariableEnricher{fCtx: fCtx}
	enricherMap[model.EnumConstant] = &EnumConstantEnricher{fCtx: fCtx}
	enricherMap[model.Lambda] = &LambdaEnricher{fCtx: fCtx}
	enricherMap[model.MethodRef] = &MethodRefEnricher{fCtx: fCtx}
	enricherMap[model.ScopeBlock] = &ScopeBlockEnricher{fCtx: fCtx}
	enricherMap[model.AnonymousClass] = &AnonymousClassEnricher{fCtx: fCtx}

	return &Enricher{
		fCtx:        fCtx,
		enricherMap: enricherMap,
	}
}

// EnrichMetadata 为一个定义填充通用元数据并按类型分派。
func (e *Enricher) EnrichMetadata(entry *core.DefinitionEntry) {
	srcs := e.fCtx.SourceBytes
	elem := entry.Element
	node := entry.Node

	elem.Doc, elem.Comment = e.extractComments(node, srcs)
	elem.Extra = &model.Extra{Mores: make(map[string]interface{})}
	elem.Extra.Modifiers, elem.Extra.Annotations = e.extractModifiersAndAnnotations(node, srcs)

	switch elem.Kind {
	case model.File:
		e.enricherMap[model.File].EnrichMetadata(elem, node)

	case model.Class, model.Interface, model.KAnnotation:
		e.enricherMap[model.Class].EnrichMetadata(elem, node)

	case model.Method:
		if node.Kind() == "annotation_type_element_declaration" {
			e.enricherMap[model.Method+model.KAnnotation].EnrichMetadata(elem, node)
		} else {
			e.enricherMap[model.Method].EnrichMetadata(elem, node)
		}

	case model.Field:
		e.enricherMap[model.Field].EnrichMetadata(elem, node)

	case model.Variable:
		e.enricherMap[model.Variable].EnrichMetadata(elem, node)

	case model.EnumConstant:
		e.enricherMap[model.EnumConstant].EnrichMetadata(elem, node)

	case model.Lambda:
		e.enricherMap[model.Lambda].EnrichMetadata(elem, node)

	case model.MethodRef:
		e.enricherMap[model.MethodRef].EnrichMetadata(elem, node)

	case model.ScopeBlock:
		e.enricherMap[model.ScopeBlock].EnrichMetadata(elem, node)

	case model.AnonymousClass:
		e.enricherMap[model.AnonymousClass].EnrichMetadata(elem, node)
	}
}

// extractModifiersAndAnnotations 从 modifiers 节点中拆分修饰符与注解。
func (e *Enricher) extractModifiersAndAnnotations(n coreast.Node, src *[]byte) ([]string, []string) {
	var mods = make([]string, 0)
	var annos = make([]string, 0)

	if mNode := helper.FindNamedChildOfType(n, "modifiers"); mNode != nil {
		for i := 0; i < mNode.ChildCount(); i++ {
			child := mNode.Child(i)
			txt := helper.GetNodeContent(child, *src)
			if strings.Contains(child.Kind(), "annotation") {
				annos = append(annos, txt)
			} else if txt != "" {
				mods = append(mods, txt)
			}
		}
	}

	return mods, annos
}

// extractComments 提取前导注释：Javadoc 记入 Doc，普通注释记入 Comment。
func (e *Enricher) extractComments(node coreast.Node, src *[]byte) (doc, comment string) {
	curr := node
	if node.Kind() == "variable_declarator" && node.Parent() != nil {
		curr = node.Parent()
	}
	prev := curr.PrevSibling()
	for prev != nil {
		if prev.Kind() == "block_comment" || prev.Kind() == "line_comment" {
			text := helper.GetNodeContent(prev, *src)
			if strings.HasPrefix(text, "/**") {
				doc = text
			} else {
				comment = text
			}
			break
		}
		if strings.TrimSpace(helper.GetNodeContent(prev, *src)) != "" {
			break
		}
		prev = prev.PrevSibling()
	}
	return
}
