// Package collector 实现 Java 语言 core.Collector 接口：
// 将单个源文件解析为 FileContext（含全部代码元素定义与索引）。
package collector

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/collector/desugar"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/enricher/ele"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

// Collector 实现 core.Collector，按五步流程收集符号定义：
// initFileElem -> processTopLevelDeclarations -> collectBasicDefinitions
// -> refineVariableScopes -> enrichMetadata -> applySyntacticSugar。
type Collector struct {
	resolver core.SymbolResolver
	desugar  *collectordesugar.DeSugar
}

// New 创建 Java Collector。
func New(resolver core.SymbolResolver) *Collector {
	return &Collector{
		resolver: resolver,
		desugar:  collectordesugar.NewDeSugar(resolver),
	}
}

// =============================================================================
// core.Collector 接口实现
// =============================================================================

// CollectDefinitions 从源文件收集符号定义并产出 FileContext。
func (c *Collector) CollectDefinitions(src coreast.SourceFile) (*core.FileContext, error) {
	if src == nil || src.Root() == nil {
		return nil, fmt.Errorf("empty source file")
	}
	root := src.Root()
	content := src.Content()
	path := src.Path()

	fCtx := core.NewFileContext(path, "", root, &content)

	// Step 0: 注册文件自身节点并计算指标
	c.initFileElem(fCtx)

	// Step 1: 基础声明 (Package & Imports)
	c.processTopLevelDeclarations(fCtx)

	// Step 2: 递归收集定义 (Building Tree)
	nameOccurrence := make(map[string]int)
	c.collectBasicDefinitions(root, fCtx, fCtx.PackageName, nameOccurrence)

	// Step 3: 变量特殊作用域修正
	c.refineVariableScopes(fCtx)

	// Step 4: 元数据增强 (Metadata & Signatures)
	c.enrichMetadata(fCtx)

	// Step 5: 语法糖处理 (Records, Enums, Constructors, Lombok)
	c.applySyntacticSugar(fCtx)

	return fCtx, nil
}

// =============================================================================
// 核心生命周期 (Core Workflow)
// =============================================================================

func (c *Collector) initFileElem(fCtx *core.FileContext) {
	filePath := fCtx.FilePath

	fileElem := &model.CodeElement{
		Kind:          model.File,
		Name:          filepath.Base(filePath),
		QualifiedName: filePath,
		Path:          filePath,
		IsFormSource:  true,
	}

	fCtx.AddDefinition(&core.DefinitionEntry{Element: fileElem, ParentQN: "", Node: fCtx.Root})
}

func (c *Collector) processTopLevelDeclarations(fCtx *core.FileContext) {
	root := fCtx.Root
	for i := 0; i < root.ChildCount(); i++ {
		child := root.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "package_declaration":
			if ident := helper.FindNamedChildOfType(child, "scoped_identifier"); ident != nil {
				fCtx.PackageName = helper.GetNodeContent(ident, *fCtx.SourceBytes)
			} else if nameNode := child.ChildByFieldName("name"); nameNode != nil {
				fCtx.PackageName = helper.GetNodeContent(nameNode, *fCtx.SourceBytes)
			}
		case "import_declaration":
			c.handleImport(child, fCtx)
		}
	}
}

func (c *Collector) collectBasicDefinitions(node coreast.Node, fCtx *core.FileContext, currentQN string, occurrences map[string]int) {
	if node.IsNamed() {
		if elems, kind := c.identifyElements(node, fCtx, currentQN); len(elems) > 0 {
			for _, elem := range elems {
				c.applyUniqueQN(elem, node, currentQN, occurrences, fCtx.SourceBytes)
				fCtx.AddDefinition(&core.DefinitionEntry{Element: elem, ParentQN: currentQN, Node: node})
			}

			// 如果是作用域容器（如类或方法），继续深入
			if helper.IsScopeContainer(kind) {
				childOccurrences := make(map[string]int)
				for i := 0; i < node.ChildCount(); i++ {
					c.collectBasicDefinitions(node.Child(i), fCtx, elems[0].QualifiedName, childOccurrences)
				}
				return
			}
		}
	}
	for i := 0; i < node.ChildCount(); i++ {
		c.collectBasicDefinitions(node.Child(i), fCtx, currentQN, occurrences)
	}
}

