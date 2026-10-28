package ele

import (
	"fmt"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

// AnnotationMemberEnricher 为注解类型中的元素方法生成默认值与签名。
type AnnotationMemberEnricher struct {
	fCtx *core.FileContext
}

// EnrichMetadata 填充注解元素 Mores（是否注解方法、默认值）与 Signature。
func (e *AnnotationMemberEnricher) EnrichMetadata(elem *model.CodeElement, node coreast.Node) {
	srcs := e.fCtx.SourceBytes

	elem.Extra.Mores[constants.MethodIsAnnotation] = true

	if valNode := node.ChildByFieldName("value"); valNode != nil {
		elem.Extra.Mores[constants.MethodDefaultValue] = helper.GetNodeContent(valNode, *srcs)
	}

	if typeNode := node.ChildByFieldName("type"); typeNode != nil {
		vType := helper.GetNodeContent(typeNode, *srcs)
		elem.Signature = fmt.Sprintf("%s %s()", vType, elem.Name)
	}
}
