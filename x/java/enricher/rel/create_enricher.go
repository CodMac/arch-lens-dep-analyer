package rel

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
)

// CreateEnricher 为实例化关系（Create）提取变量名/数组/构造链等元数据。
type CreateEnricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
	gCtx     *core.GlobalContext
}

// EnrichMetadata 填充 Create 关系的专用属性，并特殊处理 super() 构造链。
func (e *CreateEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	_, _, stmt := GetRelTmpValue(rel)
	src := *e.fCtx.SourceBytes

	if stmt == nil {
		return
	}

	// 1. 通用属性
	rel.Mores[constants.RelAstKind] = stmt.Kind()
	rel.Mores[constants.RelRawText] = stmt.Utf8Text(src)

	// 2. 专用属性提取：变量名 (RelCreateVariableName)
	contextNode := stmt
	if stmt.Kind() == "object_creation_expression" || stmt.Kind() == "array_creation_expression" {
		if p := stmt.Parent(); p != nil && p.Kind() == "variable_declarator" {
			contextNode = p
		}
	}
	if contextNode.Kind() == "variable_declarator" {
		if nameNode := contextNode.ChildByFieldName("name"); nameNode != nil {
			rel.Mores[constants.RelCreateVariableName] = nameNode.Utf8Text(src)
		}
	}

	// 3. 专用属性提取：数组 (RelCreateIsArray)
	if stmt.Kind() == "array_creation_expression" {
		rel.Mores[constants.RelCreateIsArray] = true
	}

	// 4. 特殊处理 super() -> Object 的情况
	if stmt.Kind() == "explicit_constructor_invocation" && strings.Contains(stmt.Utf8Text(src), "super") {
		rel.Target.Name = "Object"
		rel.Target.QualifiedName = "Object"
	}
}
