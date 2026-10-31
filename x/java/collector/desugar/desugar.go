// Package collectordesugar 处理 Java 语法糖（Record/Enum/默认构造器/Lombok 全系），
// 生成编译器语义上隐式存在的代码元素（IsFormSugar=true）。
// 作为 collector 包的生命周期子阶段（CollectDefinitions 的 Step 5）内嵌其下。
package collectordesugar

import (
	"fmt"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

// DeSugar 提供 Java 语法糖元素的生成能力。
type DeSugar struct {
	resolver core.SymbolResolver
}

// NewDeSugar 创建语法糖处理器。
func NewDeSugar(resolver core.SymbolResolver) *DeSugar {
	return &DeSugar{resolver: resolver}
}

// addDef 便捷：以 DefinitionEntry 形态向文件上下文添加定义。
func addDef(fCtx *core.FileContext, elem *model.CodeElement, parentQN string, node coreast.Node) {
	fCtx.AddDefinition(&core.DefinitionEntry{Element: elem, ParentQN: parentQN, Node: node})
}

// =============================================================================
// 原生 Java 语法糖处理 (Native Java Syntactic Sugar)
// =============================================================================

// DesugarDefaultConstructor 为无构造器的类生成默认构造器。
func (c *DeSugar) DesugarDefaultConstructor(elem *model.CodeElement, node coreast.Node, fCtx *core.FileContext) {
	body := node.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := 0; i < body.NamedChildCount(); i++ {
		if body.NamedChild(i).Kind() == "constructor_declaration" {
			return
		}
	}

	// 带有接管构造器生成的 Lombok 注解时跳过默认合成：@Value/@RequiredArgsConstructor/@Builder
	// 不生成无参构造器，@NoArgsConstructor/@Data 由 Lombok 生成器产出（语义一致）。
	if c.hasLombokConstructorAnnotation(elem) {
		return
	}

	consName := elem.Name
	consQN := c.resolver.BuildQualifiedName(elem.QualifiedName, consName+"()")
	if _, ok := fCtx.FindByQualifiedName(consQN); !ok {
		addDef(fCtx, &model.CodeElement{
			Kind:          model.Method,
			Name:          consName,
			QualifiedName: consQN,
			Path:          fCtx.FilePath,
			Location:      elem.Location,
			Signature:     fmt.Sprintf("public %s()", consName),
			Extra: &model.Extra{
				Modifiers:   make([]string, 0),
				Annotations: make([]string, 0),
				Mores: map[string]interface{}{
					constants.MethodIsConstructor: true,
					constants.MethodIsImplicit:    true,
				},
			},
			IsFormSugar: true,
		}, elem.QualifiedName, node)
	}
}

// DesugarEnumMethods 为枚举生成 values() 与 valueOf(String) 方法。
func (c *DeSugar) DesugarEnumMethods(elem *model.CodeElement, node coreast.Node, fCtx *core.FileContext) {
	vQN := c.resolver.BuildQualifiedName(elem.QualifiedName, "values()")
	if _, ok := fCtx.FindByQualifiedName(vQN); !ok {
		addDef(fCtx, &model.CodeElement{
			Kind: model.Method, Name: "values", QualifiedName: vQN, Path: fCtx.FilePath, Location: elem.Location, IsFormSugar: true,
			Signature: fmt.Sprintf("public static %s[] values()", elem.Name),
			Extra: &model.Extra{
				Modifiers:   make([]string, 0),
				Annotations: make([]string, 0),
				Mores: map[string]interface{}{
					constants.MethodIsConstructor: false,
					constants.MethodIsImplicit:    true,
					constants.MethodReturnType:    elem.Name + "[]",
				},
			},
		}, elem.QualifiedName, node)
	}

	voQN := c.resolver.BuildQualifiedName(elem.QualifiedName, "valueOf(String)")
	if _, ok := fCtx.FindByQualifiedName(voQN); !ok {
		addDef(fCtx, &model.CodeElement{
			Kind: model.Method, Name: "valueOf", QualifiedName: voQN, Path: fCtx.FilePath, Location: elem.Location, IsFormSugar: true,
			Signature: fmt.Sprintf("public static %s valueOf(String name)", elem.Name),
			Extra: &model.Extra{
				Modifiers:   make([]string, 0),
				Annotations: make([]string, 0),
				Mores: map[string]interface{}{
					constants.MethodIsConstructor: false,
					constants.MethodIsImplicit:    true,
					constants.MethodReturnType:    elem.Name,
					constants.MethodParameters:    []string{"String name"},
				},
			},
		}, elem.QualifiedName, node)
	}
}

// DesugarRecordMembers 为 Record 生成组件字段与访问器方法、全参构造器。
func (c *DeSugar) DesugarRecordMembers(elem *model.CodeElement, node coreast.Node, fCtx *core.FileContext) {
	paramList := helper.FindNamedChildOfType(node, "formal_parameters")
	if paramList == nil {
		return
	}

	type component struct{ name, vType string }
	var comps []component
	for i := 0; i < paramList.NamedChildCount(); i++ {
		child := paramList.NamedChild(i)
		if child.Kind() == "formal_parameter" {
			comps = append(comps, component{
				name:  helper.GetNodeContent(child.ChildByFieldName("name"), *fCtx.SourceBytes),
				vType: helper.GetNodeContent(child.ChildByFieldName("type"), *fCtx.SourceBytes),
			})
		}
	}

	for _, comp := range comps {
		fieldQN := c.resolver.BuildQualifiedName(elem.QualifiedName, comp.name)
		if defs, _ := fCtx.FindByShortName(comp.name); len(defs) > 0 {
			for _, d := range defs {
				if d.Element.QualifiedName == fieldQN {
					d.Element.Kind = model.Field
					// Record 组件字段由源码变量改写而来，补齐字段语义元数据，与 FieldEnricher 产出对齐。
					// 保留 variable 键不删（extractor/rel 双键查询兼容），仅补字段标准键。
					d.Element.Extra.Mores[constants.FieldIsRecordComponent] = true
					d.Element.Extra.Mores[constants.FieldIsFinal] = true
					if _, ok := d.Element.Extra.Mores[constants.FieldRawType]; !ok {
						d.Element.Extra.Mores[constants.FieldRawType] = comp.vType
					}
				}
			}
		}
		mIdentity := comp.name + "()"
		mQN := c.resolver.BuildQualifiedName(elem.QualifiedName, mIdentity)
		if _, ok := fCtx.FindByQualifiedName(mQN); !ok {
			addDef(fCtx, &model.CodeElement{
				Kind: model.Method, Name: comp.name, QualifiedName: mQN, Path: fCtx.FilePath, Location: elem.Location, IsFormSugar: true,
				Signature: fmt.Sprintf("public %s %s()", comp.vType, comp.name),
				Extra: &model.Extra{
					Modifiers:   make([]string, 0),
					Annotations: make([]string, 0),
					Mores: map[string]interface{}{
						constants.MethodIsConstructor: false,
						constants.MethodIsImplicit:    true,
						constants.MethodReturnType:    comp.vType,
					},
				},
			}, elem.QualifiedName, node)
		}
	}

	var pTypes []string
	for _, comp := range comps {
		pTypes = append(pTypes, strings.TrimSpace(strings.Split(comp.vType, "<")[0]))
	}
	cIdentity := fmt.Sprintf("%s(%s)", elem.Name, strings.Join(pTypes, ","))
	cQN := c.resolver.BuildQualifiedName(elem.QualifiedName, cIdentity)
	if _, ok := fCtx.FindByQualifiedName(cQN); !ok {
		var pStrings []string
		for _, comp := range comps {
			pStrings = append(pStrings, comp.vType+" "+comp.name)
		}
		addDef(fCtx, &model.CodeElement{
			Kind: model.Method, Name: elem.Name, QualifiedName: cQN, Path: fCtx.FilePath, Location: elem.Location, IsFormSugar: true,
			Signature: fmt.Sprintf("public %s(%s)", elem.Name, strings.Join(pStrings, ", ")),
			Extra: &model.Extra{
				Modifiers:   make([]string, 0),
				Annotations: make([]string, 0),
				Mores: map[string]interface{}{
					constants.MethodIsConstructor: true,
					constants.MethodIsImplicit:    true,
					constants.MethodParameters:    pStrings,
				},
			},
		}, elem.QualifiedName, node)
	}

	// Record 隐式 Object 方法：equals(Object) / hashCode() / toString()。
	// 若用户已显式定义同名方法则跳过（FindByQualifiedName 已包含源码定义与语法糖定义）。
	recordObjectMethods := []struct {
		name      string
		identity  string
		signature string
		retType   string
		params    []string
	}{
		{"equals", "equals(Object)", "public boolean equals(Object o)", "boolean", []string{"Object o"}},
		{"hashCode", "hashCode()", "public int hashCode()", "int", nil},
		{"toString", "toString()", "public String toString()", "String", nil},
	}
	for _, m := range recordObjectMethods {
		mQN := c.resolver.BuildQualifiedName(elem.QualifiedName, m.identity)
		if _, ok := fCtx.FindByQualifiedName(mQN); ok {
			continue
		}
		mores := map[string]interface{}{
			constants.MethodIsConstructor: false,
			constants.MethodIsImplicit:    true,
			constants.MethodReturnType:    m.retType,
		}
		if len(m.params) > 0 {
			mores[constants.MethodParameters] = m.params
		}
		addDef(fCtx, &model.CodeElement{
			Kind: model.Method, Name: m.name, QualifiedName: mQN, Path: fCtx.FilePath, Location: elem.Location, IsFormSugar: true,
			Signature: m.signature,
			Extra: &model.Extra{
				Modifiers:   make([]string, 0),
				Annotations: make([]string, 0),
				Mores:       mores,
			},
		}, elem.QualifiedName, node)
	}
}

// =============================================================================
// Lombok 语法糖处理 (Lombok Syntactic Sugar)
// =============================================================================

// DesugarLombok 按类上的 Lombok 注解分发到具体生成器。
func (c *DeSugar) DesugarLombok(elem *model.CodeElement, node coreast.Node, fCtx *core.FileContext) {
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
	if c.hasLombokAnnotation(annotations, "Setter") {
		c.lombokSetter(elem, node, fCtx)
	}
	if c.hasLombokAnnotation(annotations, "ToString") {
		c.lombokToString(elem, node, fCtx)
	}
	if c.hasLombokAnnotation(annotations, "EqualsAndHashCode") {
		c.lombokEqualsAndHashCode(elem, node, fCtx)
	}
	if c.hasLombokAnnotation(annotations, "NoArgsConstructor") {
		c.lombokNoArgsConstructor(elem, node, fCtx)
	}
	if c.hasLombokAnnotation(annotations, "AllArgsConstructor") {
		c.lombokAllArgsConstructor(elem, node, fCtx)
	}
	if c.hasLombokAnnotation(annotations, "RequiredArgsConstructor") {
		c.lombokRequiredArgsConstructor(elem, node, fCtx)
	}
	if c.hasLombokAnnotation(annotations, "Data") {
		c.lombokData(elem, node, fCtx)
	}
	if c.hasLombokAnnotation(annotations, "Value") {
		c.lombokValue(elem, node, fCtx)
	}
	if c.hasLombokAnnotation(annotations, "Builder") {
		c.lombokBuilder(elem, node, fCtx)
	}
}

func (c *DeSugar) hasLombokAnnotation(annotations []string, target string) bool {
	for _, anno := range annotations {
		if strings.Contains(anno, "@"+target) || strings.Contains(anno, "lombok."+target) {
			return true
		}
	}
	return false
}

// hasLombokConstructorAnnotation 判断类是否携带接管构造器生成的 Lombok 注解。
// 这些注解要么不生成无参构造器（Value/RequiredArgsConstructor/Builder），
// 要么由 Lombok 生成器按注解语义精确产出（NoArgsConstructor/AllArgsConstructor/Data），
// 因此默认构造器合成必须跳过，避免误生成与真实 Lombok 产物不符的无参构造器。
func (c *DeSugar) hasLombokConstructorAnnotation(elem *model.CodeElement) bool {
	if elem.Extra == nil || len(elem.Extra.Annotations) == 0 {
		return false
	}
	for _, name := range []string{
		"NoArgsConstructor", "AllArgsConstructor", "RequiredArgsConstructor",
		"Data", "Value", "Builder",
	} {
		if c.hasLombokAnnotation(elem.Extra.Annotations, name) {
			return true
		}
	}
	return false
}

// lombokSlf4j 处理 @Slf4j：生成私有静态 final 的 Logger 字段 log。
func (c *DeSugar) lombokSlf4j(elem *model.CodeElement, node coreast.Node, fCtx *core.FileContext) {
	logFieldName := "log"
	logFieldQN := c.resolver.BuildQualifiedName(elem.QualifiedName, logFieldName)

	if _, ok := fCtx.FindByQualifiedName(logFieldQN); ok {
		return
	}

	addDef(fCtx, &model.CodeElement{
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
				constants.FieldRawType:            "org.slf4j.Logger",
				constants.FieldIsStatic:           true,
				constants.FieldIsFinal:            true,
				constants.MethodIsLombokGenerated: true,
				constants.LombokAnnotationType:    "Slf4j",
			},
		},
		IsFormSugar: true,
	}, elem.QualifiedName, node)
}

