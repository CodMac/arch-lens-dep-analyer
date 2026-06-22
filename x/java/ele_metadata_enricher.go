package java

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type EleMetadataEnricher struct {
	resolver core.SymbolResolver
	fCtx     *core.FileContext
}

func (c *EleMetadataEnricher) ProcessMetadataForEntry(entry *core.DefinitionEntry) {
	node, elem := entry.Node, entry.Element
	mods, annos := c.extractModifiersAndAnnotations(node, *c.fCtx.SourceBytes)
	elem.Doc, elem.Comment = c.extractComments(node, c.fCtx.SourceBytes)

	extra := &model.Extra{
		Modifiers: mods, Annotations: annos, Mores: make(map[string]interface{}),
	}
	isStatic, isFinal := Contains(mods, "static"), Contains(mods, "final")

	switch elem.Kind {
	case model.File:
		c.fillFileMetadata(elem, extra)
	case model.Class, model.Interface, model.KAnnotation:
		c.fillTypeMetadata(elem, node, extra, mods, isFinal)
	case model.Method:
		if node.Kind() == "annotation_type_element_declaration" {
			c.fillAnnotationMember(elem, node, extra)
		} else {
			c.fillMethodMetadata(elem, node, extra, mods)
		}
	case model.Field:
		c.fillFieldMetadata(elem, node, extra, mods, isStatic, isFinal)
	case model.Variable:
		c.fillLocalVariableMetadata(elem, node, extra, mods, isFinal)
	case model.EnumConstant:
		c.fillEnumConstantMetadata(elem, node, extra)
	case model.Lambda:
		c.fillLambdaMetadata(elem, node, extra)
	case model.MethodRef:
		// 1. 设置原始签名 (如 System.out::println)
		elem.Signature = GetNodeContent(node, *c.fCtx.SourceBytes)

		// 2. 深度解析被引用的目标
		c.fillMethodReferenceDetails(elem, node, extra)
	case model.ScopeBlock:
		c.fillScopeBlockMetadata(elem, node, extra)
	case model.AnonymousClass:
		c.fillAnonymousClassMetadata(elem, node, extra)
	}
	elem.Extra = extra
}

// =============================================================================
// 原始信息提取
// =============================================================================

func (c *EleMetadataEnricher) extractTypeString(node *sitter.Node, src *[]byte) string {
	if node.Kind() == "identifier" {
		return "inferred"
	}
	if tNode := node.ChildByFieldName("type"); tNode != nil {
		return GetNodeContent(tNode, *src)
	}
	if node.Kind() == "spread_parameter" {
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(uint(i))
			if strings.Contains(child.Kind(), "type") {
				return GetNodeContent(child, *src) + "..."
			}
		}
	}
	return "unknown"
}

func (c *EleMetadataEnricher) extractThrows(node *sitter.Node, src *[]byte) []string {
	tNode := FindNamedChildOfType(node, "throws")
	if tNode == nil {
		return nil
	}
	var types []string
	for i := 0; i < int(tNode.NamedChildCount()); i++ {
		child := tNode.NamedChild(uint(i))
		if child.IsNamed() && child.Kind() != "throws" {
			types = append(types, GetNodeContent(child, *src))
		}
	}
	return types
}

func (c *EleMetadataEnricher) extractModifiersAndAnnotations(n *sitter.Node, src []byte) ([]string, []string) {
	var mods = make([]string, 0)
	var annos = make([]string, 0)
	if mNode := FindNamedChildOfType(n, "modifiers"); mNode != nil {
		for i := 0; i < int(mNode.ChildCount()); i++ {
			child := mNode.Child(uint(i))
			txt := GetNodeContent(child, src)
			if strings.Contains(child.Kind(), "annotation") {
				annos = append(annos, txt)
			} else if txt != "" {
				mods = append(mods, txt)
			}
		}
	}
	return mods, annos
}

