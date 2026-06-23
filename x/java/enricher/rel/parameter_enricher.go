package rel

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"strings"
)

type ParameterEnricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
	gCtx     *core.GlobalContext
}

func (e *ParameterEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	_, rawText, _ := GetRelTmpValue(rel)

	if params, ok := rel.Source.Extra.Mores[constants.MethodParameters].([]string); ok {
		for i, p := range params {
			if strings.Contains(p, rel.Target.Name) || strings.Contains(p, rawText) {
				rel.Mores[constants.RelParameterIndex] = i
				parts := strings.Fields(p)
				if len(parts) >= 2 {
					rel.Mores[constants.RelParameterName] = parts[len(parts)-1]
				}
				if strings.Contains(p, "...") {
					rel.Mores[constants.RelParameterIsVarargs] = true
				}
			}
		}
	}
}
