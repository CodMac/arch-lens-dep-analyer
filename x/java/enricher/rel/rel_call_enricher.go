package rel

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

type CallEnricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
	gCtx     *core.GlobalContext
}

func (e *CallEnricher) Enrich(rel *model.DependencyRelation) {

}