func (c *EleMetadataEnricher) extractComments(node *sitter.Node, src *[]byte) (doc, comment string) {
	curr := node
	if node.Kind() == "variable_declarator" && node.Parent() != nil {
		curr = node.Parent()
	}
	prev := curr.PrevSibling()
	for prev != nil {
		if prev.Kind() == "block_comment" || prev.Kind() == "line_comment" {
			text := GetNodeContent(prev, *src)
			if strings.HasPrefix(text, "/**") {
				doc = text
			} else {
				comment = text
			}
			break
		}
		if strings.TrimSpace(GetNodeContent(prev, *src)) != "" {
			break
		}
		prev = prev.PrevSibling()
	}
	return
}

func (c *EleMetadataEnricher) extractParameterList(node *sitter.Node, src *[]byte) []string {
	pNode := node.ChildByFieldName("parameters")
	if pNode == nil {
		return nil
	}
	var params []string
	for i := 0; i < int(pNode.NamedChildCount()); i++ {
		params = append(params, GetNodeContent(pNode.NamedChild(uint(i)), *src))
	}
	return params
}

func (c *EleMetadataEnricher) extractParameterWithNames(node *sitter.Node, src *[]byte) string {
	if pNode := node.ChildByFieldName("parameters"); pNode != nil {
		return GetNodeContent(pNode, *src)
	}
	return "()"
}

func (c *EleMetadataEnricher) extractInterfaceListFromNode(node *sitter.Node, src *[]byte) []string {
	var results []string
	target := node
	if node.Kind() != "type_list" {
		if listNode := FindNamedChildOfType(node, "type_list"); listNode != nil {
			target = listNode
		}
	}
	for i := 0; i < int(target.NamedChildCount()); i++ {
		child := target.NamedChild(uint(i))
		if strings.Contains(child.Kind(), "type") || child.Kind() == "type_identifier" {
			results = append(results, GetNodeContent(child, *src))
		}
	}
	return results
}

// =============================================================================
// metrics指标计算
// =============================================================================

func (c *EleMetadataEnricher) calculateComplexity(node *sitter.Node) int {
	complexity := 1
	var traverse func(*sitter.Node)
	traverse = func(n *sitter.Node) {
		if n == nil {
			return
		}
		kind := n.Kind()
		switch kind {
		case "if_statement", "for_statement", "while_statement", "do_statement",
			"catch_clause", "conditional_expression", "ternary_expression", "switch_label":
			complexity++
		case "binary_expression":
			// 对于 binary_expression，我们需要检查运算符是否为 && 或 ||
			// 注意：tree-sitter-java 可能将 && 和 || 视为不同的子节点
			for i := 0; i < int(n.ChildCount()); i++ {
				child := n.Child(uint(i))
				if child.Kind() == "&&" || child.Kind() == "||" {
					complexity++
				}
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			traverse(n.Child(uint(i)))
		}
	}
	traverse(node)
	return complexity
}

func (c *EleMetadataEnricher) calculateLOC(source []byte) int {
	// 1. 移除块注释
	reBlock := regexp.MustCompile(`(?s)/\*.*?\*/`)
	content := reBlock.ReplaceAll(source, []byte(""))

	// 2. 按行切分并统计
	lines := strings.Split(string(content), "\n")
	loc := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 排除空行和单行注释
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
			loc++
		}
	}
	return loc
}

func (c *EleMetadataEnricher) calculateRawLOC(source []byte) int {
	// 按行切分并统计
	lines := strings.Split(string(source), "\n")

	return len(lines)
}

// =============================================================================
// 元数据填充
// =============================================================================

func (c *EleMetadataEnricher) fillFileMetadata(elem *model.CodeElement, extra *model.Extra) {
	extra.Mores[constants.FileRawLOC] = c.calculateRawLOC(*c.fCtx.SourceBytes)
	extra.Mores[constants.FileLOC] = c.calculateLOC(*c.fCtx.SourceBytes)
}

