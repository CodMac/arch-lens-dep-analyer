package rel

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type CreateEnricher struct {
	fCtx *core.FileContext
	gCtx *core.GlobalContext
}

func (e *CreateEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	node, _ := rel.Mores[constants.TmpNode].(*sitter.Node)
	ctxNode, _ := rel.Mores[constants.TmpCtxNode].(*sitter.Node)
	if node == nil || ctxNode == nil {
		return
	}
	src := *e.fCtx.SourceBytes

	// 1. 🎯 修复：提取创建表达式绑定或赋值给的变量/字段名称（独立提取，不与 Kind 强绑定）
	if varName := e.findTargetVariableName(ctxNode, src); varName != "" {
		rel.Mores[constants.RelCreateVariableName] = varName
	}

	// 2. 专用属性提取：数组 (RelCreateIsArray)
	if ctxNode.Kind() == "array_creation_expression" {
		rel.Mores[constants.RelCreateIsArray] = true
	}

	// 3. 特殊处理 super() -> Object 的情况
	if ctxNode.Kind() == "explicit_constructor_invocation" && strings.Contains(ctxNode.Utf8Text(src), "super") {
		rel.Target.Name = "Object"
		rel.Target.QualifiedName = "Object"
	}
}

// findTargetVariableName 向上追溯寻找宿主变量或字段的名字
func (e *CreateEnricher) findTargetVariableName(ctxNode *sitter.Node, src []byte) string {
	curr := ctxNode

	// 向上看 2 层（当前上下文节点，以及它的父节点），兼容各种嵌套 AST 结构
	for i := 0; i < 2 && curr != nil; i++ {
		switch curr.Kind() {

		// 场景 A: 变量/字段声明初始化 (如 StringBuilder sb = new StringBuilder("init"))
		case "variable_declarator":
			if nameNode := curr.ChildByFieldName("name"); nameNode != nil {
				return nameNode.Utf8Text(src)
			}

		// 场景 B: 纯赋值表达式 (如 staticMap = new HashMap() 或 this.fieldInstance = new ArrayList())
		case "assignment_expression":
			if leftNode := curr.ChildByFieldName("left"); leftNode != nil {
				// 情况 1: 纯标识符 (staticMap = ...)
				if leftNode.Kind() == "identifier" {
					return leftNode.Utf8Text(src)
				}
				// 情况 2: 显式字段访问 (this.fieldInstance = ...)
				if leftNode.Kind() == "field_access" {
					if fieldNode := leftNode.ChildByFieldName("field"); fieldNode != nil {
						return fieldNode.Utf8Text(src)
					}
				}
			}
		}
		curr = curr.Parent()
	}
	return ""
}
