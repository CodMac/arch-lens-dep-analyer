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

// VariableEnricher 为局部变量/参数生成签名与类型元数据。
type VariableEnricher struct {
	fCtx *core.FileContext
}

// EnrichMetadata 填充变量级 Mores（类型、final、是否参数）与 Signature。
func (c *VariableEnricher) EnrichMetadata(elem *model.CodeElement, node coreast.Node) {
	srcs := c.fCtx.SourceBytes
	extra := elem.Extra
	mods := elem.Extra.Modifiers

	isFinal := helper.Contains(mods, "final")
	vType := c.extractTypeString(node, srcs)
	extra.Mores[constants.VariableRawType] = vType
	extra.Mores[constants.VariableIsFinal] = isFinal
	extra.Mores[constants.VariableIsParam] = node.Kind() == "formal_parameter" || node.Kind() == "spread_parameter"

	elem.Signature = strings.TrimSpace(fmt.Sprintf("%s %s %s", strings.Join(mods, " "), vType, elem.Name))
}

// extractTypeString 提取变量类型文本（含泛型/可变参数等形态）。
func (c *VariableEnricher) extractTypeString(node coreast.Node, src *[]byte) string {
	if node.Kind() == "identifier" {
		return "inferred"
	}

	if tNode := node.ChildByFieldName("type"); tNode != nil {
		return helper.GetNodeContent(tNode, *src)
	}

	if node.Kind() == "spread_parameter" {
		for i := 0; i < node.NamedChildCount(); i++ {
			child := node.NamedChild(i)
			if strings.Contains(child.Kind(), "type") {
				return helper.GetNodeContent(child, *src) + "..."
			}
		}
	}

	// catch_formal_parameter 无 type 字段，类型在 catch_type 位置子节点
	// （tree-sitter-java grammar: seq(modifiers?, catch_type, variable_declarator_id)）。
	// 多异常联合类型（catch (IOException | SQLException e)）取第一个类型作为代表：
	// 联合类型无法作为单一符号解析，取首项保证下游（throw 解析/receiverContainer）得到有效类型。
	if node.Kind() == "catch_formal_parameter" {
		if ct := helper.FindNamedChildOfType(node, "catch_type"); ct != nil {
			raw := helper.GetNodeContent(ct, *src)
			if i := strings.Index(raw, "|"); i >= 0 {
				return strings.TrimSpace(raw[:i])
			}
			return raw
		}
	}

	return "unknown"
}
