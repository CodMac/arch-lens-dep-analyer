package rel

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

const (
	TmpNode = "tmp_node"
	TmpRaw  = "tmp_raw"
	TmpStmt = "tmp_stmt"
)

type IEnricher interface {
	EnrichMetadata(rel *model.DependencyRelation)
}

type Enricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
	gCtx     *core.GlobalContext

	enricherMap map[model.DependencyType]IEnricher
}

func NewEnricher(resolver core.SymbolResolver, fCtx *core.FileContext, gCtx *core.GlobalContext) *Enricher {
	enricherMap := make(map[model.DependencyType]IEnricher)
	enricherMap[model.Call] = &CallEnricher{resolver: resolver, fCtx: fCtx, gCtx: gCtx}
	enricherMap[model.Create] = &CreateEnricher{resolver: resolver, fCtx: fCtx, gCtx: gCtx}
	enricherMap[model.Assign] = &AssignEnricher{resolver: resolver, fCtx: fCtx, gCtx: gCtx}
	enricherMap[model.Use] = &UseEnricher{resolver: resolver, fCtx: fCtx, gCtx: gCtx}
	enricherMap[model.Cast] = &CastEnricher{resolver: resolver, fCtx: fCtx, gCtx: gCtx}
	enricherMap[model.Throw] = &ThrowEnricher{resolver: resolver, fCtx: fCtx, gCtx: gCtx}
	enricherMap[model.Parameter] = &ParameterEnricher{resolver: resolver, fCtx: fCtx, gCtx: gCtx}
	enricherMap[model.Return] = &ReturnEnricher{resolver: resolver, fCtx: fCtx, gCtx: gCtx}
	enricherMap[model.Annotation] = &AnnotationEnricher{resolver: resolver, fCtx: fCtx, gCtx: gCtx}

	return &Enricher{
		resolver:    resolver,
		fCtx:        fCtx,
		gCtx:        gCtx,
		enricherMap: enricherMap,
	}
}

func (e *Enricher) EnrichCoreMetadata(rel *model.DependencyRelation) {
	if enricher, ok := e.enricherMap[rel.Type]; ok {
		enricher.EnrichMetadata(rel)
	}

	// 清理临时字段
	delete(rel.Mores, TmpNode)
	delete(rel.Mores, TmpRaw)
	delete(rel.Mores, TmpStmt)
}

// =============================================================================
// 包级公开辅助函数
// =============================================================================

func GetRelTmpValue(rel *model.DependencyRelation) (*sitter.Node, string, *sitter.Node) {
	node, _ := rel.Mores[TmpNode].(*sitter.Node)
	rawText, _ := rel.Mores[TmpRaw].(string)
	stmt, _ := rel.Mores[TmpStmt].(*sitter.Node)
	return node, rawText, stmt
}
