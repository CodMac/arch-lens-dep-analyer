// Package rel 提供 Java 依赖关系的元数据增强能力：
// 在 Extractor 的 enrichRelMetadata 阶段，依据关系类型（Call/Use/Assign/...）
// 从 AST 临时节点中提取语言特化属性，填充到 Relation 的 Mores 中。
package rel

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// IEnricher 是单个关系类型的元数据增强契约。
type IEnricher interface {
	EnrichMetadata(rel *model.DependencyRelation)
}

// Enricher 按 DependencyType 分派到具体的关系增强器。
// 内部持有 SymbolResolver 与双上下文，供 receiver 类型解析等场景使用。
type Enricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
	gCtx     *core.GlobalContext

	enricherMap map[model.DependencyType]IEnricher
}

// NewEnricher 创建关系元数据增强器（注册全部动作关系类型）。
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

// EnrichCoreMetadata 按关系类型分派增强，并在结束后统一清理私有临时字段。
func (e *Enricher) EnrichCoreMetadata(rel *model.DependencyRelation) {
	// 惰性初始化 Mores，避免各 enricher 直接写入 nil map 触发 panic。
	rel.GetMores()

	if enricher, ok := e.enricherMap[rel.Type]; ok {
		enricher.EnrichMetadata(rel)
	}

	// 清理临时字段，杜绝泄漏到输出
	rel.ClearTemp()
}

// =============================================================================
// 包级公开辅助函数
// =============================================================================

// GetRelTmpValue 从关系的私有临时字段中读取 (node, rawText, stmt)。
// 兼容各 enricher 从 v1 的 Mores 读取方式迁移后的统一取数入口。
func GetRelTmpValue(rel *model.DependencyRelation) (coreast.Node, string, coreast.Node) {
	node, _ := rel.TmpNode().(coreast.Node)
	rawText, _ := rel.TmpExpr().(string)
	stmt, _ := rel.TmpCtx().(coreast.Node)
	return node, rawText, stmt
}
