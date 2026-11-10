// Package collector 实现 C/C++ 语言 core.Collector 接口：
// 将单个源文件解析为 FileContext（含全部代码元素定义与索引）。
//
// C/C++ 与 Java 的关键差异：
//   - QN 分隔符为 ::（namespace/类/方法层级）
//   - 声明与定义分离（FUNCTION_DECL/METHOD_DECL 与 FUNCTION/METHOD 成对）
//   - 条件编译分支全保留，元素记 cpp.condition
//   - 匿名 struct 不产独立元素，字段挂承载实体
package collector

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/constants"
	cppctx "github.com/CodMac/arch-lens-dep-analyer/x/cpp/context"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/helper"
)

// Collector 实现 core.Collector。
type Collector struct {
	resolver core.SymbolResolver

	// condStacks 条件编译分支栈：文件路径 -> (条件节点起始字节 -> 条件标签)。
	// 引擎按文件并行调用 CollectDefinitions，故需加锁保护。
	mu         sync.Mutex
	condStacks map[string]map[int]string
}

// New 创建 C/C++ Collector。
func New(resolver core.SymbolResolver) *Collector {
	return &Collector{
		resolver:   resolver,
		condStacks: make(map[string]map[int]string),
	}
}

// identified 描述一个待注册的元素及其声明器节点（QN 计算用）。
type identified struct {
	elem     *model.CodeElement
	declNode coreast.Node // 声明器节点（函数/变量/字段），可为 nil
	sig      string       // 函数参数签名 "(int,char*)"，非函数为空
	isDecl   bool         // 是否为「声明」（非定义）：声明与定义共享同一 QN，不参与重复计数
	parentQN string       // 计算后的父级 QN（assignQN 填充）
}

// =============================================================================
// core.Collector 接口实现
// =============================================================================

// CollectDefinitions 从源文件收集符号定义并产出 FileContext。
func (c *Collector) CollectDefinitions(src coreast.SourceFile) (core.FileContext, error) {
	if src == nil || src.Root() == nil {
		return nil, fmt.Errorf("empty source file")
	}
	root := src.Root()
	content := src.Content()
	path := src.Path()

	fCtx := cppctx.NewFileContext(path, root, &content)

	// Step 0: 文件自身元素
	c.initFileElem(fCtx)

	// Step 1: 顶层声明（#include）
	c.processTopLevelDeclarations(fCtx)

	// Step 2: 递归收集定义
	nameOccurrence := make(map[string]int)
	c.collectBasicDefinitions(root, fCtx, "", "", nameOccurrence)

	// Step 3: 元数据增强
	c.enrichMetadata(fCtx)

	// Step 4: 汇总命名空间列表（C++ 多 namespace）
	c.collectNamespaces(fCtx)

	c.releaseCondStack(fCtx.FilePath())
	return fCtx, nil
}

// =============================================================================
// 核心生命周期
// =============================================================================

func (c *Collector) initFileElem(fCtx core.FileContext) {
	filePath := fCtx.FilePath()
	fileElem := &model.CodeElement{
		Kind:          model.File,
		Name:          filepath.Base(filePath),
		QualifiedName: filePath,
		Path:          filePath,
		IsFormSource:  true,
	}
	fCtx.AddDefinition(&core.DefinitionEntry{Element: fileElem, ParentQN: "", Node: fCtx.Root()})
}

// processTopLevelDeclarations 收集 #include 声明（可出现在条件编译分支内）。
func (c *Collector) processTopLevelDeclarations(fCtx core.FileContext) {
	c.walkIncludes(fCtx.Root(), fCtx)
}

func (c *Collector) walkIncludes(node coreast.Node, fCtx core.FileContext) {
	if node == nil {
		return
	}
	if node.Kind() == "preproc_include" {
		c.handleInclude(node, fCtx)
		return
	}
	for i := 0; i < node.ChildCount(); i++ {
		c.walkIncludes(node.Child(i), fCtx)
	}
}

// handleInclude 将 #include 登记为文件级导入。
// 项目内 "foo.h" → File 元素；系统 <stdio.h> → 外部符号（cpp.include.is_system=true）。
func (c *Collector) handleInclude(node coreast.Node, fCtx core.FileContext) {
	pathNode := node.ChildByFieldName("path")
	if pathNode == nil {
		return
	}
	raw := helper.GetNodeContent(pathNode, *fCtx.SourceBytes())
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "<")
	raw = strings.TrimSuffix(raw, ">")
	raw = strings.TrimPrefix(raw, "\"")
	raw = strings.TrimSuffix(raw, "\"")
	if raw == "" {
		return
	}
	isSystem := pathNode.Kind() == "system_lib_string"
	alias := raw
	if i := strings.LastIndexAny(raw, "/\\"); i >= 0 {
		alias = raw[i+1:]
	}
	info := &core.ImportInfo{
		Kind:          model.File,
		Alias:         alias,
		RawImportPath: raw,
		Location:      helper.ExtractLocation(node, fCtx.FilePath()),
	}
	info.SetAttr(constants.ImportAttrIsSystem, isSystem)
	fCtx.AddImport(alias, info)
}