func (c *EleMetadataEnricher) fillTypeMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra, mods []string, isFinal bool) {
	extra.Mores[constants.ClassIsAbstract], extra.Mores[constants.ClassIsFinal] = Contains(mods, "abstract"), isFinal
	extra.Mores[constants.ClassIsStatic] = Contains(mods, "static")

	typeParams := ""
	if tpNode := node.ChildByFieldName("type_parameters"); tpNode != nil {
		typeParams = GetNodeContent(tpNode, *c.fCtx.SourceBytes)
	}

	heritage := ""
	if super := node.ChildByFieldName("superclass"); super != nil {
		content := GetNodeContent(super, *c.fCtx.SourceBytes)
		extra.Mores[constants.ClassSuperClass] = strings.TrimSpace(strings.TrimPrefix(content, "extends"))
		heritage += " " + content
	}

	ifacesNode := c.findInterfacesNode(node)
	if ifacesNode != nil {
		if ifaces := c.extractInterfaceListFromNode(ifacesNode, c.fCtx.SourceBytes); len(ifaces) > 0 {
			mKey := constants.InterfaceImplementedInterfaces
			if elem.Kind == model.Class {
				mKey = constants.ClassImplementedInterfaces
			}
			extra.Mores[mKey] = ifaces
			heritage += " " + GetNodeContent(ifacesNode, *c.fCtx.SourceBytes)
		}
	}

	displayKind := strings.Replace(node.Kind(), "_declaration", "", 1)
	elem.Signature = strings.TrimSpace(fmt.Sprintf("%s %s %s%s%s",
		strings.Join(mods, " "), displayKind, elem.Name, typeParams, heritage))
}

func (c *EleMetadataEnricher) fillMethodMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra, mods []string) {
	extra.Mores[constants.MethodIsConstructor] = (node.Kind() == "constructor_declaration")
	extra.Mores[constants.MethodComplexity] = c.calculateComplexity(node)

	typeParams := ""
	if tpNode := node.ChildByFieldName("type_parameters"); tpNode != nil {
		typeParams = GetNodeContent(tpNode, *c.fCtx.SourceBytes) + " "
	}

	retType := ""
	if tNode := node.ChildByFieldName("type"); tNode != nil {
		retType = GetNodeContent(tNode, *c.fCtx.SourceBytes)
		extra.Mores[constants.MethodReturnType] = retType
	}

	paramsRaw := c.extractParameterWithNames(node, c.fCtx.SourceBytes)
	if params := c.extractParameterList(node, c.fCtx.SourceBytes); len(params) > 0 {
		extra.Mores[constants.MethodParameters] = params
	}

	throwsList := c.extractThrows(node, c.fCtx.SourceBytes)
	throwsStr := ""
	if len(throwsList) > 0 {
		extra.Mores[constants.MethodThrowsTypes] = throwsList
		throwsStr = " throws " + strings.Join(throwsList, ", ")
	}

	elem.Signature = strings.TrimSpace(fmt.Sprintf("%s %s%s %s%s%s",
		strings.Join(mods, " "), typeParams, retType, elem.Name, paramsRaw, throwsStr))
}

func (c *EleMetadataEnricher) fillFieldMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra, mods []string, isStatic, isFinal bool) {
	vType := c.extractTypeString(node, c.fCtx.SourceBytes)
	extra.Mores[constants.FieldRawType], extra.Mores[constants.FieldIsStatic], extra.Mores[constants.FieldIsFinal] = vType, isStatic, isFinal
	extra.Mores[constants.FieldIsConstant] = isStatic && isFinal
	elem.Signature = strings.TrimSpace(fmt.Sprintf("%s %s %s", strings.Join(mods, " "), vType, elem.Name))
}

func (c *EleMetadataEnricher) fillLocalVariableMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra, mods []string, isFinal bool) {
	vType := c.extractTypeString(node, c.fCtx.SourceBytes)
	extra.Mores[constants.VariableRawType], extra.Mores[constants.VariableIsFinal] = vType, isFinal
	extra.Mores[constants.VariableIsParam] = (node.Kind() == "formal_parameter" || node.Kind() == "spread_parameter")
	elem.Signature = strings.TrimSpace(fmt.Sprintf("%s %s %s", strings.Join(mods, " "), vType, elem.Name))
}

