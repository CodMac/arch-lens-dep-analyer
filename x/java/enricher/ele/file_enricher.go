package ele

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

// FileEnricher 为文件元素生成行数指标。
type FileEnricher struct {
	fCtx *core.FileContext
}

// EnrichMetadata 填充文件级 Mores（LOC / RawLOC）。
func (e *FileEnricher) EnrichMetadata(elem *model.CodeElement, node coreast.Node) {
	srcs := e.fCtx.SourceBytes

	elem.Extra.Mores[constants.FileRawLOC] = helper.CalculateRawLOC(*srcs)
	elem.Extra.Mores[constants.FileLOC] = helper.CalculateLOC(*srcs)
}
