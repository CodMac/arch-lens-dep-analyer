package rel

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

type ReturnEnricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
	gCtx     *core.GlobalContext
}

func (e *ReturnEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	rawText := rel.Mores[constants.RelRawText].(string)

	//_, rawText, _ := GetRelTmpValue(rel)
	isPrimitive := e.resolver.IsPrimitive(helper.Clean(rawText))
	rel.Mores[constants.RelReturnIsPrimitive] = isPrimitive
	rel.Mores[constants.RelReturnIsArray] = strings.Contains(rawText, "[]")
}
