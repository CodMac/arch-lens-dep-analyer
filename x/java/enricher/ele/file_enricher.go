package ele

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

type FileEnricher struct {
	fCtx *core.FileContext
}

func (e *FileEnricher) EnrichMetadata(elem *model.CodeElement, node *tree_sitter.Node) {
	srcs := e.fCtx.SourceBytes

	elem.Extra.Mores[constants.FileRawLOC] = helper.CalculateRawLOC(*srcs)
	elem.Extra.Mores[constants.FileLOC] = helper.CalculateLOC(*srcs)
}
