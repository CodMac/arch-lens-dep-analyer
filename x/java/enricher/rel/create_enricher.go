package rel

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"strings"
)

type CreateEnricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
	gCtx     *core.GlobalContext
}

func (e *CreateEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	_, ctx := GetRelTmpValue(rel)
	src := *e.fCtx.SourceBytes

	if ctx == nil {
		return
	}

	// 1. 通用属性
	rel.Mores[constants.RelAstKind] = ctx.Kind()
	rel.Mores[constants.RelRawText] = ctx.Utf8Text(src)

	// 2. 专用属性提取：变量名 (RelCreateVariableName)
	contextNode := ctx
	if ctx.Kind() == "object_creation_expression" || ctx.Kind() == "array_creation_expression" {
		if p := ctx.Parent(); p != nil && p.Kind() == "variable_declarator" {
			contextNode = p
		}
	}
	if contextNode.Kind() == "variable_declarator" {
		if nameNode := contextNode.ChildByFieldName("name"); nameNode != nil {
			rel.Mores[constants.RelCreateVariableName] = nameNode.Utf8Text(src)
		}
	}

	// 3. 专用属性提取：数组 (RelCreateIsArray)
	if ctx.Kind() == "array_creation_expression" {
		rel.Mores[constants.RelCreateIsArray] = true
	}

	// 4. 特殊处理 super() -> Object 的情况
	if ctx.Kind() == "explicit_constructor_invocation" && strings.Contains(ctx.Utf8Text(src), "super") {
		rel.Target.Name = "Object"
		rel.Target.QualifiedName = "Object"
	}
}
