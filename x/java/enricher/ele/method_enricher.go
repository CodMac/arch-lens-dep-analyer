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

type MethodEnricher struct {
	fCtx *core.FileContext
}

func (c *MethodEnricher) EnrichMetadata(elem *model.CodeElement, node *sitter.Node) {
	srcs := c.fCtx.SourceBytes
	extra := elem.Extra
	mods := elem.Extra.Modifiers

	extra.Mores[constants.MethodIsConstructor] = node.Kind() == "constructor_declaration"
	extra.Mores[constants.MethodComplexity] = helper.CalculateComplexity(node)

	typeParams := ""
	if tpNode := node.ChildByFieldName("type_parameters"); tpNode != nil {
		typeParams = helper.GetNodeContent(tpNode, *srcs) + " "
	}

	retType := ""
	if tNode := node.ChildByFieldName("type"); tNode != nil {
		retType = helper.GetNodeContent(tNode, *srcs)
		extra.Mores[constants.MethodReturnType] = retType
	}

	paramsRaw := c.extractParameterWithNames(node, srcs)
	if params := c.extractParameterList(node, srcs); len(params) > 0 {
		extra.Mores[constants.MethodParameters] = params
	}

	throwsList := c.extractThrows(node, srcs)
	throwsStr := ""
	if len(throwsList) > 0 {
		extra.Mores[constants.MethodThrowsTypes] = throwsList
		throwsStr = " throws " + strings.Join(throwsList, ", ")
	}

	elem.Signature = strings.TrimSpace(fmt.Sprintf("%s %s%s %s%s%s", strings.Join(mods, " "), typeParams, retType, elem.Name, paramsRaw, throwsStr))
}

func (c *MethodEnricher) extractThrows(node *sitter.Node, src *[]byte) []string {
	tNode := helper.FindNamedChildOfType(node, "throws")
	if tNode == nil {
		return nil
	}

	var types []string
	for i := 0; i < int(tNode.NamedChildCount()); i++ {
		child := tNode.NamedChild(uint(i))
		if child.IsNamed() && child.Kind() != "throws" {
			types = append(types, helper.GetNodeContent(child, *src))
		}
	}

	return types
}

func (c *MethodEnricher) extractParameterList(node *sitter.Node, src *[]byte) []string {
	pNode := node.ChildByFieldName("parameters")
	if pNode == nil {
		return nil
	}

	var params []string
	for i := 0; i < int(pNode.NamedChildCount()); i++ {
		params = append(params, helper.GetNodeContent(pNode.NamedChild(uint(i)), *src))
	}

	return params
}

func (c *MethodEnricher) extractParameterWithNames(node *sitter.Node, src *[]byte) string {
	if pNode := node.ChildByFieldName("parameters"); pNode != nil {
		return helper.GetNodeContent(pNode, *src)
	}

	return "()"
}
