package ele

import (
	"fmt"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type AnnotationMemberEnricher struct {
	fCtx *core.FileContext
}

func (e *AnnotationMemberEnricher) EnrichMetadata(elem *model.CodeElement, node *sitter.Node) {
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
