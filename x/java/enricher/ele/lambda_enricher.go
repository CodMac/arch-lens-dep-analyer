package ele

import (
	"fmt"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

type LambdaEnricher struct {
	fCtx *core.FileContext
}

func (c *LambdaEnricher) EnrichMetadata(elem *model.CodeElement, node *tree_sitter.Node) {
	srcs := c.fCtx.SourceBytes
	extra := elem.Extra

	// 1. 提取参数部分
	// Lambda 参数可能是: (a, b) -> ... 或 a -> ... 或 (int a) -> ...
	var paramsStr string
	paramNode := node.ChildByFieldName("parameters")
	if paramNode != nil {
		paramsStr = helper.GetNodeContent(paramNode, *srcs)
	} else {
		// 处理单参数没有括号的情况: s -> s.toLowerCase()
		// 在 tree-sitter-java 中，这种 identifier 会是 lambda_expression 的第一个命名子节点
		if firstChild := node.NamedChild(0); firstChild != nil && firstChild.Kind() == "identifier" {
			paramsStr = helper.GetNodeContent(firstChild, *srcs)
		}
	}
	extra.Mores[constants.LambdaParameters] = paramsStr
	extra.Mores[constants.MethodComplexity] = helper.CalculateComplexity(node)

	// 2. 识别 Body 类型
	// body 可能是 block 或 表达式
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		isBlock := bodyNode.Kind() == "block"
		extra.Mores[constants.LambdaBodyIsBlock] = isBlock

		// 生成更具描述性的 Signature，例如 (s) -> { ... } 或 (a, b) -> expr
		bodyType := "expr"
		if isBlock {
			bodyType = "{...}"
		}
		elem.Signature = fmt.Sprintf("%s -> %s", paramsStr, bodyType)
	}
}