// lombokGetter 处理 @Getter：为每个非 static 字段生成 getter 方法。
func (c *DeSugar) lombokGetter(elem *model.CodeElement, node coreast.Node, fCtx *core.FileContext) {
	fields := c.extractClassFields(elem, fCtx)

	for _, fieldEntry := range fields {
		fieldName := fieldEntry.Element.Name
		fieldType, _ := fieldEntry.Element.Extra.Mores[constants.FieldRawType].(string)

		isStatic, _ := fieldEntry.Element.Extra.Mores[constants.FieldIsStatic].(bool)
		if isStatic {
			continue
		}

		var getterName string
		if fieldType == "boolean" || strings.Contains(fieldType, "Boolean") {
			getterName = "is" + strings.ToUpper(fieldName[:1]) + fieldName[1:]
		} else {
			getterName = "get" + strings.ToUpper(fieldName[:1]) + fieldName[1:]
		}

		if c.checkMethodExists(elem.QualifiedName, getterName, "()", fCtx) {
			continue
		}

		getterQN := c.resolver.BuildQualifiedName(elem.QualifiedName, getterName+"()")

		addDef(fCtx, &model.CodeElement{
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
					constants.MethodIsConstructor:     false,
					constants.MethodIsLombokGenerated: true,
					constants.LombokAnnotationType:    "Getter",
					constants.MethodReturnType:        fieldType,
				},
			},
			IsFormSugar: true,
		}, elem.QualifiedName, node)
	}
}

