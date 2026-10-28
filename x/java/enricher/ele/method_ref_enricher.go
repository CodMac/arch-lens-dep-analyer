package ele

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

// MethodRefEnricher 为方法引用 (如 System.out::println) 生成接收者/目标元数据。
type MethodRefEnricher struct {
	fCtx *core.FileContext
}

// EnrichMetadata 填充方法引用级 Mores（receiver/target/type_args）与 Signature。
func (c *MethodRefEnricher) EnrichMetadata(elem *model.CodeElement, node coreast.Node) {
	srcs := c.fCtx.SourceBytes
	extra := elem.Extra

	var receiver, target string
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
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