// collectBasicDefinitions 递归收集定义。
// currentQN 为当前作用域 QN；tmplParams 为模板参数原文。
func (c *Collector) collectBasicDefinitions(node coreast.Node, fCtx core.FileContext, currentQN, tmplParams string, occurrences map[string]int) {
	if node == nil {
		return
	}
	src := *fCtx.SourceBytes()

	// 条件编译：进入分支时压栈（#elif/#else 作为 alternative 子节点自然内层优先）。
	if cond := conditionFor(node, src); cond != "" {
		c.pushCond(fCtx.FilePath(), node.StartByte(), cond)
		defer c.popCond(fCtx.FilePath(), node.StartByte())
	}

	// friend 声明（friend void f(); / friend class X;）不是成员定义/声明，
	// 不产元素；其内部 declaration 也不应挂到当前类下。
	if node.Kind() == "friend_declaration" {
		return
	}

	// 模板声明：记录参数文本，传递给内部类型/函数。
	// 模板参数列表只作为文本记录，其内部 parameter_declaration（如 `Small T`）
	// 不是变量元素，不能下钻收集。
	if node.Kind() == "template_declaration" {
		params := ""
		if p := node.ChildByFieldName("parameters"); p != nil {
			params = helper.GetNodeContent(p, src)
		}
		childOcc := make(map[string]int)
		for i := 0; i < node.ChildCount(); i++ {
			ch := node.Child(i)
			if ch.Kind() == "template_parameter_list" {
				continue
			}
			c.collectBasicDefinitions(ch, fCtx, currentQN, params, childOcc)
		}
		return
	}

	if node.IsNamed() {
		items, kind := c.identifyElements(node, fCtx, tmplParams)
		if len(items) > 0 {
			cond := c.activeCondition(fCtx.FilePath(), node)
			for i := range items {
				it := &items[i]
				c.assignQN(it, currentQN, occurrences, src)
				it.elem.Path = fCtx.FilePath()
				it.elem.Location = helper.ExtractLocation(node, fCtx.FilePath())
				if cond != "" {
					it.elem.GetMores()[constants.Condition] = cond
				}
				fCtx.AddDefinition(&core.DefinitionEntry{
					Element:       it.elem,
					ParentQN:      it.parentQN,
					IsDeclaration: it.isDecl,
					Node:          node,
				})
			}

			if helper.IsScopeContainer(kind) {
				scopeQN := items[0].elem.QualifiedName
				childOcc := make(map[string]int)
				for i := 0; i < node.ChildCount(); i++ {
					c.collectBasicDefinitions(node.Child(i), fCtx, scopeQN, tmplParams, childOcc)
				}
				return
			}

			// 匿名 struct/union 承载实体（决策 #2）：字段挂 typedef 名 / 变量名。
			// 多声明器（typedef struct {...} A, *B; / struct {...} s1, s2;）时每个承载实体各挂一份。
			if carriers := anonymousStructCarriers(node, items, src); len(carriers) > 0 {
				for _, carrier := range carriers {
					childOcc := make(map[string]int)
					for i := 0; i < node.ChildCount(); i++ {
						c.collectBasicDefinitions(node.Child(i), fCtx, carrier, tmplParams, childOcc)
					}
				}
				return
			}
		}
	}

	for i := 0; i < node.ChildCount(); i++ {
		c.collectBasicDefinitions(node.Child(i), fCtx, currentQN, tmplParams, occurrences)
	}
}

// anonymousStructCarriers 判定节点是否为「匿名 struct/union 的承载声明」，
// 若是则返回全部承载实体的 QN（typedef 名 / 变量名 / 字段名），供其成员字段挂载。
// 覆盖形态：typedef struct {...} Point; / struct {...} obj; / struct {...} inner;
// 多声明器形态（typedef struct {...} A, *B;）返回全部承载实体，字段各挂一份（允许冗余）。
func anonymousStructCarriers(node coreast.Node, items []identified, src []byte) []string {
	if len(items) == 0 {
		return nil
	}
	switch node.Kind() {
	case "type_definition", "declaration", "field_declaration":
	default:
		return nil
	}
	t := node.ChildByFieldName("type")
	if t == nil {
		return nil
	}
	if t.Kind() != "struct_specifier" && t.Kind() != "union_specifier" {
		return nil
	}
	if t.ChildByFieldName("name") != nil {
		return nil
	}
	// 只取承载实体（变量/字段/类型别名），函数声明器不是匿名类型的承载者。
	out := make([]string, 0, len(items))
	for _, it := range items {
		switch it.elem.Kind {
		case model.Variable, model.Field, model.Type:
			if it.elem.QualifiedName != "" {
				out = append(out, it.elem.QualifiedName)
			}
		}
	}
	return out
}

// resolveRelativeQN 将类外定义的相对限定名按当前命名空间补全前缀。
// 例：namespace app 内的 core::Base::helper -> app::core::Base::helper；
// 已含当前前缀（app::core::Base::helper）时不重复补全。
func resolveRelativeQN(full, currentQN string) string {
	if currentQN == "" {
		return full
	}
	if strings.HasPrefix(full, currentQN+helper.QNSep) {
		return full
	}
	return helper.BuildQN(currentQN, full)
}

// =============================================================================
// 条件编译栈
// =============================================================================

func (c *Collector) pushCond(path string, start int, cond string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, ok := c.condStacks[path]
	if !ok {
		s = make(map[int]string)
		c.condStacks[path] = s
	}
	s[start] = cond
}

func (c *Collector) popCond(path string, start int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if s, ok := c.condStacks[path]; ok {
		delete(s, start)
	}
}

func (c *Collector) releaseCondStack(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.condStacks, path)
}

// activeCondition 返回节点所在的最内层条件编译分支标签。
func (c *Collector) activeCondition(path string, node coreast.Node) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, ok := c.condStacks[path]
	if !ok {
		return ""
	}
	best := ""
	bestStart := -1
	for start, cond := range s {
		if start <= node.StartByte() && start > bestStart {
			bestStart, best = start, cond
		}
	}
	return best
}

// conditionFor 返回条件编译节点的标签（#ifdef X / #ifndef Y / #if C / #elif C / #else）。
func conditionFor(node coreast.Node, src []byte) string {
	switch node.Kind() {
	case "preproc_ifdef":
		kind := "#ifdef"
		if node.Child(0) != nil && node.Child(0).Kind() == "#ifndef" {
			kind = "#ifndef"
		}
		name := ""
		if n := node.ChildByFieldName("name"); n != nil {
			name = helper.GetNodeContent(n, src)
		}
		return kind + " " + name
	case "preproc_if":
		cond := ""
		if n := node.ChildByFieldName("condition"); n != nil {
			cond = helper.GetNodeContent(n, src)
		}
		return "#if " + cond
	case "preproc_elif":
		cond := ""
		if n := node.ChildByFieldName("condition"); n != nil {
			cond = helper.GetNodeContent(n, src)
		}
		return "#elif " + cond
	case "preproc_else":
		return "#else"
	}
	return ""
}

// =============================================================================
// 元素识别
// =============================================================================