func (c *Collector) refineVariableScopes(fCtx *core.FileContext) {
	// 获取所有已注册的 block，用于后续比对
	blocks, _ := fCtx.FindByShortName("block")
	if len(blocks) == 0 {
		return
	}

	for _, entry := range fCtx.Definitions {
		// 仅针对变量进行作用域修正
		if entry.Element.Kind != model.Variable {
			continue
		}

		// 1. 向上寻找最近的逻辑容器 (try/for/if/catch)
		containerNode := c.findNearestBlockParent(entry.Node)
		if containerNode == nil {
			continue
		}

		// 2. 遍历容器的子节点，寻找该变量逻辑上所属的 block 节点。
		// tree-sitter 中 if/else 是同一 if_statement 下的两个 block 子节点（else 无独立节点），
		// 因此不能固定匹配第一个 block（否则 else 块变量被误挂到 if 体块）。
		// 策略：优先选「包含变量声明行」的 block（最内层）；若无 block 包含变量行
		// （try-with-resources 资源变量/catch 参数/for 头变量/instanceof 模式变量，声明在块外），
		// 回退到第一个 block 子节点，保持原有行为不回归。
		varRow := -1
		if entry.Element.Location != nil {
			varRow = entry.Element.Location.StartLine
		}
		var fallback *core.DefinitionEntry
		for i := 0; i < containerNode.ChildCount(); i++ {
			child := containerNode.Child(i)
			// 只有当子节点是 block，且不是变量自身的定义节点时才处理
			if child.Kind() != "block" {
				continue
			}

			// 3. 在已采集的定义中，通过 Location 匹配找到对应的 block 实体
			for _, bDef := range blocks {
				if helper.MatchLocation(child, bDef.Element) {
					if fallback == nil {
						fallback = bDef
					}
					if varRow >= 0 {
						sp, ep := child.StartPoint(), child.EndPoint()
						if varRow >= int(sp.Row)+1 && varRow <= int(ep.Row)+1 {
							// 4. 变量声明行落在该 block 内：优先重定位到此 block
							entry.ParentQN = bDef.Element.QualifiedName
							entry.Element.QualifiedName = c.resolver.BuildQualifiedName(bDef.Element.QualifiedName, entry.Element.Name)
							goto nextVariable
						}
					}
				}
			}
		}

		// 5. 无 block 包含变量声明行：回退到第一个 block（try-with-resources/catch/for 头/instanceof 模式变量）
		if fallback != nil {
			entry.ParentQN = fallback.Element.QualifiedName
			entry.Element.QualifiedName = c.resolver.BuildQualifiedName(fallback.Element.QualifiedName, entry.Element.Name)
		}
	nextVariable:
	}
}

func (c *Collector) enrichMetadata(fCtx *core.FileContext) {
	enricher := ele.NewEnricher(fCtx)

	for _, entry := range fCtx.Definitions {
		enricher.EnrichMetadata(entry)
	}
}

func (c *Collector) applySyntacticSugar(fCtx *core.FileContext) {
	clazz, ok := fCtx.FindByKind(model.Class)
	if ok {
		for _, entry := range clazz {
			elem, node := entry.Element, entry.Node
			if node.Kind() == "record_declaration" {
				c.desugar.DesugarRecordMembers(elem, node, fCtx)
			} else if node.Kind() == "class_declaration" {
				c.desugar.DesugarDefaultConstructor(elem, node, fCtx)
				c.desugar.DesugarLombok(elem, node, fCtx)
			}
		}
	}

	enums, ok := fCtx.FindByKind(model.Enum)
	if ok {
		for _, entry := range enums {
			c.desugar.DesugarEnumMethods(entry.Element, entry.Node, fCtx)
		}
	}
}

// =============================================================================
// 元素识别逻辑 (Element Identification)
// =============================================================================