func (c *EleMetadataEnricher) fillEnumConstantMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra) {
	elem.Signature = elem.Name

	if argList := FindNamedChildOfType(node, "argument_list"); argList != nil {
		var args []string
		for i := 0; i < int(argList.NamedChildCount()); i++ {
			args = append(args, GetNodeContent(argList.NamedChild(uint(i)), *c.fCtx.SourceBytes))
		}
		extra.Mores[constants.EnumArguments] = args
	}
}

func (c *EleMetadataEnricher) fillAnnotationMember(elem *model.CodeElement, node *sitter.Node, extra *model.Extra) {
	extra.Mores[constants.MethodIsAnnotation] = true
	if valNode := node.ChildByFieldName("value"); valNode != nil {
		extra.Mores[constants.MethodDefaultValue] = GetNodeContent(valNode, *c.fCtx.SourceBytes)
	}
	vType := GetNodeContent(node.ChildByFieldName("type"), *c.fCtx.SourceBytes)
	elem.Signature = fmt.Sprintf("%s %s()", vType, elem.Name)
}

func (c *EleMetadataEnricher) fillScopeBlockMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra) {
	isStatic := (node.Kind() == "static_initializer")
	extra.Mores[constants.BlockIsStatic] = isStatic
	extra.Mores[constants.MethodComplexity] = c.calculateComplexity(node)
	elem.Signature = "{...}"
	if isStatic {
		elem.Signature = "static {...}"
	}
}

func (c *EleMetadataEnricher) fillMethodReferenceDetails(elem *model.CodeElement, node *sitter.Node, extra *model.Extra) {
	var receiver, target string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		kind := child.Kind()

		// 1. 忽略不需要的符号和中间件
		if kind == "::" || kind == "type_arguments" {
			if kind == "type_arguments" {
				extra.Mores[constants.MethodRefTypeArgs] = GetNodeContent(child, *c.fCtx.SourceBytes)
			}
			continue
		}

		// 2. 识别内容
		content := GetNodeContent(child, *c.fCtx.SourceBytes)
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
}

func (c *EleMetadataEnricher) fillLambdaMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra) {
	// 1. 提取参数部分
	// Lambda 参数可能是: (a, b) -> ... 或 a -> ... 或 (int a) -> ...
	var paramsStr string
	paramNode := node.ChildByFieldName("parameters")
	if paramNode != nil {
		paramsStr = GetNodeContent(paramNode, *c.fCtx.SourceBytes)
	} else {
		// 处理单参数没有括号的情况: s -> s.toLowerCase()
		// 在 tree-sitter-java 中，这种 identifier 会是 lambda_expression 的第一个命名子节点
		if firstChild := node.NamedChild(0); firstChild != nil && firstChild.Kind() == "identifier" {
			paramsStr = GetNodeContent(firstChild, *c.fCtx.SourceBytes)
		}
	}
	extra.Mores[constants.LambdaParameters] = paramsStr
	extra.Mores[constants.MethodComplexity] = c.calculateComplexity(node)

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

func (c *EleMetadataEnricher) fillAnonymousClassMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra) {
	// 在 identifyElement 中, AnonymousClass 锚定的 node 是 "object_creation_expression"
	if node.Kind() != "object_creation_expression" {
		return
	}

	// 提取 new 关键字后的类型，例如 new Runnable() { ... } 中的 Runnable
	typeNode := node.ChildByFieldName("type")
	if typeNode != nil {
		typeName := GetNodeContent(typeNode, *c.fCtx.SourceBytes)
		extra.Mores[constants.AnonymousClassType] = typeName
		elem.Signature = "anonymous extends/implements " + typeName
	}
}

// =============================================================================
// 辅助方法
// =============================================================================

func (c *EleMetadataEnricher) findInterfacesNode(node *sitter.Node) *sitter.Node {
	if n := node.ChildByFieldName("interfaces"); n != nil {
		return n
	}
	if n := node.ChildByFieldName("extends"); n != nil {
		return n
	}
	return FindNamedChildOfType(node, "extends_interfaces")
}
