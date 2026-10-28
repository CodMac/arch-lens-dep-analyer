package rel

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
)

// ParameterEnricher 为参数类型关系（Parameter）匹配签名参数索引与名称。
type ParameterEnricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
	gCtx     *core.GlobalContext
}

// EnrichMetadata 在 Source 方法的参数列表中定位目标参数的位置与形态。
func (e *ParameterEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	// 结构关系不设 TmpExpr，raw_text 存于 Mores[RelRawText]，
	// 且与 MethodParameters 中的元素逐字相等（见 extractStructural），故用精确匹配定位参数。
	rawText, _ := rel.Mores[constants.RelRawText].(string)

	if params, ok := rel.Source.Extra.Mores[constants.MethodParameters].([]string); ok {
		for i, p := range params {
			if rawText != "" {
				if p != rawText {
					continue
				}
			} else if !strings.Contains(p, rel.Target.Name) {
				continue
			}
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