// lombokSetter 处理 @Setter：为每个非 static 非 final 字段生成 setter 方法。
func (c *DeSugar) lombokSetter(elem *model.CodeElement, node coreast.Node, fCtx *core.FileContext) {
	fields := c.extractClassFields(elem, fCtx)

	for _, fieldEntry := range fields {
		fieldName := fieldEntry.Element.Name
		fieldType, _ := fieldEntry.Element.Extra.Mores[constants.FieldRawType].(string)

		isStatic, _ := fieldEntry.Element.Extra.Mores[constants.FieldIsStatic].(bool)
		if isStatic {
			continue
		}

		isFinal, _ := fieldEntry.Element.Extra.Mores[constants.FieldIsFinal].(bool)
		if isFinal {
			continue
		}

		setterName := "set" + strings.ToUpper(fieldName[:1]) + fieldName[1:]

		if c.checkMethodExists(elem.QualifiedName, setterName, "("+fieldType+")", fCtx) {
			continue
		}

		setterQN := c.resolver.BuildQualifiedName(elem.QualifiedName, setterName+"("+fieldType+")")

		addDef(fCtx, &model.CodeElement{
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
					constants.MethodIsConstructor:     false,
					constants.MethodIsLombokGenerated: true,
					constants.LombokAnnotationType:    "Setter",
					constants.MethodParameters:        []string{fieldType + " " + fieldName},
				},
			},
			IsFormSugar: true,
		}, elem.QualifiedName, node)
	}
}

