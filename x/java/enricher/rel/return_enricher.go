package rel

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

// ReturnEnricher 为返回类型关系（Return）标记基本类型与数组形态。
type ReturnEnricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
	gCtx     *core.GlobalContext
}

// EnrichMetadata 依据原始返回类型文本判定 primitive/array。
func (e *ReturnEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	// 结构关系不设 TmpExpr，raw_text 存于 Mores[RelRawText]（如 "byte[]"、"double[][]"）。
	rawText, _ := rel.Mores[constants.RelRawText].(string)
	isArray := strings.Contains(rawText, "[]")
	// 数组返回类型的 primitive 判定须先剥离数组尾缀（byte[] 的 element type 是 byte）。
	base := strings.ReplaceAll(rawText, "[]", "")
	isPrimitive := helper.IsPrimitiveType(helper.Clean(base))
	rel.Mores[constants.RelReturnIsPrimitive] = isPrimitive
	rel.Mores[constants.RelReturnIsArray] = isArray
}