func (c *Collector) identifyElements(node coreast.Node, fCtx core.FileContext, tmplParams string) ([]identified, model.ElementKind) {
	src := *fCtx.SourceBytes()
	switch node.Kind() {
	case "function_definition":
		decl := node.ChildByFieldName("declarator")
		name, _ := declaratorFullName(decl, src)
		kind := model.Function
		if isMethodContext(node, decl, src) {
			kind = model.Method
		}
		elem := newElem(kind, name, tmplParams)
		// `= default` / `= delete`：无函数体，但仍是类内定义（编译器生成/删除），
		// 记 is_declaration 便于区分，kind 保持定义语义。
		if node.ChildByFieldName("body") == nil {
			elem.GetMores()[constants.IsDeclaration] = true
		}
		return []identified{{elem: elem, declNode: decl, sig: paramSignature(decl, src)}}, kind

	case "declaration":
		return c.identifyDeclaration(node, fCtx, tmplParams)

	case "field_declaration":
		decl := node.ChildByFieldName("declarator")
		if isFunc, _ := classifyDeclarator(decl, src); isFunc {
			name, _ := declaratorFullName(decl, src)
			elem := newElem(model.MethodDecl, name, tmplParams)
			elem.GetMores()[constants.FunctionIsDeclaration] = true
			return []identified{{elem: elem, declNode: decl, sig: paramSignature(decl, src), isDecl: true}}, model.MethodDecl
		}
		var out []identified
		for _, d := range collectDeclarators(node) {
			name, _ := declaratorFullName(d, src)
			if name == "" {
				continue
			}
			out = append(out, identified{elem: newElem(model.Field, name, tmplParams), declNode: d})
		}
		if len(out) == 0 {
			return nil, ""
		}
		return out, model.Field

	case "struct_specifier", "union_specifier":
		if node.ChildByFieldName("name") == nil {
			// 匿名 struct：不产独立元素，字段挂承载实体（决策 #2）。
			return nil, ""
		}
		if isTypeReference(node) {
			// 局部/字段声明中的类型引用（struct Node n;）不是定义。
			return nil, ""
		}
		name := normalizeDeclaredTypeName(node.ChildByFieldName("name"), src)
		return []identified{{
			elem:   newElem(model.Struct, name, tmplParams),
			isDecl: node.ChildByFieldName("body") == nil,
		}}, model.Struct

	case "class_specifier":
		if node.ChildByFieldName("name") == nil {
			return nil, ""
		}
		if isTypeReference(node) {
			return nil, ""
		}
		name := normalizeDeclaredTypeName(node.ChildByFieldName("name"), src)
		return []identified{{
			elem:   newElem(model.Class, name, tmplParams),
			isDecl: node.ChildByFieldName("body") == nil,
		}}, model.Class

	case "enum_specifier":
		if node.ChildByFieldName("name") == nil {
			return nil, ""
		}
		if isTypeReference(node) {
			return nil, ""
		}
		name := normalizeDeclaredTypeName(node.ChildByFieldName("name"), src)
		return []identified{{
			elem:   newElem(model.Enum, name, tmplParams),
			isDecl: node.ChildByFieldName("body") == nil,
		}}, model.Enum

	case "enumerator":
		nameNode := node.ChildByFieldName("name")
		if nameNode == nil {
			return nil, ""
		}
		name := helper.GetNodeContent(nameNode, src)
		return []identified{{elem: newElem(model.EnumConstant, name, tmplParams)}}, model.EnumConstant

	case "type_definition":
		// typedef 可含多个声明器：typedef int TI1, TI2; / typedef struct {...} A, *B;
		var out []identified
		for _, d := range collectDeclarators(node) {
			name, _ := declaratorFullName(d, src)
			if name == "" {
				continue
			}
			out = append(out, identified{elem: newElem(model.Type, name, tmplParams), declNode: d})
		}
		if len(out) == 0 {
			return nil, ""
		}
		return out, model.Type

	case "alias_declaration":
		nameNode := node.ChildByFieldName("name")
		if nameNode == nil {
			return nil, ""
		}
		name := helper.GetNodeContent(nameNode, src)
		return []identified{{elem: newElem(model.Type, name, tmplParams)}}, model.Type

	case "namespace_definition":
		nameNode := node.ChildByFieldName("name")
		name := "$anon"
		if nameNode != nil {
			name = helper.GetNodeContent(nameNode, src)
		}
		return []identified{{elem: newElem(model.Namespace, name, tmplParams)}}, model.Namespace

	case "preproc_def", "preproc_function_def":
		nameNode := node.ChildByFieldName("name")
		if nameNode == nil {
			return nil, ""
		}
		name := helper.GetNodeContent(nameNode, src)
		elem := newElem(model.Macro, name, tmplParams)
		elem.GetMores()[constants.MacroName] = name
		if v := node.ChildByFieldName("value"); v != nil {
			elem.GetMores()[constants.MacroBody] = helper.GetNodeContent(v, src)
		}
		elem.GetMores()[constants.MacroIsFunctionLike] = node.Kind() == "preproc_function_def"
		return []identified{{elem: elem}}, model.Macro

	case "parameter_declaration":
		name := paramName(node, src)
		if name == "" {
			return nil, ""
		}
		return []identified{{elem: newElem(model.Variable, name, tmplParams)}}, model.Variable

	case "for_range_loop":
		// range-for 的循环变量不在 declaration 节点内（type/declarator 直接挂 for_range_loop）。
		decl := node.ChildByFieldName("declarator")
		name, _ := declaratorFullName(decl, src)
		if name == "" {
			return nil, ""
		}
		return []identified{{elem: newElem(model.Variable, name, tmplParams), declNode: decl}}, model.Variable

	case "lambda_expression":
		return []identified{{elem: newElem(model.Lambda, "lambda", tmplParams)}}, model.Lambda

	case "compound_statement":
		if !isBlockScope(node) {
			return nil, ""
		}
		return []identified{{elem: newElem(model.ScopeBlock, "block", tmplParams)}}, model.ScopeBlock
	}
	return nil, ""
}