func (c *Collector) handleImport(node coreast.Node, fCtx *core.FileContext) {
	isStatic, isWildcard := false, false
	var pathParts []string
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "static":
			isStatic = true
		case "scoped_identifier", "identifier", "asterisk":
			pathParts = append(pathParts, helper.GetNodeContent(child, *fCtx.SourceBytes))
			if child.Kind() == "asterisk" {
				isWildcard = true
			}
		}
	}
	if len(pathParts) == 0 {
		return
	}
	fullPath := strings.Join(pathParts, ".")
	parts := strings.Split(fullPath, ".")
	alias := parts[len(parts)-1]

	entryKind := model.Class
	if isStatic {
		entryKind = model.Constant
	} else if isWildcard {
		entryKind = model.Package
	}

	fCtx.AddImport(alias, &core.ImportInfo{
		Kind:          entryKind,
		Alias:         alias,
		RawImportPath: fullPath,
		IsWildcard:    isWildcard,
		IsStatic:      isStatic,
		Location:      helper.ExtractLocation(node, fCtx.FilePath),
	})
}

func (c *Collector) identifyElements(node coreast.Node, fCtx *core.FileContext, parentQN string) ([]*model.CodeElement, model.ElementKind) {
	var kind model.ElementKind
	var names []string
	kindStr := node.Kind()

	switch kindStr {
	case "class_declaration", "record_declaration":
		kind = model.Class
	case "interface_declaration":
		kind = model.Interface
	case "enum_declaration":
		kind = model.Enum
	case "enum_constant":
		kind = model.EnumConstant
	case "annotation_type_declaration":
		kind = model.KAnnotation
	case "annotation_type_element_declaration", "method_declaration", "constructor_declaration":
		kind = model.Method
	case "field_declaration":
		kind = model.Field
		names = c.extractAllVariableNames(node, fCtx.SourceBytes)
	case "local_variable_declaration", "formal_parameter", "spread_parameter", "resource", "catch_formal_parameter":
		kind = model.Variable
		names = c.extractAllVariableNames(node, fCtx.SourceBytes)
	case "enhanced_for_statement", "instanceof_expression":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			kind = model.Variable
			names = []string{helper.GetNodeContent(nameNode, *fCtx.SourceBytes)}
		}
	case "lambda_expression":
		kind = model.Lambda
		names = []string{"lambda"}
	case "method_reference":
		kind = model.MethodRef
		names = []string{"method_ref"}
	case "static_initializer":
		kind = model.ScopeBlock
		names = []string{"$static"}
	case "identifier":
		if k, n := c.identifyLambdaParameter(node, fCtx); k != "" {
			kind = k
			names = []string{n}
		}
	case "block":
		kind, names = c.identifyBlockType(node)
	case "object_creation_expression":
		if helper.FindNamedChildOfType(node, "class_body") != nil {
			kind = model.AnonymousClass
			names = []string{"anonymousClass"}
		}
	}

	if kind != "" && names == nil {
		names = []string{c.resolveMissingName(node, kind, parentQN, fCtx.SourceBytes)}
	}
	if kind == "" || names == nil {
		return nil, ""
	}

	var elements []*model.CodeElement
	for _, name := range names {
		elements = append(elements, &model.CodeElement{
			Kind:         kind,
			Name:         name,
			Path:         fCtx.FilePath,
			Location:     helper.ExtractLocation(node, fCtx.FilePath),
			IsFormSource: true,
		})
	}
	return elements, kind
}

func (c *Collector) identifyLambdaParameter(node coreast.Node, fCtx *core.FileContext) (model.ElementKind, string) {
	parent := node.Parent()
	if parent == nil {
		return "", ""
	}

	pKind := parent.Kind()
	if pKind == "inferred_parameters" || pKind == "lambda_expression" {
		// 如果是单参数 Lambda (s -> ...)，确保 identifier 是参数位置而非 Body 位置
		if pKind == "lambda_expression" {
			firstChild := parent.NamedChild(0)
			if firstChild == nil || helper.GetNodeContent(firstChild, *fCtx.SourceBytes) != helper.GetNodeContent(node, *fCtx.SourceBytes) {
				return "", ""
			}
		}

		return model.Variable, helper.GetNodeContent(node, *fCtx.SourceBytes)
	}
	return "", ""
}

