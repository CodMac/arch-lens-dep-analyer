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

type ClassEnricher struct {
	fCtx *core.FileContext
}

func (c *ClassEnricher) EnrichMetadata(elem *model.CodeElement, node *sitter.Node) {
	extra := elem.Extra
	mods := elem.Extra.Modifiers

	extra.Mores[constants.ClassIsAbstract] = helper.Contains(mods, "abstract")
	extra.Mores[constants.ClassIsFinal] = helper.Contains(mods, "final")
	extra.Mores[constants.ClassIsStatic] = helper.Contains(mods, "static")

	typeParams := ""
	if tpNode := node.ChildByFieldName("type_parameters"); tpNode != nil {
		typeParams = helper.GetNodeContent(tpNode, *c.fCtx.SourceBytes)
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

func (c *ClassEnricher) findInterfacesNode(node *sitter.Node) *sitter.Node {
	if n := node.ChildByFieldName("interfaces"); n != nil {
		return n
	}
	if n := node.ChildByFieldName("extends"); n != nil {
		return n
	}
	return helper.FindNamedChildOfType(node, "extends_interfaces")
}

func (c *ClassEnricher) extractInterfaceList(node *sitter.Node, srcs *[]byte) []string {
	var results []string
	target := node
	if node.Kind() != "type_list" {
		if listNode := helper.FindNamedChildOfType(node, "type_list"); listNode != nil {
			target = listNode
		}
	}
	for i := 0; i < int(target.NamedChildCount()); i++ {
		child := target.NamedChild(uint(i))
		if strings.Contains(child.Kind(), "type") || child.Kind() == "type_identifier" {
			results = append(results, helper.GetNodeContent(child, *srcs))
		}
	}
	return results
}
