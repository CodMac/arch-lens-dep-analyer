package rel

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

type ThrowEnricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
	gCtx     *core.GlobalContext
}

func (e *ThrowEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	node, rawText, _ := GetRelTmpValue(rel)

	if node != nil {
		rel.Mores[constants.RelAstKind] = "throw_statement"
		rel.Target.Name = helper.Clean(rel.Target.Name)
		rel.Target.QualifiedName = helper.Clean(rel.Target.QualifiedName)
		if node.Kind() == "type_identifier" || (node.Parent() != nil && node.Parent().Kind() == "object_creation_expression") {
			rel.Mores[constants.RelThrowIsRuntime] = true
		} else if node.Kind() == "identifier" {
			rel.Mores[constants.RelThrowIsRethrow] = true
		}
		return
	}

	if rawText != "" && rel.Source != nil && rel.Source.Extra != nil {
		if ths, ok := rel.Source.Extra.Mores[constants.MethodThrowsTypes].([]string); ok {
			for i, ex := range ths {
				if helper.Clean(ex) == rel.Target.Name {
					rel.Mores[constants.RelThrowIndex] = i
					rel.Mores[constants.RelThrowIsSignature] = true
					break
				}
			}
		}
	}
}
