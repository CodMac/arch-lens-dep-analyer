package ele

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type EnumConstantEnricher struct {
	fCtx *core.FileContext
}

func (c *EnumConstantEnricher) EnrichMetadata(elem *model.CodeElement, node *sitter.Node) {
	srcs := c.fCtx.SourceBytes
	extra := elem.Extra

	if argList := helper.FindNamedChildOfType(node, "argument_list"); argList != nil {
		var args []string
		for i := 0; i < int(argList.NamedChildCount()); i++ {
			args = append(args, helper.GetNodeContent(argList.NamedChild(uint(i)), *srcs))
		}
		extra.Mores[constants.EnumArguments] = args
	}

	elem.Signature = elem.Name
}
