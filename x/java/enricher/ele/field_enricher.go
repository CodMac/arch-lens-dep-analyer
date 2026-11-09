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

// FieldEnricher 为成员字段生成签名与类型/修饰符元数据。
type FieldEnricher struct {
	fCtx core.FileContext
}

// EnrichMetadata 填充字段级 Mores（类型、static/final/常量）与 Signature。
func (c *FieldEnricher) EnrichMetadata(elem *model.CodeElement, node coreast.Node) {
	srcs := c.fCtx.SourceBytes()
	extra := elem.Extra
	mods := elem.Extra.Modifiers

	// 接口常量（constant_declaration，如 interface X { String PLAIN = "plain"; }）
	// 无显式 modifiers 节点，按 JLS §9.3 合成隐式 public static final，
	// 保证 ResolveField 的静态访问校验（isIllegalStatic）正确放行。
	if node.Kind() == "constant_declaration" {
		mods = append(mods, "public", "static", "final")
		elem.Extra.Modifiers = mods
	}

	isFinal := helper.Contains(mods, "final")
	isStatic := helper.Contains(mods, "static")

	vType := c.extractTypeString(node, srcs)
	extra.Mores[constants.FieldRawType] = vType
	extra.Mores[constants.FieldIsStatic] = isStatic
	extra.Mores[constants.FieldIsFinal] = isFinal
	extra.Mores[constants.FieldIsConstant] = isStatic && isFinal

	elem.Signature = strings.TrimSpace(fmt.Sprintf("%s %s %s", strings.Join(mods, " "), vType, elem.Name))
}

// extractTypeString 提取字段类型文本（含泛型/可变参数等形态）。
func (c *FieldEnricher) extractTypeString(node coreast.Node, src *[]byte) string {
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

	return "unknown"
}