// lombokToString 处理 @ToString：生成 toString() 方法。
func (c *DeSugar) lombokToString(elem *model.CodeElement, node coreast.Node, fCtx *core.FileContext) {
	if c.checkMethodExists(elem.QualifiedName, "toString", "()", fCtx) {
		return
	}

	toStringQN := c.resolver.BuildQualifiedName(elem.QualifiedName, "toString()")

	addDef(fCtx, &model.CodeElement{
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
				constants.MethodIsConstructor:     false,
				constants.MethodIsLombokGenerated: true,
				constants.LombokAnnotationType:    "ToString",
				constants.MethodReturnType:        "String",
			},
		},
		IsFormSugar: true,
	}, elem.QualifiedName, node)
}

// lombokEqualsAndHashCode 处理 @EqualsAndHashCode：生成 equals/hashCode/canEqual。
func (c *DeSugar) lombokEqualsAndHashCode(elem *model.CodeElement, node coreast.Node, fCtx *core.FileContext) {
	if !c.checkMethodExists(elem.QualifiedName, "equals", "(Object)", fCtx) {
		equalsQN := c.resolver.BuildQualifiedName(elem.QualifiedName, "equals(Object)")
		addDef(fCtx, &model.CodeElement{
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
					constants.MethodIsConstructor:     false,
					constants.MethodIsLombokGenerated: true,
					constants.LombokAnnotationType:    "EqualsAndHashCode",
					constants.MethodReturnType:        "boolean",
					constants.MethodParameters:        []string{"Object obj"},
				},
			},
			IsFormSugar: true,
		}, elem.QualifiedName, node)
	}

	if !c.checkMethodExists(elem.QualifiedName, "hashCode", "()", fCtx) {
		hashCodeQN := c.resolver.BuildQualifiedName(elem.QualifiedName, "hashCode()")
		addDef(fCtx, &model.CodeElement{
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
					constants.MethodIsConstructor:     false,
					constants.MethodIsLombokGenerated: true,
					constants.LombokAnnotationType:    "EqualsAndHashCode",
					constants.MethodReturnType:        "int",
				},
			},
			IsFormSugar: true,
		}, elem.QualifiedName, node)
	}

	if !c.checkMethodExists(elem.QualifiedName, "canEqual", "(Object)", fCtx) {
		canEqualQN := c.resolver.BuildQualifiedName(elem.QualifiedName, "canEqual(Object)")
		addDef(fCtx, &model.CodeElement{
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
					constants.MethodIsConstructor:     false,
					constants.MethodIsLombokGenerated: true,
					constants.LombokAnnotationType:    "EqualsAndHashCode",
					constants.MethodReturnType:        "boolean",
					constants.MethodParameters:        []string{"Object obj"},
				},
			},
			IsFormSugar: true,
		}, elem.QualifiedName, node)
	}
}

