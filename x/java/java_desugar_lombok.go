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

	if c._hasLombokAnnotation(annotations, "Slf4j") {
		c._lombokSlf4j(elem, node, fCtx)
	}
	if c._hasLombokAnnotation(annotations, "Getter") {
		c._lombokGetter(elem, node, fCtx)
	}
	if c._hasLombokAnnotation(annotations, "Setter") {
		c._lombokSetter(elem, node, fCtx)
	}
	if c._hasLombokAnnotation(annotations, "ToString") {
		c._lombokToString(elem, node, fCtx)
	}
	if c._hasLombokAnnotation(annotations, "EqualsAndHashCode") {
		c._lombokEqualsAndHashCode(elem, node, fCtx)
	}
	if c._hasLombokAnnotation(annotations, "NoArgsConstructor") {
		c._lombokNoArgsConstructor(elem, node, fCtx)
	}
	if c._hasLombokAnnotation(annotations, "AllArgsConstructor") {
		c._lombokAllArgsConstructor(elem, node, fCtx)
	}
	if c._hasLombokAnnotation(annotations, "RequiredArgsConstructor") {
		c._lombokRequiredArgsConstructor(elem, node, fCtx)
	}
	if c._hasLombokAnnotation(annotations, "Data") {
		c._lombokData(elem, node, fCtx)
	}
	if c._hasLombokAnnotation(annotations, "Value") {
		c._lombokValue(elem, node, fCtx)
	}
	if c._hasLombokAnnotation(annotations, "Builder") {
		c._lombokBuilder(elem, node, fCtx)
	}
}

func (c *Collector) _hasLombokAnnotation(annotations []string, target string) bool {
	for _, anno := range annotations {
		if strings.Contains(anno, "@"+target) || strings.Contains(anno, "lombok."+target) {
			return true
		}
	}
	return false
}

