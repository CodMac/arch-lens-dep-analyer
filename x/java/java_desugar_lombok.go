package java

import (
	"fmt"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// =============================================================================
// Lombok 语法糖处理 (Lombok Syntactic Sugar)
// =============================================================================

func (c *Collector) desugarLombok(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	if elem.Extra == nil || len(elem.Extra.Annotations) == 0 {
		return
	}

	annotations := elem.Extra.Annotations

	if c.hasLombokAnnotation(annotations, "Slf4j") {
		c.lombokSlf4j(elem, node, fCtx)
	}
	if c.hasLombokAnnotation(annotations, "Getter") {
		c.lombokGetter(elem, node, fCtx)
	}
}

func (c *Collector) hasLombokAnnotation(annotations []string, target string) bool {
	for _, anno := range annotations {
		if strings.Contains(anno, "@"+target) || strings.Contains(anno, "lombok."+target) {
			return true
		}
	}
	return false
}

// lombokSlf4j: 处理 @Slf4j 注解
// 生成私有静态final的Logger字段 log
func (c *Collector) lombokSlf4j(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	logFieldName := "log"
	logFieldQN := c.resolver.BuildQualifiedName(elem.QualifiedName, logFieldName)

	// 检查是否已存在名为 "log" 的字段，避免重复生成
	if len(c.findDefinitionsByQN(fCtx, logFieldQN)) > 0 {
		return
	}

	fCtx.AddDefinition(&model.CodeElement{
		Kind:          model.Field,
		Name:          logFieldName,
		QualifiedName: logFieldQN,
		Path:          fCtx.FilePath,
		Location:      elem.Location,
		Signature:     "private static final Logger log",
		Extra: &model.Extra{
			Modifiers:   []string{"private", "static", "final"},
			Annotations: make([]string, 0),
			Mores: map[string]interface{}{
				VariableRawType:         "org.slf4j.Logger",
				FieldIsStatic:           true,
				FieldIsFinal:            true,
				MethodIsLombokGenerated: true,
				LombokAnnotationType:    "Slf4j",
			},
		},
		IsFormSugar: true,
	}, elem.QualifiedName, node)
}

// lombokGetter: 处理 @Getter 注解
// 为每个非static字段生成getter方法
func (c *Collector) lombokGetter(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	fields := c.extractClassFields(elem, fCtx)

	for _, fieldEntry := range fields {
		fieldName := fieldEntry.Element.Name
		fieldType, _ := fieldEntry.Element.Extra.Mores[FieldRawType].(string)

		// 跳过static字段
		isStatic, _ := fieldEntry.Element.Extra.Mores[FieldIsStatic].(bool)
		if isStatic {
			continue
		}

		// 判断字段类型是否为boolean或Boolean，决定getter名称
		var getterName string
		if fieldType == "boolean" || strings.Contains(fieldType, "Boolean") {
			getterName = "is" + strings.ToUpper(fieldName[:1]) + fieldName[1:]
		} else {
			getterName = "get" + strings.ToUpper(fieldName[:1]) + fieldName[1:]
		}

		// 检查方法是否已存在
		if c.checkMethodExists(elem.QualifiedName, getterName, "()", fCtx) {
			continue
		}

		getterQN := c.resolver.BuildQualifiedName(elem.QualifiedName, getterName+"()")

		fCtx.AddDefinition(&model.CodeElement{
			Kind:          model.Method,
			Name:          getterName,
			QualifiedName: getterQN,
			Path:          fCtx.FilePath,
			Location:      elem.Location,
			Signature:     fmt.Sprintf("public %s %s()", fieldType, getterName),
			Extra: &model.Extra{
				Modifiers:   []string{"public"},
				Annotations: make([]string, 0),
				Mores: map[string]interface{}{
					MethodIsLombokGenerated: true,
					LombokAnnotationType:    "Getter",
					MethodReturnType:        fieldType,
				},
			},
			IsFormSugar: true,
		}, elem.QualifiedName, node)
	}
}
