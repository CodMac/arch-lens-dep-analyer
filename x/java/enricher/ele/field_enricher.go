package ele

import (
	"fmt"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type FieldEnricher struct {
	fCtx *core.FileContext
}

func (c *FieldEnricher) EnrichMetadata(elem *model.CodeElement, node *sitter.Node) {
	srcs := c.fCtx.SourceBytes
	extra := elem.Extra
	mods := elem.Extra.Modifiers

	isFinal := helper.Contains(mods, "final")
	isStatic := helper.Contains(mods, "static")

	vType := c.extractTypeString(node, srcs)
	extra.Mores[constants.FieldRawType] = vType
	extra.Mores[constants.FieldIsStatic] = isStatic
	extra.Mores[constants.FieldIsFinal] = isFinal
	extra.Mores[constants.FieldIsConstant] = isStatic && isFinal

	elem.Signature = strings.TrimSpace(fmt.Sprintf("%s %s %s", strings.Join(mods, " "), vType, elem.Name))
}

func (c *FieldEnricher) extractTypeString(node *sitter.Node, src *[]byte) string {
	if node.Kind() == "identifier" {
		return "inferred"
	}

	if tNode := node.ChildByFieldName("type"); tNode != nil {
		return helper.GetNodeContent(tNode, *src)
	}

	if node.Kind() == "spread_parameter" {
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(uint(i))
			if strings.Contains(child.Kind(), "type") {
				return helper.GetNodeContent(child, *src) + "..."
			}
		}
	}

	return "unknown"
}
