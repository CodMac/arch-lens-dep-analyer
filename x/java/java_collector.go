package java

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/x/java/enricher/ele"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/desugar"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Collector struct {
	resolver core.SymbolResolver
	builder  core.SymbolBuilder
	desugar  *desugar.DeSugar
}

func NewJavaCollector() *Collector {
	builder := NewSymbolBuilder()
	return &Collector{
		builder: builder,
		desugar: desugar.NewDeSugar(builder),
	}
}

// =============================================================================
// 1. 核心生命周期 (Core Workflow)
// =============================================================================

func (c *Collector) CollectDefinitions(rootNode *sitter.Node, filePath string, sourceBytes *[]byte) (*core.FileContext, error) {
	fCtx := core.NewFileContext(filePath, rootNode, sourceBytes)

	// Step 0: 注册文件自身节点并计算指标
	c.initFileElem(fCtx)

	// Step 1: 基础声明 (Package & Imports)
	c.processTopLevelDeclarations(fCtx)

	// Step 2: 递归收集定义 (Building Tree)
	nameOccurrence := make(map[string]int)
	c.collectBasicDefinitions(fCtx.RootNode, fCtx, fCtx.PackageName, nameOccurrence)

	// Step 3: 变量特殊作用域修正
	c.refineVariableScopes(fCtx)

	// Step 4: 元数据增强 (Metadata & Signatures)
	c.enrichMetadata(fCtx)

	// Step 5: 语法糖处理 (Records, Enums, Constructors, Lombok)
	c.applySyntacticSugar(fCtx)

	return fCtx, nil
}

func (c *Collector) initFileElem(fCtx *core.FileContext) {
	filePath := fCtx.FilePath

	fileElem := &model.CodeElement{
		Kind:          model.File,
		Name:          filepath.Base(filePath),
		QualifiedName: filePath,
		Path:          filePath,
		IsFormSource:  true,
	}

	fCtx.AddDefinition(fileElem, "", fCtx.RootNode)
}

func (c *Collector) processTopLevelDeclarations(fCtx *core.FileContext) {
	for i := 0; i < int(fCtx.RootNode.ChildCount()); i++ {
		child := fCtx.RootNode.Child(uint(i))
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
			c._handleImport(child, fCtx)
		}
	}
}