// identifyDeclaration 处理 declaration：函数原型 → *Decl；其余声明器 → Variable。
// 混合形态（int mixf(int), mixv;）逐个声明器独立判定，不因命中首个函数而丢弃其余变量。
func (c *Collector) identifyDeclaration(node coreast.Node, fCtx core.FileContext, tmplParams string) ([]identified, model.ElementKind) {
	src := *fCtx.SourceBytes()
	decls := collectDeclarators(node)
	if len(decls) == 0 {
		return nil, ""
	}

	_, extern, _ := scanStorage(node, src)
	var out []identified
	kind := model.ElementKind("")
	for _, d := range decls {
		// 结构化绑定（auto [a, b] = ...）：每个绑定名各产一个变量元素。
		if bindings := structuredBindingNames(d, src); len(bindings) > 0 {
			for _, b := range bindings {
				out = append(out, identified{elem: newElem(model.Variable, b, tmplParams), declNode: d})
			}
			if kind == "" {
				kind = model.Variable
			}
			continue
		}
		name, _ := declaratorFullName(d, src)
		if name == "" {
			continue
		}
		if isFunc, _ := classifyDeclarator(d, src); isFunc {
			k := model.FunctionDecl
			if isMethodContext(node, d, src) {
				k = model.MethodDecl
			}
			elem := newElem(k, name, tmplParams)
			elem.GetMores()[constants.FunctionIsDeclaration] = true
			out = append(out, identified{elem: elem, declNode: d, sig: paramSignature(d, src), isDecl: true})
			if kind == "" {
				kind = k
			}
			continue
		}
		out = append(out, identified{elem: newElem(model.Variable, name, tmplParams), declNode: d, isDecl: extern})
		if kind == "" {
			kind = model.Variable
		}
	}
	if len(out) == 0 {
		return nil, ""
	}
	return out, kind
}

// structuredBindingNames 返回结构化绑定声明器中的各绑定名。
// 形态：init_declarator -> structured_binding_declarator [ identifier, identifier, ... ]
func structuredBindingNames(decl coreast.Node, src []byte) []string {
	d := decl
	if d != nil && d.Kind() == "init_declarator" {
		d = innerDeclarator(d)
	}
	if d == nil || d.Kind() != "structured_binding_declarator" {
		return nil
	}
	var out []string
	for i := 0; i < d.NamedChildCount(); i++ {
		ch := d.NamedChild(i)
		if ch.Kind() == "identifier" {
			out = append(out, strings.TrimSpace(ch.Utf8Text(src)))
		}
	}
	return out
}

// newElem 构造一个源码元素（Location/Path 由调用方填充）。
func newElem(kind model.ElementKind, name, tmplParams string) *model.CodeElement {
	elem := &model.CodeElement{
		Kind:         kind,
		Name:         name,
		IsFormSource: true,
	}
	if tmplParams != "" {
		elem.GetMores()[constants.TemplateParams] = tmplParams
	}
	return elem
}

// =============================================================================
// QN 计算
// =============================================================================

// assignQN 计算元素 QualifiedName（含声明器限定名与函数签名）。
func (c *Collector) assignQN(it *identified, currentQN string, occurrences map[string]int, src []byte) {
	elem := it.elem

	// 限定声明器（out-of-line 定义/类外变量）：相对限定名按当前命名空间补全前缀。
	if it.declNode != nil {
		if full, ok := declaratorFullName(it.declNode, src); ok && strings.Contains(full, helper.QNSep) {
			resolved := resolveRelativeQN(full, currentQN)
			elem.QualifiedName = resolved + it.sig
			it.parentQN = helper.PkgOf(resolved)
			// Name 只取最后一段（类外定义 Widget::~Widget 的 Name 应为 ~Widget），
			// 否则构造/析构/运算符等按名称判定的语义会失效。
			if parts := helper.SplitQN(resolved); len(parts) > 0 {
				elem.Name = parts[len(parts)-1]
			}
			return
		}
	}
	it.parentQN = currentQN

	identity := elem.Name
	if isFunctionKind(elem.Kind) {
		identity += it.sig
	}

	// 声明与定义共享同一 QN：声明不参与同名重复计数，
	// 否则同文件的 `int f(int); int f(int){}` 会被消歧成 f(int) / f(int)$2。
	if it.isDecl {
		elem.QualifiedName = c.resolver.BuildQualifiedName(currentQN, identity)
		return
	}

	switch elem.Kind {
	case model.Lambda, model.ScopeBlock:
		occurrences[elem.Name]++
		identity = fmt.Sprintf("%s$%d", elem.Name, occurrences[elem.Name])
	default:
		occurrences[identity]++
		if occurrences[identity] > 1 {
			identity = fmt.Sprintf("%s$%d", identity, occurrences[identity])
		}
	}
	elem.QualifiedName = c.resolver.BuildQualifiedName(currentQN, identity)
}

func isFunctionKind(k model.ElementKind) bool {
	switch k {
	case model.Function, model.Method, model.FunctionDecl, model.MethodDecl:
		return true
	}
	return false
}

// =============================================================================
// 声明器工具
// =============================================================================

// collectDeclarators 返回声明节点中的全部声明器（跳过类型/存储类/限定符）。
func collectDeclarators(node coreast.Node) []coreast.Node {
	typeNode := node.ChildByFieldName("type")
	var out []coreast.Node
	for i := 0; i < node.NamedChildCount(); i++ {
		ch := node.NamedChild(i)
		if ch == nil {
			continue
		}
		if typeNode != nil && ch.Equal(typeNode) {
			continue
		}
		switch ch.Kind() {
		case "storage_class_specifier", "type_qualifier", "attribute_declaration",
			"attribute_specifier", "virtual_specifier", "alignas_qualifier", "ms_declspec_modifier":
			continue
		}
		out = append(out, ch)
	}
	return out
}

// classifyDeclarator 判定声明器是否为函数形态。
// 函数指针变量（int (*fp)(int)）的 function_declarator 内层为 parenthesized_declarator
// 且括号内无函数声明器，不视为函数；
// 返回函数指针/数组指针的函数（int (*rfp(int))(int) / int (*rap(void))[5]）括号内有
// 函数声明器，仍视为函数。
func classifyDeclarator(decl coreast.Node, src []byte) (bool, string) {
	if decl == nil {
		return false, ""
	}
	switch decl.Kind() {
	case "function_declarator":
		inner := decl.ChildByFieldName("declarator")
		if inner != nil && inner.Kind() == "parenthesized_declarator" {
			if findFunctionDeclaratorInParens(inner) == nil {
				return false, ""
			}
		}
		name, _ := declaratorFullName(inner, src)
		return true, name
	case "pointer_declarator", "reference_declarator", "init_declarator", "array_declarator":
		return classifyDeclarator(innerDeclarator(decl), src)
	case "parenthesized_declarator":
		if findFunctionDeclaratorInParens(decl) == nil {
			return false, ""
		}
		name, _ := declaratorFullName(decl, src)
		return true, name
	case "operator_cast":
		// 转换运算符（operator bool / operator int）本身即函数形态。
		name, _ := declaratorFullName(decl, src)
		return true, name
	}
	return false, ""
}

