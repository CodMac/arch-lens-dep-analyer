package rel

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type IEnricher interface {
	EnrichMetadata(rel *model.DependencyRelation)
}

type Enricher struct {
	fCtx *core.FileContext
	gCtx *core.GlobalContext

	enricherMap map[model.DependencyType]IEnricher
}

func NewEnricher(fCtx *core.FileContext, gCtx *core.GlobalContext) *Enricher {
	enricherMap := make(map[model.DependencyType]IEnricher)
	enricherMap[model.Call] = &CallEnricher{fCtx: fCtx, gCtx: gCtx}
	enricherMap[model.Create] = &CreateEnricher{fCtx: fCtx, gCtx: gCtx}
	enricherMap[model.Assign] = &AssignEnricher{fCtx: fCtx, gCtx: gCtx}
	enricherMap[model.Use] = &UseEnricher{fCtx: fCtx, gCtx: gCtx}
	enricherMap[model.Cast] = &CastEnricher{fCtx: fCtx, gCtx: gCtx}
	enricherMap[model.Throw] = &ThrowEnricher{fCtx: fCtx, gCtx: gCtx}
	enricherMap[model.Parameter] = &ParameterEnricher{fCtx: fCtx, gCtx: gCtx}
	enricherMap[model.Return] = &ReturnEnricher{fCtx: fCtx, gCtx: gCtx}
	enricherMap[model.Annotation] = &AnnotationEnricher{fCtx: fCtx, gCtx: gCtx}

	return &Enricher{
		fCtx:        fCtx,
		gCtx:        gCtx,
		enricherMap: enricherMap,
	}
}

func (e *Enricher) EnrichCoreMetadata(rel *model.DependencyRelation) {
	if enricher, ok := e.enricherMap[rel.Type]; ok {
		enricher.EnrichMetadata(rel)
	}
}

// =============================================================================
// 包级公开辅助函数
// =============================================================================

func GetRelTmpValue(rel *model.DependencyRelation) (*sitter.Node, *sitter.Node) {
	node, _ := rel.Mores[constants.TmpNode].(*sitter.Node)
	ctxNode, _ := rel.Mores[constants.TmpExpressNode].(*sitter.Node)
	return node, ctxNode
}