func (c *Collector) collectBasicDefinitions(node *sitter.Node, fCtx *core.FileContext, currentQN string, occurrences map[string]int) {
	if node.IsNamed() {
		if elems, kind := c._identifyElements(node, fCtx, currentQN); len(elems) > 0 {
			for _, elem := range elems {
				c._applyUniqueQN(elem, node, currentQN, occurrences, fCtx.SourceBytes)
				fCtx.AddDefinition(elem, currentQN, node)
			}

			// 如果是作用域容器（如类或方法），继续深入
			if helper.IsScopeContainer(kind) {
				childOccurrences := make(map[string]int)
				for i := 0; i < int(node.ChildCount()); i++ {
					c.collectBasicDefinitions(node.Child(uint(i)), fCtx, elems[0].QualifiedName, childOccurrences)
				}
				return
			}
		}
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		c.collectBasicDefinitions(node.Child(uint(i)), fCtx, currentQN, occurrences)
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
		containerNode := c._findNearestBlockParent(entry.Node)
		if containerNode == nil {
			continue
		}

		// 2. 遍历容器的子节点，寻找该变量逻辑上所属的 block 节点
		for i := 0; i < int(containerNode.ChildCount()); i++ {
			child := containerNode.Child(uint(i))
			// 只有当子节点是 block，且不是变量自身的定义节点时才处理
			if child.Kind() != "block" {
				continue
			}

			// 3. 在已采集的定义中，通过 Location 匹配找到对应的 block 实体
			for _, bDef := range blocks {
				if helper.MatchLocation(child, bDef.Element) {
					newParentQN := bDef.Element.QualifiedName

					// 4. 更新 ParentQN 并重新构建 QualifiedName
					entry.ParentQN = newParentQN
					entry.Element.QualifiedName = c.builder.BuildQualifiedName(newParentQN, entry.Element.Name)

					// 一旦找到匹配的 block 并完成重定位，即可跳出当前变量的查找
					goto nextVariable
				}
			}
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
	clazz, ok := fCtx.FindByElementKind(model.Class)
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

	enums, ok := fCtx.FindByElementKind(model.Enum)
	if ok {
		for _, entry := range enums {
			c.desugar.DesugarEnumMethods(entry.Element, entry.Node, fCtx)
		}
	}
}

// =============================================================================
// 2. 元素识别逻辑 (Element Identification)
// =============================================================================

func (c *Collector) _handleImport(node *sitter.Node, fCtx *core.FileContext) {
	isStatic, isWildcard := false, false
	var pathParts []string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
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

	fCtx.AddImport(alias, &core.ImportEntry{
		Kind: entryKind, Alias: alias, RawImportPath: fullPath, IsWildcard: isWildcard, IsStatic: isStatic, Location: helper.ExtractLocation(node, fCtx.FilePath),
	})
}

func (c *Collector) _identifyElements(node *sitter.Node, fCtx *core.FileContext, parentQN string) ([]*model.CodeElement, model.ElementKind) {
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
		names = c._extractAllVariableNames(node, fCtx.SourceBytes)
	case "local_variable_declaration", "formal_parameter", "spread_parameter", "resource", "catch_formal_parameter":
		kind = model.Variable
		names = c._extractAllVariableNames(node, fCtx.SourceBytes)
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
		if k, n := c._identifyLambdaParameter(node, fCtx); k != "" {
			kind = k
			names = []string{n}
		}
	case "block":
		kind, names = c._identifyBlockType(node)
	case "object_creation_expression":
		if helper.FindNamedChildOfType(node, "class_body") != nil {
			kind = model.AnonymousClass
			names = []string{"anonymousClass"}
		}
	}

	if kind != "" && names == nil {
		names = []string{c._resolveMissingName(node, kind, parentQN, fCtx.SourceBytes)}
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

func (c *Collector) _identifyLambdaParameter(node *sitter.Node, fCtx *core.FileContext) (model.ElementKind, string) {
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

func (c *Collector) _identifyBlockType(node *sitter.Node) (model.ElementKind, []string) {
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

func (c *Collector) _applyUniqueQN(elem *model.CodeElement, node *sitter.Node, parentQN string, occurrences map[string]int, src *[]byte) {
	identity := elem.Name
	if elem.Kind == model.Method && (node.Kind() == "method_declaration" || node.Kind() == "constructor_declaration" || node.Kind() == "annotation_type_element_declaration") {
		identity += c._extractParameterTypesOnly(node, src)
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
	elem.QualifiedName = c.builder.BuildQualifiedName(parentQN, identity)
}

func (c *Collector) _findNearestBlockParent(node *sitter.Node) *sitter.Node {
	// for(String s : list)
	if node.Kind() == "enhanced_for_statement" {
		return node
	}

	// 往上查找
	curr := node.Parent()
	for curr != nil {
		k := curr.Kind()
		if k == "for_statement" || k == "try_with_resources_statement" || k == "catch_clause" || k == "if_statement" {
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

func (c *Collector) _extractTypeString(node *sitter.Node, src *[]byte) string {
	if node.Kind() == "identifier" {
		return "inferred"
	}
	if tNode := node.ChildByFieldName("type"); tNode != nil {
		return helper.GetNodeContent(tNode, *src)
	}
	if node.Kind() == "spread_parameter" {
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(uint(i))
			if strings.Contains(child.Kind(), "type") {
				return helper.GetNodeContent(child, *src) + "..."
			}
		}
	}
	return "unknown"
}

func (c *Collector) _extractParameterTypesOnly(node *sitter.Node, src *[]byte) string {
	pNode := node.ChildByFieldName("parameters")
	if pNode == nil {
		return "()"
	}
	var types []string
	for i := 0; i < int(pNode.NamedChildCount()); i++ {
		tStr := strings.Split(c._extractTypeString(pNode.NamedChild(uint(i)), src), "<")[0]
		types = append(types, strings.TrimSpace(tStr))
	}
	return "(" + strings.Join(types, ",") + ")"
}

func (c *Collector) _extractAllVariableNames(node *sitter.Node, src *[]byte) []string {
	var names []string
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(uint(i))
		if child.Kind() == "variable_declarator" {
			if nNode := child.ChildByFieldName("name"); nNode != nil {
				names = append(names, helper.GetNodeContent(nNode, *src))
			}
		}
	}
	return names
}

func (c *Collector) _resolveMissingName(node *sitter.Node, kind model.ElementKind, parentQN string, src *[]byte) string {
	if nNode := node.ChildByFieldName("name"); nNode != nil {
		return helper.GetNodeContent(nNode, *src)
	}
	if kind == model.Method {
		parts := strings.Split(parentQN, ".")
		return parts[len(parts)-1]
	}
	return ""
}