// _lombokSlf4j: 处理 @Slf4j 注解
// 生成私有静态final的Logger字段 log
func (c *Collector) _lombokSlf4j(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
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

// _lombokGetter: 处理 @Getter 注解
// 为每个非static字段生成getter方法
func (c *Collector) _lombokGetter(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
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

// _lombokSetter: 处理 @Setter 注解
// 为每个非static、非final字段生成setter方法
func (c *Collector) _lombokSetter(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	fields := c.extractClassFields(elem, fCtx)

	for _, fieldEntry := range fields {
		fieldName := fieldEntry.Element.Name
		fieldType, _ := fieldEntry.Element.Extra.Mores[FieldRawType].(string)

		// 跳过static字段
		isStatic, _ := fieldEntry.Element.Extra.Mores[FieldIsStatic].(bool)
		if isStatic {
			continue
		}

		// 跳过final字段
		isFinal, _ := fieldEntry.Element.Extra.Mores[FieldIsFinal].(bool)
		if isFinal {
			continue
		}

		// 生成setter名称
		setterName := "set" + strings.ToUpper(fieldName[:1]) + fieldName[1:]

		// 检查方法是否已存在
		if c.checkMethodExists(elem.QualifiedName, setterName, "("+fieldType+")", fCtx) {
			continue
		}

		setterQN := c.resolver.BuildQualifiedName(elem.QualifiedName, setterName+"("+fieldType+")")

		fCtx.AddDefinition(&model.CodeElement{
			Kind:          model.Method,
			Name:          setterName,
			QualifiedName: setterQN,
			Path:          fCtx.FilePath,
			Location:      elem.Location,
			Signature:     fmt.Sprintf("public void %s(%s %s)", setterName, fieldType, fieldName),
			Extra: &model.Extra{
				Modifiers:   []string{"public"},
				Annotations: make([]string, 0),
				Mores: map[string]interface{}{
					MethodIsLombokGenerated: true,
					LombokAnnotationType:    "Setter",
					MethodParameters:        []string{fieldType + " " + fieldName},
				},
			},
			IsFormSugar: true,
		}, elem.QualifiedName, node)
	}
}

// _lombokToString: 处理 @ToString 注解
// 生成 toString() 方法
func (c *Collector) _lombokToString(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	// 检查 toString() 是否已存在
	if c.checkMethodExists(elem.QualifiedName, "toString", "()", fCtx) {
		return
	}

	toStringQN := c.resolver.BuildQualifiedName(elem.QualifiedName, "toString()")

	fCtx.AddDefinition(&model.CodeElement{
		Kind:          model.Method,
		Name:          "toString",
		QualifiedName: toStringQN,
		Path:          fCtx.FilePath,
		Location:      elem.Location,
		Signature:     "public String toString()",
		Extra: &model.Extra{
			Modifiers:   []string{"public"},
			Annotations: make([]string, 0),
			Mores: map[string]interface{}{
				MethodIsLombokGenerated: true,
				LombokAnnotationType:    "ToString",
				MethodReturnType:        "String",
			},
		},
		IsFormSugar: true,
	}, elem.QualifiedName, node)
}

// _lombokEqualsAndHashCode: 处理 @EqualsAndHashCode 注解
// 生成 equals(), hashCode(), canEqual() 三个方法
func (c *Collector) _lombokEqualsAndHashCode(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	// 1. 生成 equals(Object)
	if !c.checkMethodExists(elem.QualifiedName, "equals", "(Object)", fCtx) {
		equalsQN := c.resolver.BuildQualifiedName(elem.QualifiedName, "equals(Object)")
		fCtx.AddDefinition(&model.CodeElement{
			Kind:          model.Method,
			Name:          "equals",
			QualifiedName: equalsQN,
			Path:          fCtx.FilePath,
			Location:      elem.Location,
			Signature:     "public boolean equals(Object)",
			Extra: &model.Extra{
				Modifiers:   []string{"public"},
				Annotations: make([]string, 0),
				Mores: map[string]interface{}{
					MethodIsLombokGenerated: true,
					LombokAnnotationType:    "EqualsAndHashCode",
					MethodReturnType:        "boolean",
					MethodParameters:        []string{"Object obj"},
				},
			},
			IsFormSugar: true,
		}, elem.QualifiedName, node)
	}

	// 2. 生成 hashCode()
	if !c.checkMethodExists(elem.QualifiedName, "hashCode", "()", fCtx) {
		hashCodeQN := c.resolver.BuildQualifiedName(elem.QualifiedName, "hashCode()")
		fCtx.AddDefinition(&model.CodeElement{
			Kind:          model.Method,
			Name:          "hashCode",
			QualifiedName: hashCodeQN,
			Path:          fCtx.FilePath,
			Location:      elem.Location,
			Signature:     "public int hashCode()",
			Extra: &model.Extra{
				Modifiers:   []string{"public"},
				Annotations: make([]string, 0),
				Mores: map[string]interface{}{
					MethodIsLombokGenerated: true,
					LombokAnnotationType:    "EqualsAndHashCode",
					MethodReturnType:        "int",
				},
			},
			IsFormSugar: true,
		}, elem.QualifiedName, node)
	}

	// 3. 生成 canEqual(Object)
	if !c.checkMethodExists(elem.QualifiedName, "canEqual", "(Object)", fCtx) {
		canEqualQN := c.resolver.BuildQualifiedName(elem.QualifiedName, "canEqual(Object)")
		fCtx.AddDefinition(&model.CodeElement{
			Kind:          model.Method,
			Name:          "canEqual",
			QualifiedName: canEqualQN,
			Path:          fCtx.FilePath,
			Location:      elem.Location,
			Signature:     "protected boolean canEqual(Object)",
			Extra: &model.Extra{
				Modifiers:   []string{"protected"},
				Annotations: make([]string, 0),
				Mores: map[string]interface{}{
					MethodIsLombokGenerated: true,
					LombokAnnotationType:    "EqualsAndHashCode",
					MethodReturnType:        "boolean",
					MethodParameters:        []string{"Object obj"},
				},
			},
			IsFormSugar: true,
		}, elem.QualifiedName, node)
	}
}

// _lombokNoArgsConstructor: 处理 @NoArgsConstructor 注解
// 生成无参构造器
func (c *Collector) _lombokNoArgsConstructor(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	// 检查无参构造器是否已存在
	if c.checkMethodExists(elem.QualifiedName, elem.Name, "()", fCtx) {
		return
	}

	consQN := c.resolver.BuildQualifiedName(elem.QualifiedName, elem.Name+"()")

	fCtx.AddDefinition(&model.CodeElement{
		Kind:          model.Method,
		Name:          elem.Name,
		QualifiedName: consQN,
		Path:          fCtx.FilePath,
		Location:      elem.Location,
		Signature:     fmt.Sprintf("public %s()", elem.Name),
		Extra: &model.Extra{
			Modifiers:   []string{"public"},
			Annotations: make([]string, 0),
			Mores: map[string]interface{}{
				MethodIsConstructor:     true,
				MethodIsLombokGenerated: true,
				LombokAnnotationType:    "NoArgsConstructor",
			},
		},
		IsFormSugar: true,
	}, elem.QualifiedName, node)
}

// _lombokAllArgsConstructor: 处理 @AllArgsConstructor 注解
// 生成全参构造器
func (c *Collector) _lombokAllArgsConstructor(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	fields := c.extractClassFields(elem, fCtx)

	// 过滤掉static字段
	var nonStaticFields []*core.DefinitionEntry
	for _, field := range fields {
		isStatic, _ := field.Element.Extra.Mores[FieldIsStatic].(bool)
		if !isStatic {
			nonStaticFields = append(nonStaticFields, field)
		}
	}

	if len(nonStaticFields) == 0 {
		return
	}

	var paramTypes []string
	var paramNames []string
	var paramStrings []string
	for _, field := range nonStaticFields {
		fieldType, _ := field.Element.Extra.Mores[FieldRawType].(string)
		paramType := strings.TrimSpace(strings.Split(fieldType, "<")[0])
		paramTypes = append(paramTypes, paramType)
		paramNames = append(paramNames, field.Element.Name)
		paramStrings = append(paramStrings, paramType+" "+field.Element.Name)
	}

	// 检查是否已存在
	paramsStr := "(" + strings.Join(paramTypes, ",") + ")"
	if c.checkMethodExists(elem.QualifiedName, elem.Name, paramsStr, fCtx) {
		return
	}

	consQN := c.resolver.BuildQualifiedName(elem.QualifiedName, elem.Name+paramsStr)

	fCtx.AddDefinition(&model.CodeElement{
		Kind:          model.Method,
		Name:          elem.Name,
		QualifiedName: consQN,
		Path:          fCtx.FilePath,
		Location:      elem.Location,
		Signature:     fmt.Sprintf("public %s(%s)", elem.Name, strings.Join(paramStrings, ", ")),
		Extra: &model.Extra{
			Modifiers:   []string{"public"},
			Annotations: make([]string, 0),
			Mores: map[string]interface{}{
				MethodIsConstructor:     true,
				MethodIsLombokGenerated: true,
				LombokAnnotationType:    "AllArgsConstructor",
				MethodParameters:        paramStrings,
			},
		},
		IsFormSugar: true,
	}, elem.QualifiedName, node)
}

// _lombokRequiredArgsConstructor: 处理 @RequiredArgsConstructor 注解
// 为final字段生成构造器
func (c *Collector) _lombokRequiredArgsConstructor(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	fields := c.extractClassFields(elem, fCtx)

	// 只收集final且非static的字段
	var finalFields []*core.DefinitionEntry
	for _, field := range fields {
		isStatic, _ := field.Element.Extra.Mores[FieldIsStatic].(bool)
		isFinal, _ := field.Element.Extra.Mores[FieldIsFinal].(bool)
		if !isStatic && isFinal {
			finalFields = append(finalFields, field)
		}
	}

	if len(finalFields) == 0 {
		return
	}

	var paramTypes []string
	var paramNames []string
	var paramStrings []string
	for _, field := range finalFields {
		fieldType, _ := field.Element.Extra.Mores[FieldRawType].(string)
		paramType := strings.TrimSpace(strings.Split(fieldType, "<")[0])
		paramTypes = append(paramTypes, paramType)
		paramNames = append(paramNames, field.Element.Name)
		paramStrings = append(paramStrings, paramType+" "+field.Element.Name)
	}

	// 检查是否已存在
	paramsStr := "(" + strings.Join(paramTypes, ",") + ")"
	if c.checkMethodExists(elem.QualifiedName, elem.Name, paramsStr, fCtx) {
		return
	}

	consQN := c.resolver.BuildQualifiedName(elem.QualifiedName, elem.Name+paramsStr)

	fCtx.AddDefinition(&model.CodeElement{
		Kind:          model.Method,
		Name:          elem.Name,
		QualifiedName: consQN,
		Path:          fCtx.FilePath,
		Location:      elem.Location,
		Signature:     fmt.Sprintf("public %s(%s)", elem.Name, strings.Join(paramStrings, ", ")),
		Extra: &model.Extra{
			Modifiers:   []string{"public"},
			Annotations: make([]string, 0),
			Mores: map[string]interface{}{
				MethodIsConstructor:     true,
				MethodIsLombokGenerated: true,
				LombokAnnotationType:    "RequiredArgsConstructor",
				MethodParameters:        paramStrings,
			},
		},
		IsFormSugar: true,
	}, elem.QualifiedName, node)
}

// _lombokData: 处理 @Data 注解
// @Data 等价于 @Getter + @Setter + @ToString + @EqualsAndHashCode + @NoArgsConstructor
func (c *Collector) _lombokData(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	c._lombokGetter(elem, node, fCtx)
	c._lombokSetter(elem, node, fCtx)
	c._lombokToString(elem, node, fCtx)
	c._lombokEqualsAndHashCode(elem, node, fCtx)
	c._lombokNoArgsConstructor(elem, node, fCtx)
}

// _lombokValue: 处理 @Value 注解
// @Value 等价于 @Getter + @AllArgsConstructor + @ToString + @EqualsAndHashCode
// 不生成 setter，生成全参构造器
func (c *Collector) _lombokValue(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	c._lombokGetter(elem, node, fCtx)
	c._lombokAllArgsConstructor(elem, node, fCtx)
	c._lombokToString(elem, node, fCtx)
	c._lombokEqualsAndHashCode(elem, node, fCtx)
}

// _lombokBuilder: 处理 @Builder 注解
// 生成Builder内部类和相关方法
func (c *Collector) _lombokBuilder(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	fields := c.extractClassFields(elem, fCtx)

	// 过滤掉static字段
	var nonStaticFields []*core.DefinitionEntry
	for _, field := range fields {
		isStatic, _ := field.Element.Extra.Mores[FieldIsStatic].(bool)
		if !isStatic {
			nonStaticFields = append(nonStaticFields, field)
		}
	}

	builderClassName := "Builder"
	builderQN := c.resolver.BuildQualifiedName(elem.QualifiedName, builderClassName)

	// 1. 检查Builder内部类是否已存在
	if len(c.findDefinitionsByQN(fCtx, builderQN)) == 0 {
		fCtx.AddDefinition(&model.CodeElement{
			Kind:          model.Class,
			Name:          builderClassName,
			QualifiedName: builderQN,
			Path:          fCtx.FilePath,
			Location:      elem.Location,
			Signature:     "public static class Builder",
			Extra: &model.Extra{
				Modifiers:   []string{"public", "static"},
				Annotations: make([]string, 0),
				Mores: map[string]interface{}{
					FieldIsStatic:           false,
					MethodIsLombokGenerated: true,
					LombokAnnotationType:    "Builder",
				},
			},
			IsFormSugar: true,
		}, elem.QualifiedName, node)
	}

	// 2. 生成静态builder()方法
	builderMethodName := "builder"
	builderMethodQN := c.resolver.BuildQualifiedName(elem.QualifiedName, builderMethodName+"()")
	if !c.checkMethodExists(elem.QualifiedName, builderMethodName, "()", fCtx) {
		fCtx.AddDefinition(&model.CodeElement{
			Kind:          model.Method,
			Name:          builderMethodName,
			QualifiedName: builderMethodQN,
			Path:          fCtx.FilePath,
			Location:      elem.Location,
			Signature:     fmt.Sprintf("public static %s builder()", builderClassName),
			Extra: &model.Extra{
				Modifiers:   []string{"public", "static"},
				Annotations: make([]string, 0),
				Mores: map[string]interface{}{
					MethodIsLombokGenerated: true,
					LombokAnnotationType:    "Builder",
					MethodReturnType:        builderClassName,
				},
			},
			IsFormSugar: true,
		}, elem.QualifiedName, node)
	}

	// 3. 在Builder类中生成链式setter方法
	for _, field := range nonStaticFields {
		fieldName := field.Element.Name
		fieldType, _ := field.Element.Extra.Mores[FieldRawType].(string)

		setterQN := c.resolver.BuildQualifiedName(builderQN, fieldName+"("+fieldType+")")
		if len(c.findDefinitionsByQN(fCtx, setterQN)) == 0 {
			fCtx.AddDefinition(&model.CodeElement{
				Kind:          model.Method,
				Name:          fieldName,
				QualifiedName: setterQN,
				Path:          fCtx.FilePath,
				Location:      elem.Location,
				Signature:     fmt.Sprintf("public %s %s(%s)", builderClassName, fieldName, fieldType),
				Extra: &model.Extra{
					Modifiers:   []string{"public"},
					Annotations: make([]string, 0),
					Mores: map[string]interface{}{
						MethodIsLombokGenerated: true,
						LombokAnnotationType:    "Builder",
						MethodReturnType:        builderClassName,
						MethodParameters:        []string{fieldType + " " + fieldName},
					},
				},
				IsFormSugar: true,
			}, builderQN, node)
		}
	}

	// 4. 在Builder类中生成build()方法
	buildQN := c.resolver.BuildQualifiedName(builderQN, "build()")
	if !c.checkMethodExists(builderQN, "build", "()", fCtx) {
		fCtx.AddDefinition(&model.CodeElement{
			Kind:          model.Method,
			Name:          "build",
			QualifiedName: buildQN,
			Path:          fCtx.FilePath,
			Location:      elem.Location,
			Signature:     fmt.Sprintf("public %s build()", elem.Name),
			Extra: &model.Extra{
				Modifiers:   []string{"public"},
				Annotations: make([]string, 0),
				Mores: map[string]interface{}{
					MethodIsLombokGenerated: true,
					LombokAnnotationType:    "Builder",
					MethodReturnType:        elem.Name,
				},
			},
			IsFormSugar: true,
		}, builderQN, node)
	}

	// 5. 在主类中生成私有全参构造器（接受Builder参数）
	builderParam := "builder"
	privateConstructorQN := c.resolver.BuildQualifiedName(elem.QualifiedName, fmt.Sprintf("%s(%s)", elem.Name, builderClassName))
	if !c.checkMethodExists(elem.QualifiedName, elem.Name, "("+builderClassName+")", fCtx) {
		fCtx.AddDefinition(&model.CodeElement{
			Kind:          model.Method,
			Name:          elem.Name,
			QualifiedName: privateConstructorQN,
			Path:          fCtx.FilePath,
			Location:      elem.Location,
			Signature:     fmt.Sprintf("private %s(%s %s)", elem.Name, builderClassName, builderParam),
			Extra: &model.Extra{
				Modifiers:   []string{"private"},
				Annotations: make([]string, 0),
				Mores: map[string]interface{}{
					MethodIsConstructor:     true,
					MethodIsLombokGenerated: true,
					LombokAnnotationType:    "Builder",
					MethodParameters:        []string{builderClassName + " " + builderParam},
				},
			},
			IsFormSugar: true,
		}, elem.QualifiedName, node)
	}
}
