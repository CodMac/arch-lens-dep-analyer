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
	_, _, stmt := GetRelTmpValue(rel)
	src := *e.fCtx.SourceBytes

	if stmt == nil {
		return
	}
	rel.Mores[constants.RelAstKind] = stmt.Kind()
	rel.Mores[constants.RelRawText] = stmt.Utf8Text(src)
	rel.Mores[constants.RelCastIsInstanceof] = stmt.Kind() == "instanceof_expression"
}
