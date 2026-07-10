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

	if ctxNode.Kind() == "variable_declarator" {
		if nameNode := ctxNode.ChildByFieldName("name"); nameNode != nil {
			rel.Mores[constants.RelCreateVariableName] = nameNode.Utf8Text(src)
		}
	}

	// 3. 专用属性提取：数组 (RelCreateIsArray)
	if ctxNode.Kind() == "array_creation_expression" {
		rel.Mores[constants.RelCreateIsArray] = true
	}

	// 4. 特殊处理 super() -> Object 的情况
	if ctxNode.Kind() == "explicit_constructor_invocation" && strings.Contains(ctxNode.Utf8Text(src), "super") {
		rel.Target.Name = "Object"
		rel.Target.QualifiedName = "Object"
	}
}