// innerDeclarator 返回声明器的内层声明器。
// tree-sitter-cpp 中 pointer_declarator/init_declarator/array_declarator 用 declarator 字段，
// 但 reference_declarator（`D& f()` / `operator=`）没有该字段，内层声明器直接作为命名子节点。
func innerDeclarator(decl coreast.Node) coreast.Node {
	if decl == nil {
		return nil
	}
	if d := decl.ChildByFieldName("declarator"); d != nil {
		return d
	}
	for i := 0; i < decl.NamedChildCount(); i++ {
		ch := decl.NamedChild(i)
		switch ch.Kind() {
		case "function_declarator", "pointer_declarator", "reference_declarator",
			"array_declarator", "parenthesized_declarator", "qualified_identifier",
			"identifier", "type_identifier", "field_identifier", "operator_name",
			"destructor_name", "template_function", "attributed_declarator":
			return ch
		}
	}
	return nil
}

// declaratorFullName 提取声明器的名称（剥指针/引用/数组/函数外层）。
// 第二个返回值为 true 表示名称为限定名（含 ::）。
func declaratorFullName(decl coreast.Node, src []byte) (string, bool) {
	if decl == nil {
		return "", false
	}
	switch decl.Kind() {
	case "pointer_declarator", "reference_declarator", "init_declarator",
		"array_declarator", "attributed_declarator":
		return declaratorFullName(innerDeclarator(decl), src)
	case "function_declarator":
		return declaratorFullName(innerDeclarator(decl), src)
	case "parenthesized_declarator":
		for i := 0; i < decl.NamedChildCount(); i++ {
			if n, ok := declaratorFullName(decl.NamedChild(i), src); n != "" {
				return n, ok
			}
		}
		return "", false
	case "structured_binding_declarator":
		// 结构化绑定（auto [a, b] = ...）：无单一名称，由 identifyDeclaration 展开。
		return "", false
	case "qualified_identifier":
		return normalizeQualified(decl.Utf8Text(src)), true
	case "template_function":
		// 函数模板显式特化（template <> void f<int>(...)）的名称节点：
		// 取 name 字段（f），剥离模板实参。
		return declaratorFullName(decl.ChildByFieldName("name"), src)
	case "operator_cast":
		// 转换运算符（operator bool / operator int）：名称由 operator + 目标类型组成。
		t := ""
		if tn := decl.ChildByFieldName("type"); tn != nil {
			t = normalizeTypeText(helper.GetNodeContent(tn, src))
		}
		return normalizeTypeText("operator " + t), false
	case "destructor_name", "operator_name", "field_identifier", "identifier", "type_identifier":
		return strings.TrimSpace(decl.Utf8Text(src)), false
	}
	return "", false
}

// normalizeDeclaredTypeName 归一化类型定义的名称节点：剥离模板实参。
// 模板特化（template <> struct Box<int>）的 name 是 template_type，其文本含实参；
// QN 契约要求剥离（Box<int> → Box），与类外定义 Box<int>::run 的归一化一致。
func normalizeDeclaredTypeName(nameNode coreast.Node, src []byte) string {
	if nameNode == nil {
		return ""
	}
	if nameNode.Kind() == "template_type" {
		if n := nameNode.ChildByFieldName("name"); n != nil {
			return helper.GetNodeContent(n, src)
		}
	}
	return helper.StripTemplateArgs(helper.GetNodeContent(nameNode, src))
}

// normalizeQualified 归一化限定名文本：去掉空白与各段的模板实参。
// 例：app::Box<int>::run -> app::Box::run
func normalizeQualified(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\t", "")
	s = strings.ReplaceAll(s, "\n", "")
	parts := strings.Split(s, helper.QNSep)
	for i, p := range parts {
		parts[i] = helper.StripTemplateArgs(p)
	}
	return strings.Join(parts, helper.QNSep)
}

// isMethodContext 判定函数是否属于类/结构体方法。
func isMethodContext(node, decl coreast.Node, src []byte) bool {
	if decl != nil {
		if full, ok := declaratorFullName(decl, src); ok && strings.Contains(full, helper.QNSep) {
			return true
		}
	}
	for curr := node.Parent(); curr != nil; curr = curr.Parent() {
		switch curr.Kind() {
		case "field_declaration_list":
			p := curr.Parent()
			if p != nil && (p.Kind() == "class_specifier" || p.Kind() == "struct_specifier" || p.Kind() == "union_specifier") {
				return true
			}
		case "function_definition", "translation_unit":
			return false
		}
	}
	return false
}

// isTypeReference 判定 struct/class/enum 说明符是否为「类型引用」而非定义。
// 无 body 且父节点为声明/字段/参数/typedef 时是引用（struct Node n; / struct Fwd; 除外）。
func isTypeReference(node coreast.Node) bool {
	if node.ChildByFieldName("body") != nil {
		return false
	}
	p := node.Parent()
	if p == nil {
		return false
	}
	switch p.Kind() {
	case "translation_unit", "declaration_list":
		// 顶层/命名空间内的前向声明：保留为元素（记 is_declaration）。
		return false
	}
	return true
}

// isBlockScope 判定 compound_statement 是否应作为独立作用域块。
// 函数体/Lambda 体的块由函数/Lambda 自身承载，不重复建块。
func isBlockScope(node coreast.Node) bool {
	p := node.Parent()
	if p == nil {
		return false
	}
	switch p.Kind() {
	case "function_definition", "lambda_expression", "field_declaration_list",
		"declaration_list", "translation_unit":
		return false
	}
	return true
}

// paramSignature 返回函数参数类型签名 "(int,char*)"。
func paramSignature(decl coreast.Node, src []byte) string {
	fd := findFunctionDeclarator(decl)
	if fd == nil {
		return "()"
	}
	pNode := fd.ChildByFieldName("parameters")
	if pNode == nil {
		return "()"
	}
	var types []string
	for i := 0; i < pNode.NamedChildCount(); i++ {
		t := paramTypeText(pNode.NamedChild(i), src)
		if t == "" {
			continue
		}
		types = append(types, t)
	}
	// f(void) 等价于无参。
	if len(types) == 1 && types[0] == "void" {
		types = nil
	}
	return "(" + strings.Join(types, ",") + ")"
}

