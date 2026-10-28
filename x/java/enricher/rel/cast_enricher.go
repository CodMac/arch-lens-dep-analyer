package rel

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
)

// CastEnricher 为类型转换关系（Cast）提取类型转换形态元数据。
type CastEnricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
	gCtx     *core.GlobalContext
}

// EnrichMetadata 填充 Cast 关系的 AST kind、原始文本与 instanceof 标记。
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
