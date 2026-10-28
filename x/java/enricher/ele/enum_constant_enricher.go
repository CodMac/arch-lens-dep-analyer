package ele

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

// EnumConstantEnricher 为枚举常量生成构造参数元数据。
type EnumConstantEnricher struct {
	fCtx *core.FileContext
}

// EnrichMetadata 填充枚举常量 Mores（构造参数列表）与 Signature。
func (c *EnumConstantEnricher) EnrichMetadata(elem *model.CodeElement, node coreast.Node) {
	srcs := c.fCtx.SourceBytes
	extra := elem.Extra

	if argList := helper.FindNamedChildOfType(node, "argument_list"); argList != nil {
		var args []string
		for i := 0; i < argList.NamedChildCount(); i++ {
			args = append(args, helper.GetNodeContent(argList.NamedChild(i), *srcs))
		}
		extra.Mores[constants.EnumArguments] = args
	}

	elem.Signature = elem.Name
}