// lombokNoArgsConstructor 处理 @NoArgsConstructor：生成无参构造器。
func (c *DeSugar) lombokNoArgsConstructor(elem *model.CodeElement, node coreast.Node, fCtx *core.FileContext) {
	if c.checkMethodExists(elem.QualifiedName, elem.Name, "()", fCtx) {
		return
	}

	consQN := c.resolver.BuildQualifiedName(elem.QualifiedName, elem.Name+"()")

	addDef(fCtx, &model.CodeElement{
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
				constants.MethodIsConstructor:     true,
				constants.MethodIsLombokGenerated: true,
				constants.LombokAnnotationType:    "NoArgsConstructor",
			},
		},
		IsFormSugar: true,
	}, elem.QualifiedName, node)
}

// lombokAllArgsConstructor 处理 @AllArgsConstructor：生成全参构造器。
func (c *DeSugar) lombokAllArgsConstructor(elem *model.CodeElement, node coreast.Node, fCtx *core.FileContext) {
	fields := c.extractClassFields(elem, fCtx)

	var nonStaticFields []*core.DefinitionEntry
	for _, field := range fields {
		isStatic, _ := field.Element.Extra.Mores[constants.FieldIsStatic].(bool)
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
		fieldType, _ := field.Element.Extra.Mores[constants.FieldRawType].(string)
		paramType := strings.TrimSpace(strings.Split(fieldType, "<")[0])
		paramTypes = append(paramTypes, paramType)
		paramNames = append(paramNames, field.Element.Name)
		paramStrings = append(paramStrings, paramType+" "+field.Element.Name)
	}

	paramsStr := "(" + strings.Join(paramTypes, ",") + ")"
	if c.checkMethodExists(elem.QualifiedName, elem.Name, paramsStr, fCtx) {
		return
	}

	consQN := c.resolver.BuildQualifiedName(elem.QualifiedName, elem.Name+paramsStr)

	addDef(fCtx, &model.CodeElement{
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
				constants.MethodIsConstructor:     true,
				constants.MethodIsLombokGenerated: true,
				constants.LombokAnnotationType:    "AllArgsConstructor",
				constants.MethodParameters:        paramStrings,
			},
		},
		IsFormSugar: true,
	}, elem.QualifiedName, node)
}