// findFunctionDeclarator 沿声明器链找到「被声明实体自身」的 function_declarator。
// 返回函数指针的函数（int (*rfp(int))(int)）外层 function_declarator 描述的是返回类型，
// 需穿透 parenthesized_declarator 找到内层 rfp 的 function_declarator。
func findFunctionDeclarator(decl coreast.Node) coreast.Node {
	for decl != nil {
		switch decl.Kind() {
		case "function_declarator":
			inner := decl.ChildByFieldName("declarator")
			if inner != nil && inner.Kind() == "parenthesized_declarator" {
				if fd := findFunctionDeclaratorInParens(inner); fd != nil {
					return fd
				}
			}
			return decl
		case "pointer_declarator", "reference_declarator", "init_declarator", "array_declarator":
			decl = innerDeclarator(decl)
		case "parenthesized_declarator":
			return findFunctionDeclaratorInParens(decl)
		default:
			return nil
		}
	}
	return nil
}

// findFunctionDeclaratorInParens 在括号声明器内查找函数声明器（穿透指针/引用/数组）。
func findFunctionDeclaratorInParens(paren coreast.Node) coreast.Node {
	if paren == nil {
		return nil
	}
	for i := 0; i < paren.NamedChildCount(); i++ {
		if fd := findFunctionDeclarator(paren.NamedChild(i)); fd != nil {
			return fd
		}
	}
	return nil
}

// paramTypeText 提取参数的类型文本（含 const/指针/引用/数组语义）。
func paramTypeText(p coreast.Node, src []byte) string {
	if p == nil {
		return ""
	}
	if p.Kind() == "variadic_parameter" {
		return "..."
	}
	// optional_parameter_declaration 是带默认实参的形参（int b = 2），
	// 与 parameter_declaration 结构一致，但默认值不能进入 QN 签名。
	if p.Kind() != "parameter_declaration" && p.Kind() != "optional_parameter_declaration" {
		return normalizeTypeText(helper.GetNodeContent(p, src))
	}
	var parts []string
	for i := 0; i < p.ChildCount(); i++ {
		ch := p.Child(i)
		if ch.Kind() == "type_qualifier" {
			parts = append(parts, helper.GetNodeContent(ch, src))
		}
	}
	if t := p.ChildByFieldName("type"); t != nil {
		parts = append(parts, helper.GetNodeContent(t, src))
	}
	decl := p.ChildByFieldName("declarator")
	if decl != nil {
		if findFunctionDeclarator(decl) != nil {
			parts = append(parts, "(*)")
		} else {
			parts = append(parts, declaratorTypeSuffix(decl))
		}
	}
	return normalizeTypeText(strings.Join(parts, " "))
}

// declaratorTypeSuffix 返回声明器上的指针/引用/数组标记。
func declaratorTypeSuffix(decl coreast.Node) string {
	var sb strings.Builder
	for decl != nil {
		switch decl.Kind() {
		case "pointer_declarator", "abstract_pointer_declarator":
			sb.WriteString("*")
			decl = decl.ChildByFieldName("declarator")
		case "reference_declarator", "abstract_reference_declarator":
			// 右值引用（int&&）与左值引用（int&）在 QN 中必须区分。
			if countChildToken(decl, "&&") > 0 {
				sb.WriteString("&&")
			} else {
				sb.WriteString("&")
			}
			decl = innerDeclarator(decl)
		case "array_declarator", "abstract_array_declarator":
			sb.WriteString("[]")
			decl = innerDeclarator(decl)
		default:
			return sb.String()
		}
	}
	return sb.String()
}

// hasVariadicToken 判断参数列表中是否出现 `...`。
// tree-sitter-cpp 把可变参数的 `...` 作为匿名 token 子节点（不是 variadic_parameter）。
func hasVariadicToken(params coreast.Node) bool {
	if params == nil {
		return false
	}
	for i := 0; i < params.ChildCount(); i++ {
		if params.Child(i).Kind() == "..." {
			return true
		}
	}
	return false
}

// countChildToken 统计声明器直接子节点中指定运算符 token 的出现次数。
func countChildToken(decl coreast.Node, token string) int {
	n := 0
	for i := 0; i < decl.ChildCount(); i++ {
		if decl.Child(i).Kind() == token {
			n++
		}
	}
	return n
}

// normalizeTypeText 归一化类型文本：去掉 * & 前空格。
func normalizeTypeText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " *", "*")
	s = strings.ReplaceAll(s, " &", "&")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.ReplaceAll(s, " [", "[")
	return s
}

// paramName 提取参数名。
func paramName(p coreast.Node, src []byte) string {
	if p == nil || (p.Kind() != "parameter_declaration" && p.Kind() != "optional_parameter_declaration") {
		return ""
	}
	name, _ := declaratorFullName(p.ChildByFieldName("declarator"), src)
	return name
}

// =============================================================================
// 元数据增强
// =============================================================================

func (c *Collector) enrichMetadata(fCtx core.FileContext) {
	src := *fCtx.SourceBytes()
	for _, entry := range fCtx.Definitions() {
		elem, node := entry.Element, entry.Node
		if node == nil {
			continue
		}
		mores := elem.GetMores()

		// 通用：命名空间 / 可见性
		// （条件编译分支在 collectBasicDefinitions 注册时记录，此时条件栈已全部弹出）
		if elem.Kind != model.File {
			mores[constants.NamespaceName] = helper.PkgOf(elem.QualifiedName)
		}
		if vis := visibilityOf(node, src); vis != "" {
			mores[constants.Visibility] = vis
		}

		switch elem.Kind {
		case model.File:
			mores[constants.FileRawLOC] = len(strings.Split(string(src), "\n"))
			mores[constants.FileLOC] = countEffectiveLOC(src)
		case model.Function, model.Method, model.FunctionDecl, model.MethodDecl:
			c.enrichFunction(elem, node, src)
		case model.Variable:
			c.enrichVariable(elem, node, src)
			c.linkLambdaInitializer(elem, entry, fCtx, src)
		case model.Field:
			c.enrichField(elem, node, src)
		case model.Struct:
			c.enrichStruct(elem, node, src)
		case model.Class:
			c.enrichClass(elem, node, src)
		case model.Enum:
			c.enrichEnum(elem, node, src)
		case model.Type:
			c.enrichType(elem, node, src)
		}
	}
}

