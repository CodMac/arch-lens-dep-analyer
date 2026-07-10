package rel

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type CastEnricher struct {
	fCtx *core.FileContext
	gCtx *core.GlobalContext
}

func (e *CastEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	node, _ := rel.Mores[constants.TmpNode].(*sitter.Node)
	ctxNode, _ := rel.Mores[constants.TmpCtxNode].(*sitter.Node)
	if node == nil || ctxNode == nil {
		return
	}

	rel.Mores[constants.RelCastIsInstanceof] = ctxNode.Kind() == "instanceof_expression"
}
