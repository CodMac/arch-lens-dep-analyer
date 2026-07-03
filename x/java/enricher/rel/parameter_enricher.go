package rel

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
)

type ParameterEnricher struct {
	fCtx *core.FileContext
	gCtx *core.GlobalContext
}

func (e *ParameterEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	params, ok := rel.Source.Extra.Mores[constants.MethodParameters].([]string)
	if !ok {
		return
	}

	paramName := rel.Target.Name
	for i, paramTypeAndName := range params {
		if strings.Contains(paramTypeAndName, paramName) {
			rel.Mores[constants.RelParameterIndex] = i

			parts := strings.Fields(paramTypeAndName)
			if len(parts) >= 2 {
				rel.Mores[constants.RelParameterName] = parts[len(parts)-1]
			}

			if strings.Contains(paramTypeAndName, "...") {
				rel.Mores[constants.RelParameterIsVarargs] = true
			}
		}
	}
}
