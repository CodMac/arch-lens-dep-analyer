package ele

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

type ScopeBlockEnricher struct {
	fCtx *core.FileContext
}

func (c *ScopeBlockEnricher) EnrichMetadata(elem *model.CodeElement, node *tree_sitter.Node) {
	extra := elem.Extra

	isStatic := node.Kind() == "static_initializer"
	extra.Mores[constants.BlockIsStatic] = isStatic
	extra.Mores[constants.MethodComplexity] = helper.CalculateComplexity(node)

	elem.Signature = "{...}"
	if isStatic {
		elem.Signature = "static {...}"
	}
}