func (c *Collector) enrichFunction(elem *model.CodeElement, node coreast.Node, src []byte) {
	mores := elem.GetMores()
	decl := node.ChildByFieldName("declarator")

	// 返回类型
	if t := node.ChildByFieldName("type"); t != nil {
		mores[constants.FunctionReturnType] = normalizeTypeText(helper.GetNodeContent(t, src))
	}

	// 参数列表
	fd := findFunctionDeclarator(decl)
	if fd != nil {
		if pNode := fd.ChildByFieldName("parameters"); pNode != nil {
			var params []string
			variadic := hasVariadicToken(pNode)
			for i := 0; i < pNode.NamedChildCount(); i++ {
				p := pNode.NamedChild(i)
				if p.Kind() == "variadic_parameter" {
					// C grammar 用 variadic_parameter 节点，C++ grammar 用匿名 `...` token。
					variadic = true
					params = append(params, "...")
					continue
				}
				t := paramTypeText(p, src)
				if t == "" {
					continue
				}
				if n := paramName(p, src); n != "" {
					params = append(params, t+" "+n)
				} else {
					params = append(params, t)
				}
			}
			if variadic && (len(params) == 0 || params[len(params)-1] != "...") {
				params = append(params, "...")
			}
			if !(len(params) == 1 && params[0] == "void") {
				mores[constants.FunctionParameters] = params
			}
			if variadic {
				mores[constants.FunctionIsVariadic] = true
			}
		}
	}

	// 修饰符：storage_class_specifier / type_qualifier / virtual
	var mods []string
	isVirtual := false
	scanModifiers(node, src, &mods, &isVirtual)
	if fd != nil {
		scanModifiers(fd, src, &mods, &isVirtual)
	}
	if len(mods) > 0 {
		if elem.Extra == nil {
			elem.Extra = &model.Extra{}
		}
		elem.Extra.Modifiers = mods
	}
	if isVirtual {
		mores[constants.FunctionIsVirtual] = true
	}

	// 纯虚函数：field_declaration 的 default_value 为 0
	if node.Kind() == "field_declaration" {
		if dv := node.ChildByFieldName("default_value"); dv != nil {
			if strings.TrimSpace(dv.Utf8Text(src)) == "0" {
				mores[constants.FunctionIsPureVirtual] = true
			}
		}
	}

	// 名称语义
	name := elem.Name
	if strings.HasPrefix(name, "operator") {
		mores[constants.FunctionIsOperator] = true
	}
	if strings.HasPrefix(name, "~") {
		mores[constants.FunctionIsDestructor] = true
	}
	if node.ChildByFieldName("type") == nil && !strings.HasPrefix(name, "~") && !strings.HasPrefix(name, "operator") {
		mores[constants.FunctionIsConstructor] = true
	}
}

func scanModifiers(node coreast.Node, src []byte, mods *[]string, isVirtual *bool) {
	if node == nil {
		return
	}
	for i := 0; i < node.ChildCount(); i++ {
		ch := node.Child(i)
		switch ch.Kind() {
		case "storage_class_specifier", "type_qualifier", "virtual_specifier":
			*mods = append(*mods, helper.GetNodeContent(ch, src))
		case "virtual":
			*isVirtual = true
			*mods = append(*mods, "virtual")
		}
	}
}

func (c *Collector) enrichVariable(elem *model.CodeElement, node coreast.Node, src []byte) {
	mores := elem.GetMores()
	if t := node.ChildByFieldName("type"); t != nil {
		mores[constants.VariableRawType] = normalizeTypeText(helper.GetNodeContent(t, src))
	}
	static, extern, isConst := scanStorage(node, src)
	if static {
		mores[constants.VariableIsStatic] = true
	}
	if extern {
		mores[constants.VariableIsExtern] = true
		mores[constants.IsDeclaration] = true
	}
	if isConst {
		mores[constants.VariableIsConst] = true
	}
	if isGlobalDeclaration(node) {
		mores[constants.VariableIsGlobal] = true
	}
}

func (c *Collector) enrichField(elem *model.CodeElement, node coreast.Node, src []byte) {
	mores := elem.GetMores()
	if t := node.ChildByFieldName("type"); t != nil {
		mores[constants.FieldRawType] = normalizeTypeText(helper.GetNodeContent(t, src))
	}
	static, _, isConst := scanStorage(node, src)
	if static {
		mores[constants.FieldIsStatic] = true
	}
	if isConst {
		mores[constants.FieldIsConst] = true
	}
}

func (c *Collector) enrichStruct(elem *model.CodeElement, node coreast.Node, src []byte) {
	mores := elem.GetMores()
	if node.Kind() == "union_specifier" {
		mores[constants.StructIsUnion] = true
	}
	if node.ChildByFieldName("body") == nil {
		mores[constants.StructIsDeclaration] = true
		mores[constants.IsDeclaration] = true
	}
	// struct 同样可继承（struct D : public B），基类记录规则与 class 一致。
	if supers := collectBaseClasses(node, src); len(supers) > 0 {
		mores[constants.ClassSuperClass] = supers
	}
}

func (c *Collector) enrichClass(elem *model.CodeElement, node coreast.Node, src []byte) {
	mores := elem.GetMores()
	if node.ChildByFieldName("body") == nil {
		mores[constants.ClassIsDeclaration] = true
		mores[constants.IsDeclaration] = true
	}
	if supers := collectBaseClasses(node, src); len(supers) > 0 {
		mores[constants.ClassSuperClass] = supers
	}
}

// collectBaseClasses 提取 base_class_clause 中的基类名原文。
// 基类名保留原文（可能相对当前命名空间），由 Extractor 结合全局上下文解析。
func collectBaseClasses(node coreast.Node, src []byte) []string {
	var supers []string
	for i := 0; i < node.NamedChildCount(); i++ {
		ch := node.NamedChild(i)
		if ch.Kind() != "base_class_clause" {
			continue
		}
		for j := 0; j < ch.NamedChildCount(); j++ {
			b := ch.NamedChild(j)
			switch b.Kind() {
			case "access_specifier", "virtual":
				continue
			case "qualified_identifier", "type_identifier", "template_type":
				supers = append(supers, normalizeTypeText(helper.GetNodeContent(b, src)))
			}
		}
	}
	return supers
}

