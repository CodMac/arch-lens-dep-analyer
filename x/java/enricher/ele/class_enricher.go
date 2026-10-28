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

// ClassEnricher 为类/接口/枚举/注解类型生成签名与继承关系元数据。
type ClassEnricher struct {
	fCtx *core.FileContext
}

// EnrichMetadata 填充类级 Mores（abstract/final/static、父类、实现接口）与 Signature。
func (c *ClassEnricher) EnrichMetadata(elem *model.CodeElement, node coreast.Node) {
	extra := elem.Extra
	mods := elem.Extra.Modifiers

	extra.Mores[constants.ClassIsAbstract] = helper.Contains(mods, "abstract")
	extra.Mores[constants.ClassIsFinal] = helper.Contains(mods, "final")
	extra.Mores[constants.ClassIsStatic] = helper.Contains(mods, "static")

	typeParams := ""
	if tpNode := node.ChildByFieldName("type_parameters"); tpNode != nil {
		typeParams = helper.GetNodeContent(tpNode, *c.fCtx.SourceBytes)
		extra.Mores[constants.ClassTypeParameters] = typeParams
	}

	heritage := ""
	if super := node.ChildByFieldName("superclass"); super != nil {
		content := helper.GetNodeContent(super, *c.fCtx.SourceBytes)
		extra.Mores[constants.ClassSuperClass] = strings.TrimSpace(strings.TrimPrefix(content, "extends"))
		heritage += " " + content
	}

	ifacesNode := c.findInterfacesNode(node)
	if ifacesNode != nil {
		if ifaces := c.extractInterfaceList(ifacesNode, c.fCtx.SourceBytes); len(ifaces) > 0 {
			mKey := constants.InterfaceImplementedInterfaces
			if elem.Kind == model.Class {
				mKey = constants.ClassImplementedInterfaces
			}
			extra.Mores[mKey] = ifaces
			heritage += " " + helper.GetNodeContent(ifacesNode, *c.fCtx.SourceBytes)
		}
	}

	displayKind := strings.Replace(node.Kind(), "_declaration", "", 1)
	elem.Signature = strings.TrimSpace(fmt.Sprintf("%s %s %s%s%s", strings.Join(mods, " "), displayKind, elem.Name, typeParams, heritage))
}

// findInterfacesNode 定位接口/父接口列表节点（兼容 class 与 interface 的不同字段名）。
func (c *ClassEnricher) findInterfacesNode(node coreast.Node) coreast.Node {
	if n := node.ChildByFieldName("interfaces"); n != nil {
		return n
	}
	if n := node.ChildByFieldName("extends"); n != nil {
		return n
	}
	return helper.FindNamedChildOfType(node, "extends_interfaces")
}

// extractInterfaceList 从类型列表节点中提取全部类型名。
func (c *ClassEnricher) extractInterfaceList(node coreast.Node, srcs *[]byte) []string {
	var results []string
	target := node
	if node.Kind() != "type_list" {
		if listNode := helper.FindNamedChildOfType(node, "type_list"); listNode != nil {
			target = listNode
		}
	}
	for i := 0; i < target.NamedChildCount(); i++ {
		child := target.NamedChild(i)
		if strings.Contains(child.Kind(), "type") || child.Kind() == "type_identifier" {
			results = append(results, helper.GetNodeContent(child, *srcs))
		}
	}
	return results
}