// lombokRequiredArgsConstructor 处理 @RequiredArgsConstructor：为 final 字段生成构造器。
func (c *DeSugar) lombokRequiredArgsConstructor(elem *model.CodeElement, node coreast.Node, fCtx *core.FileContext) {
	fields := c.extractClassFields(elem, fCtx)

	var finalFields []*core.DefinitionEntry
	for _, field := range fields {
		isStatic, _ := field.Element.Extra.Mores[constants.FieldIsStatic].(bool)
		isFinal, _ := field.Element.Extra.Mores[constants.FieldIsFinal].(bool)
		// Lombok 语义：@RequiredArgsConstructor 只收「未初始化」的 final 字段（及 @NonNull 字段）。
		// 带内联初始化器的 final 字段（如 private final Executor e = getBean(...)）不进入构造器。
		if !isStatic && isFinal && !fieldHasInitializer(field, fCtx) {
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
		fieldType, _ := field.Element.Extra.Mores[constants.FieldRawType].(string)
		paramType := strings.TrimSpace(strings.Split(fieldType, "<")[0])
		paramTypes = append(paramTypes, paramType)
		paramNames = append(paramNames, field.Element.Name)
		paramStrings = append(paramStrings, paramType+" "+field.Element.Name)
	}

	paramsStr := "(" + strings.Join(paramTypes, ",") + ")"
	if c.checkMethodExists(elem.QualifiedName, elem.Name, paramsStr, fCtx) {
		return
	}

	consQN := c.resolver.BuildQualifiedName(elem.QualifiedName, elem.Name+paramsStr)

	addDef(fCtx, &model.CodeElement{
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
				constants.MethodIsConstructor:     true,
				constants.MethodIsLombokGenerated: true,
				constants.LombokAnnotationType:    "RequiredArgsConstructor",
				constants.MethodParameters:        paramStrings,
			},
		},
		IsFormSugar: true,
	}, elem.QualifiedName, node)
}

// lombokData 处理 @Data：Getter+Setter+ToString+EqualsAndHashCode+NoArgsConstructor。
func (c *DeSugar) lombokData(elem *model.CodeElement, node coreast.Node, fCtx *core.FileContext) {
	c.lombokGetter(elem, node, fCtx)
	c.lombokSetter(elem, node, fCtx)
	c.lombokToString(elem, node, fCtx)
	c.lombokEqualsAndHashCode(elem, node, fCtx)
	c.lombokNoArgsConstructor(elem, node, fCtx)
}

// lombokValue 处理 @Value：Getter+AllArgsConstructor+ToString+EqualsAndHashCode。
func (c *DeSugar) lombokValue(elem *model.CodeElement, node coreast.Node, fCtx *core.FileContext) {
	c.lombokGetter(elem, node, fCtx)
	c.lombokAllArgsConstructor(elem, node, fCtx)
	c.lombokToString(elem, node, fCtx)
	c.lombokEqualsAndHashCode(elem, node, fCtx)
}

