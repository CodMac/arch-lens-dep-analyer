package ele

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

type MethodRefEnricher struct {
	fCtx *core.FileContext
}

func (c *MethodRefEnricher) EnrichMetadata(elem *model.CodeElement, node *tree_sitter.Node) {
	srcs := c.fCtx.SourceBytes
	extra := elem.Extra

	var receiver, target string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		kind := child.Kind()

		// 1. 忽略不需要的符号和中间件
		if kind == "::" || kind == "type_arguments" {
			if kind == "type_arguments" {
				extra.Mores[constants.MethodRefTypeArgs] = helper.GetNodeContent(child, *srcs)
			}
			continue
		}

		// 2. 识别内容
		content := helper.GetNodeContent(child, *srcs)
		if content == "" {
			continue
		}

		// 逻辑：第一个非符号/非泛型节点是 Receiver，第二个是 Target
		if receiver == "" {
			receiver = content
		} else if target == "" {
			// 如果遇到了 new，说明是构造函数引用
			if kind == "new" {
				target = "new"
			} else {
				target = content
			}
		}
	}

	if receiver != "" {
		extra.Mores[constants.MethodRefReceiver] = receiver
	}
	if target != "" {
		extra.Mores[constants.MethodRefTarget] = target
	}

	// 设置原始签名 (如 System.out::println)
	elem.Signature = helper.GetNodeContent(node, *srcs)
}
