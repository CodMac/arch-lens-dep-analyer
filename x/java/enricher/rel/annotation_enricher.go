package rel

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
)

// AnnotationEnricher 为注解关系（Annotation）映射注解目标并修正 Target 名。
type AnnotationEnricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
	gCtx     *core.GlobalContext
}

// EnrichMetadata 将 Source 元素类型映射为注解目标，并去除 Target 的括号后缀。
func (e *AnnotationEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	target := e._mapElementKindToAnnotationTarget(rel.Source)
	rel.Mores[constants.RelAnnotationTarget] = target

	// 修正 Target
	rel.Target.Name = strings.Split(rel.Target.Name, "(")[0]
	rel.Target.QualifiedName = strings.Split(rel.Target.QualifiedName, "(")[0]
}

// _mapElementKindToAnnotationTarget 将元素类型映射为注解的 Java Target 语义。
func (e *AnnotationEnricher) _mapElementKindToAnnotationTarget(elem *model.CodeElement) string {
	switch elem.Kind {
	case model.Class, model.Interface, model.Enum:
		return "TYPE"
	case model.Field:
		return "FIELD"
	case model.Method:
		return "METHOD"
	case model.Variable:
		if isParam, _ := elem.Extra.Mores[constants.VariableIsParam].(bool); isParam {
			return "PARAMETER"
		}
		return "LOCAL_VARIABLE"
	}
	return "UNKNOWN"
}
