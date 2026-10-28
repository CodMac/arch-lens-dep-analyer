package ele

import (
	"fmt"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

// MethodEnricher 为方法/构造器生成签名、参数、返回类型、throws 与圈复杂度。
type MethodEnricher struct {
	fCtx *core.FileContext
}

// EnrichMetadata 填充方法级 Mores 并构造 Signature。
func (c *MethodEnricher) EnrichMetadata(elem *model.CodeElement, node coreast.Node) {
	srcs := c.fCtx.SourceBytes
	extra := elem.Extra
	mods := elem.Extra.Modifiers

	extra.Mores[constants.MethodIsConstructor] = node.Kind() == "constructor_declaration"
	extra.Mores[constants.MethodIsDefault] = helper.Contains(mods, "default")
	extra.Mores[constants.MethodComplexity] = helper.CalculateComplexity(node)

	typeParams := ""
	if tpNode := node.ChildByFieldName("type_parameters"); tpNode != nil {
		typeParams = helper.GetNodeContent(tpNode, *srcs) + " "
		extra.Mores[constants.MethodTypeParameters] = strings.TrimSpace(typeParams)
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

// extractThrows 提取 throws 声明的异常类型列表。
func (c *MethodEnricher) extractThrows(node coreast.Node, src *[]byte) []string {
	tNode := helper.FindNamedChildOfType(node, "throws")
	if tNode == nil {
		return nil
	}

	var types []string
	for i := 0; i < tNode.NamedChildCount(); i++ {
		child := tNode.NamedChild(i)
		if child.IsNamed() && child.Kind() != "throws" {
			types = append(types, helper.GetNodeContent(child, *src))
		}
	}

	return types
}

// extractParameterList 提取参数定义列表（含类型与名称的原始文本）。
func (c *MethodEnricher) extractParameterList(node coreast.Node, src *[]byte) []string {
	pNode := node.ChildByFieldName("parameters")
	if pNode == nil {
		return nil
	}

	var params []string
	for i := 0; i < pNode.NamedChildCount(); i++ {
		params = append(params, helper.GetNodeContent(pNode.NamedChild(i), *src))
	}

	return params
}

// extractParameterWithNames 提取参数括号原始文本（用于 Signature）。
func (c *MethodEnricher) extractParameterWithNames(node coreast.Node, src *[]byte) string {
	if pNode := node.ChildByFieldName("parameters"); pNode != nil {
		return helper.GetNodeContent(pNode, *src)
	}

	return "()"
}