// linkLambdaInitializer 记录「变量初始化为 lambda」的关联 Lambda QN，
// 使 `auto f = [](){...}; f();` 的调用能解析到 Lambda 元素。
func (c *Collector) linkLambdaInitializer(elem *model.CodeElement, entry *core.DefinitionEntry, fCtx core.FileContext, src []byte) {
	node := entry.Node
	if node == nil || node.Kind() != "declaration" {
		return
	}
	// 定位该变量对应的声明器，取其 value 是否为 lambda_expression。
	for _, d := range collectDeclarators(node) {
		name, _ := declaratorFullName(d, src)
		if name != elem.Name {
			continue
		}
		if d.Kind() != "init_declarator" {
			return
		}
		v := d.ChildByFieldName("value")
		if v == nil || v.Kind() != "lambda_expression" {
			return
		}
		// 在已收集定义中找位置匹配的 Lambda 元素。
		for _, def := range fCtx.Definitions() {
			if def.Element.Kind != model.Lambda || def.Node == nil {
				continue
			}
			if def.Node.StartByte() == v.StartByte() && def.Node.EndByte() == v.EndByte() {
				elem.GetMores()[constants.VariableLambdaQN] = def.Element.QualifiedName
				return
			}
		}
		return
	}
}

// enrichType 记录 typedef/using 别名的底层类型（供 Binder 解析别名链）。
func (c *Collector) enrichType(elem *model.CodeElement, node coreast.Node, src []byte) {
	mores := elem.GetMores()
	switch node.Kind() {
	case "type_definition":
		if t := node.ChildByFieldName("type"); t != nil {
			underlying := normalizeTypeText(helper.GetNodeContent(t, src))
			// 多声明器时各别名的指针/数组修饰不同：typedef int TI, *PTI;
			for _, d := range collectDeclarators(node) {
				name, _ := declaratorFullName(d, src)
				if name != elem.Name {
					continue
				}
				underlying += declaratorTypeSuffix(d)
				break
			}
			mores[constants.TypeUnderlying] = underlying
		}
	case "alias_declaration":
		if t := node.ChildByFieldName("type"); t != nil {
			raw := helper.GetNodeContent(t, src)
			// type_descriptor 的文本含内部类型，直接使用即可。
			mores[constants.TypeUnderlying] = normalizeTypeText(raw)
		}
	}
}

func (c *Collector) enrichEnum(elem *model.CodeElement, node coreast.Node, src []byte) {
	text := node.Utf8Text(src)
	if strings.Contains(text, "enum class") || strings.Contains(text, "enum struct") {
		elem.GetMores()[constants.EnumIsScoped] = true
	}
	// 前向声明（enum Tagged;）无 body，无枚举常量。
	if node.ChildByFieldName("body") == nil {
		elem.GetMores()[constants.IsDeclaration] = true
	}
}

// countEffectiveLOC 计算有效代码行数（排除空行与纯注释行）。
// 简化处理：不处理块注释内的代码行，与 Java 版语义对齐（Java 版移除块注释后再统计）。
func countEffectiveLOC(src []byte) int {
	text := string(src)
	// 移除块注释。
	for {
		start := strings.Index(text, "/*")
		if start < 0 {
			break
		}
		end := strings.Index(text[start+2:], "*/")
		if end < 0 {
			text = text[:start]
			break
		}
		text = text[:start] + text[start+2+end+2:]
	}
	loc := 0
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
			loc++
		}
	}
	return loc
}

// scanStorage 提取 static/extern/const 标记。
func scanStorage(node coreast.Node, src []byte) (static, extern, isConst bool) {
	if node == nil {
		return
	}
	for i := 0; i < node.ChildCount(); i++ {
		ch := node.Child(i)
		switch ch.Kind() {
		case "storage_class_specifier":
			switch strings.TrimSpace(ch.Utf8Text(src)) {
			case "static":
				static = true
			case "extern":
				extern = true
			}
		case "type_qualifier":
			if strings.TrimSpace(ch.Utf8Text(src)) == "const" {
				isConst = true
			}
		}
	}
	return
}

// isGlobalDeclaration 判定 declaration 是否为全局变量（顶层或 namespace 内）。
func isGlobalDeclaration(node coreast.Node) bool {
	curr := node.Parent()
	for curr != nil {
		switch curr.Kind() {
		case "translation_unit":
			return true
		case "declaration_list":
			p := curr.Parent()
			if p != nil && p.Kind() == "namespace_definition" {
				return true
			}
			return false
		case "linkage_specification":
			curr = curr.Parent()
			continue
		default:
			return false
		}
	}
	return false
}

// visibilityOf 返回成员所在的访问级别（最近的前置 access_specifier）。
func visibilityOf(node coreast.Node, src []byte) string {
	p := node.Parent()
	if p == nil || p.Kind() != "field_declaration_list" {
		return ""
	}
	vis := ""
	for i := 0; i < p.ChildCount(); i++ {
		ch := p.Child(i)
		if ch.Kind() == "access_specifier" {
			vis = strings.TrimSpace(ch.Utf8Text(src))
			continue
		}
		if ch.Equal(node) {
			return vis
		}
	}
	return vis
}

// collectNamespaces 汇总文件内出现的命名空间（含嵌套前缀）。
func (c *Collector) collectNamespaces(fCtx core.FileContext) {
	seen := make(map[string]bool)
	var out []string
	for _, entry := range fCtx.Definitions() {
		if entry.Element.Kind != model.Namespace {
			continue
		}
		parts := helper.SplitQN(entry.Element.QualifiedName)
		var cur []string
		for _, p := range parts {
			cur = append(cur, p)
			full := strings.Join(cur, helper.QNSep)
			if !seen[full] {
				seen[full] = true
				out = append(out, full)
			}
		}
	}
	if len(out) > 0 {
		if nfc, ok := fCtx.(cppctx.NamespaceAware); ok {
			nfc.SetNamespaces(out)
		}
	}
}
