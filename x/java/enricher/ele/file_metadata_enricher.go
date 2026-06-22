package ele

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

type FileMetadataEnricher struct{}

func (e *FileMetadataEnricher) EnrichMetadata(elem *model.CodeElement, fCtx *core.FileContext, extra *model.Extra) {
	extra.Mores[constants.FileRawLOC] = helper.CalculateRawLOC(*fCtx.SourceBytes)
	extra.Mores[constants.FileLOC] = helper.CalculateLOC(*fCtx.SourceBytes)
}
