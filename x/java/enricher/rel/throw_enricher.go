package rel

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

// ThrowEnricher 为异常抛出关系（Throw）识别运行时异常/重抛/签名声明。
type ThrowEnricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
	gCtx     *core.GlobalContext
}

// EnrichMetadata 依据触发节点形态区分 throw 语句与 throws 签名声明。
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

	// 结构关系（throws 签名）不设 TmpExpr，raw_text 存于 Mores[RelRawText]。
	rawText, _ = rel.Mores[constants.RelRawText].(string)
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
