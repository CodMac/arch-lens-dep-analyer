package rel

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
)

type CastEnricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
	gCtx     *core.GlobalContext
}

func (e *CastEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	_, ctx := GetRelTmpValue(rel)
	src := *e.fCtx.SourceBytes

	if ctx == nil {
		return
	}
	rel.Mores[constants.RelAstKind] = ctx.Kind()
	rel.Mores[constants.RelRawText] = ctx.Utf8Text(src)
	rel.Mores[constants.RelCastIsInstanceof] = ctx.Kind() == "instanceof_expression"
}
