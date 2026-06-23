package ele

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

type AnonymousClassEnricher struct {
	fCtx *core.FileContext
}

func (c *AnonymousClassEnricher) EnrichMetadata(elem *model.CodeElement, node *tree_sitter.Node) {
	extra := elem.Extra

	// 在 identifyElement 中, AnonymousClass 锚定的 node 是 "object_creation_expression"
	if node.Kind() != "object_creation_expression" {
		return
	}

	// 提取 new 关键字后的类型，例如 new Runnable() { ... } 中的 Runnable
	typeNode := node.ChildByFieldName("type")
	if typeNode != nil {
		typeName := helper.GetNodeContent(typeNode, *c.fCtx.SourceBytes)
		extra.Mores[constants.AnonymousClassType] = typeName
		elem.Signature = "anonymous extends/implements " + typeName
	}
}