func (c *Collector) identifyBlockType(node coreast.Node) (model.ElementKind, []string) {
	parent := node.Parent()
	if parent == nil {
		return "", nil
	}

	pKind := parent.Kind()
	if pKind == "class_body" {
		return model.ScopeBlock, []string{"$instance"}
	}

	// 排除已经拥有作用域名称的块，防止 QN 冗余
	if pKind == "method_declaration" ||
		pKind == "constructor_declaration" ||
		pKind == "static_initializer" ||
		pKind == "lambda_expression" ||
		pKind == "method_reference" {
		return "", nil
	}

	return model.ScopeBlock, []string{"block"}
}

func (c *Collector) applyUniqueQN(elem *model.CodeElement, node coreast.Node, parentQN string, occurrences map[string]int, src *[]byte) {
	identity := elem.Name
	if elem.Kind == model.Method && (node.Kind() == "method_declaration" || node.Kind() == "constructor_declaration" || node.Kind() == "annotation_type_element_declaration") {
		identity += c.extractParameterTypesOnly(node, src)
	}

	if elem.Kind == model.AnonymousClass || elem.Kind == model.Lambda || elem.Kind == model.ScopeBlock || elem.Kind == model.MethodRef {
		occurrences[elem.Name]++
		identity = fmt.Sprintf("%s$%d", elem.Name, occurrences[elem.Name])
	} else {
		occurrences[identity]++
		if occurrences[identity] > 1 {
			identity = fmt.Sprintf("%s$%d", identity, occurrences[identity])
		}
	}
	elem.QualifiedName = c.resolver.BuildQualifiedName(parentQN, identity)
}

func (c *Collector) findNearestBlockParent(node coreast.Node) coreast.Node {
	// for(String s : list)
	if node.Kind() == "enhanced_for_statement" {
		return node
	}

	// 往上查找
	curr := node.Parent()
	for curr != nil {
		k := curr.Kind()
		if k == "for_statement" || k == "enhanced_for_statement" || k == "try_with_resources_statement" || k == "catch_clause" || k == "if_statement" {
			return curr
		}
		if k == "method_declaration" || k == "class_declaration" {
			break
		}
		curr = curr.Parent()
	}
	return nil
}

// =============================================================================
// 辅助工具逻辑 (Helper Utilities)
// =============================================================================

func (c *Collector) extractTypeString(node coreast.Node, src *[]byte) string {
	if node.Kind() == "identifier" {
		return "inferred"
	}
	if tNode := node.ChildByFieldName("type"); tNode != nil {
		return helper.GetNodeContent(tNode, *src)
	}
	if node.Kind() == "spread_parameter" {
		for i := 0; i < node.NamedChildCount(); i++ {
			child := node.NamedChild(i)
			if strings.Contains(child.Kind(), "type") {
				return helper.GetNodeContent(child, *src) + "..."
			}
		}
	}
	return "unknown"
}

func (c *Collector) extractParameterTypesOnly(node coreast.Node, src *[]byte) string {
	pNode := node.ChildByFieldName("parameters")
	if pNode == nil {
		return "()"
	}
	var types []string
	for i := 0; i < pNode.NamedChildCount(); i++ {
		tStr := strings.Split(c.extractTypeString(pNode.NamedChild(i), src), "<")[0]
		types = append(types, strings.TrimSpace(tStr))
	}
	return "(" + strings.Join(types, ",") + ")"
}

func (c *Collector) extractAllVariableNames(node coreast.Node, src *[]byte) []string {
	var names []string
	for i := 0; i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.Kind() == "variable_declarator" {
			if nNode := child.ChildByFieldName("name"); nNode != nil {
				names = append(names, helper.GetNodeContent(nNode, *src))
			}
		}
	}
	return names
}

func (c *Collector) resolveMissingName(node coreast.Node, kind model.ElementKind, parentQN string, src *[]byte) string {
	if nNode := node.ChildByFieldName("name"); nNode != nil {
		return helper.GetNodeContent(nNode, *src)
	}
	if kind == model.Method {
		parts := strings.Split(parentQN, ".")
		return parts[len(parts)-1]
	}
	return ""
}