// lombokBuilder 处理 @Builder：生成 Builder 内部类、builder()/链式 setter/build() 及私有构造器。
func (c *DeSugar) lombokBuilder(elem *model.CodeElement, node coreast.Node, fCtx *core.FileContext) {
	fields := c.extractClassFields(elem, fCtx)

	var nonStaticFields []*core.DefinitionEntry
	for _, field := range fields {
		isStatic, _ := field.Element.Extra.Mores[constants.FieldIsStatic].(bool)
		if !isStatic {
			nonStaticFields = append(nonStaticFields, field)
		}
	}

	builderClassName := "Builder"
	builderQN := c.resolver.BuildQualifiedName(elem.QualifiedName, builderClassName)

	if _, ok := fCtx.FindByQualifiedName(builderQN); !ok {
		addDef(fCtx, &model.CodeElement{
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
					constants.ClassIsStatic:           true,
					constants.MethodIsLombokGenerated: true,
					constants.LombokAnnotationType:    "Builder",
				},
			},
			IsFormSugar: true,
		}, elem.QualifiedName, node)
	}

	builderMethodName := "builder"
	builderMethodQN := c.resolver.BuildQualifiedName(elem.QualifiedName, builderMethodName+"()")
	if !c.checkMethodExists(elem.QualifiedName, builderMethodName, "()", fCtx) {
		addDef(fCtx, &model.CodeElement{
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
					constants.MethodIsConstructor:     false,
					constants.MethodIsLombokGenerated: true,
					constants.LombokAnnotationType:    "Builder",
					constants.MethodReturnType:        builderClassName,
				},
			},
			IsFormSugar: true,
		}, elem.QualifiedName, node)
	}

	for _, field := range nonStaticFields {
		fieldName := field.Element.Name
		fieldType, _ := field.Element.Extra.Mores[constants.FieldRawType].(string)

		setterQN := c.resolver.BuildQualifiedName(builderQN, fieldName+"("+fieldType+")")
		if _, ok := fCtx.FindByQualifiedName(setterQN); !ok {
			addDef(fCtx, &model.CodeElement{
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
						constants.MethodIsConstructor:     false,
						constants.MethodIsLombokGenerated: true,
						constants.LombokAnnotationType:    "Builder",
						constants.MethodReturnType:        builderClassName,
						constants.MethodParameters:        []string{fieldType + " " + fieldName},
					},
				},
				IsFormSugar: true,
			}, builderQN, node)
		}
	}

	buildQN := c.resolver.BuildQualifiedName(builderQN, "build()")
	if !c.checkMethodExists(builderQN, "build", "()", fCtx) {
		addDef(fCtx, &model.CodeElement{
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
					constants.MethodIsConstructor:     false,
					constants.MethodIsLombokGenerated: true,
					constants.LombokAnnotationType:    "Builder",
					constants.MethodReturnType:        elem.Name,
				},
			},
			IsFormSugar: true,
		}, builderQN, node)
	}

	builderParam := "builder"
	privateConstructorQN := c.resolver.BuildQualifiedName(elem.QualifiedName, fmt.Sprintf("%s(%s)", elem.Name, builderClassName))
	if !c.checkMethodExists(elem.QualifiedName, elem.Name, "("+builderClassName+")", fCtx) {
		addDef(fCtx, &model.CodeElement{
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
					constants.MethodIsConstructor:     true,
					constants.MethodIsLombokGenerated: true,
					constants.LombokAnnotationType:    "Builder",
					constants.MethodParameters:        []string{builderClassName + " " + builderParam},
				},
			},
			IsFormSugar: true,
		}, elem.QualifiedName, node)
	}
}

// =============================================================================
// 辅助方法
// =============================================================================

// fieldHasInitializer 判断字段是否有内联初始化器（如 private final Executor e = getBean(...)）。
// Lombok @RequiredArgsConstructor 语义：带初始化器的 final 字段不进入构造器参数。
// tree-sitter 中初始化器挂在 variable_declarator 的 value 字段上（field_declaration 本身无 value）。
func fieldHasInitializer(field *core.DefinitionEntry, fCtx *core.FileContext) bool {
	src := *fCtx.SourceBytes
	for i := 0; i < field.Node.NamedChildCount(); i++ {
		child := field.Node.NamedChild(i)
		if child.Kind() != "variable_declarator" {
			continue
		}
		nameNode := child.ChildByFieldName("name")
		if nameNode == nil || nameNode.Utf8Text(src) != field.Element.Name {
			continue
		}
		// 命中目标声明器：是否存在初始化器（= value）。
		return child.ChildByFieldName("value") != nil
	}
	return false
}

// extractClassFields 提取类中的所有字段（不含语法糖生成的字段）。
func (c *DeSugar) extractClassFields(elem *model.CodeElement, fCtx *core.FileContext) []*core.DefinitionEntry {
	var fields []*core.DefinitionEntry

	if defs, ok := fCtx.FindByKind(model.Field); ok {
		for _, entry := range defs {
			if entry.ParentQN == elem.QualifiedName && !entry.Element.IsFormSugar {
				fields = append(fields, entry)
			}
		}
	}

	return fields
}

// checkMethodExists 检查方法是否已存在（避免重复生成）。
func (c *DeSugar) checkMethodExists(parentQN, methodName, paramTypes string, fCtx *core.FileContext) bool {
	methodQN := c.resolver.BuildQualifiedName(parentQN, methodName+paramTypes)
	if _, ok := fCtx.FindByQualifiedName(methodQN); ok {
		return true
	}
	return false
}
