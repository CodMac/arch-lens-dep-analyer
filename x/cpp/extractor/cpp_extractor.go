// Package extractor 实现 C/C++ 语言 core.Extractor 接口：
// 从单个文件提取静态结构关系（Import/Contain/Extend/Parameter/Return/TypeArg）
// 与动作关系（Call/Create/Cast/Assign/Use/Throw）。
package extractor

import (
	"fmt"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/helper"
)

// Extractor 实现 core.Extractor。
type Extractor struct {
	resolver core.SymbolResolver
}

// New 创建 C/C++ Extractor。
func New(resolver core.SymbolResolver) *Extractor {
	return &Extractor{resolver: resolver}
}

// Extract 提取指定文件（相对路径）的全部依赖关系。
func (e *Extractor) Extract(relPath string, gc core.GlobalContext) ([]*model.DependencyRelation, error) {
	fCtx, ok := gc.FileContext(relPath)
	if !ok {
		return nil, fmt.Errorf("file context not found: %s", relPath)
	}

	var rels []*model.DependencyRelation
	rels = append(rels, e.extractHierarchy(fCtx, gc)...)
	rels = append(rels, e.extractStructural(fCtx, gc)...)
	actionRels := e.extractActions(fCtx, gc)
	rels = append(rels, actionRels...)
	rels = append(rels, deriveCaptures(actionRels)...)
	return rels, nil
}

// deriveCaptures 从 Lambda 的 USE/ASSIGN 推导 CAPTURE 关系：
// 目标为 Lambda 作用域之外的局部变量（非字段/枚举常量）即为闭包捕获。
func deriveCaptures(actionRels []*model.DependencyRelation) []*model.DependencyRelation {
	var out []*model.DependencyRelation
	seen := make(map[string]bool)
	for _, r := range actionRels {
		if r.Source == nil || r.Target == nil || r.Source.Kind != model.Lambda {
			continue
		}
		if r.Type != model.Use && r.Type != model.Assign {
			continue
		}
		if r.Target.Kind != model.Variable || r.Target.IsFormExternal {
			continue
		}
		// 目标位于 Lambda 自身作用域内（参数/局部变量）则非捕获。
		if strings.HasPrefix(r.Target.QualifiedName, r.Source.QualifiedName+helper.QNSep) {
			continue
		}
		key := r.Source.QualifiedName + "->" + r.Target.QualifiedName
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, &model.DependencyRelation{
			Type:     model.Capture,
			Source:   r.Source,
			Target:   r.Target,
			Location: r.Location,
			Mores:    map[string]interface{}{constants.RelRawText: r.Target.Name},
		})
	}
	return out
}

// =============================================================================
// 结构关系
// =============================================================================

// extractHierarchy 提取 Import（#include）与 Contain 关系。
func (e *Extractor) extractHierarchy(fCtx core.FileContext, gc core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation

	fileDef, ok := gc.FindByQualifiedName(fCtx.FilePath())
	if ok {
		fileSource := fileDef.Element
		for _, imports := range fCtx.Imports() {
			for _, imp := range imports {
				target := e.resolveInclude(gc, fCtx, imp)
				if target == nil {
					continue
				}
				rels = append(rels, &model.DependencyRelation{
					Type:     model.Import,
					Source:   fileSource,
					Target:   target,
					Location: imp.Location,
					Mores: map[string]interface{}{
						constants.RelRawText:         imp.RawImportPath,
						constants.RelIncludeIsSystem: imp.IsSystem,
					},
				})
			}
		}
	}

	// Contain 按 (SourceQN, TargetQN) 去重：声明/定义同 QN 共存时只产一条。
	seen := make(map[string]bool)
	for _, entry := range fCtx.Definitions() {
		if entry.ParentQN == "" {
			continue
		}
		parent, ok := gc.FindByQualifiedName(entry.ParentQN)
		if !ok {
			continue
		}
		key := entry.ParentQN + "->" + entry.Element.QualifiedName
		if seen[key] {
			continue
		}
		seen[key] = true
		rels = append(rels, &model.DependencyRelation{
			Type: model.Contain, Source: parent.Element, Target: entry.Element,
		})
	}
	return rels
}

// resolveInclude 解析 #include 目标：项目内头文件 → File 元素；系统头 → 外部符号。
func (e *Extractor) resolveInclude(gc core.GlobalContext, fCtx core.FileContext, imp *core.ImportInfo) *model.CodeElement {
	if !imp.IsSystem {
		// 项目内头文件：按路径后缀匹配已收集的 File 元素（相对项目根）。
		if target := e.findFileByInclude(gc, imp.RawImportPath); target != nil {
			return target
		}
	}
	return helper.NewExternalElement(model.File, imp.Alias, imp.RawImportPath)
}

// findFileByInclude 按 include 路径后缀匹配 File 元素。
func (e *Extractor) findFileByInclude(gc core.GlobalContext, raw string) *model.CodeElement {
	raw = strings.ReplaceAll(raw, "\\", "/")
	suffix := "/" + raw
	for _, def := range gc.Definitions() {
		if def.Element.Kind != model.File {
			continue
		}
		qn := def.Element.QualifiedName
		if qn == raw || strings.HasSuffix(qn, suffix) {
			return def.Element
		}
	}
	return nil
}

// extractStructural 提取 Extend/Parameter/Return/TypeArg。
func (e *Extractor) extractStructural(fCtx core.FileContext, gc core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	for _, entry := range fCtx.Definitions() {
		elem := entry.Element
		if elem.Extra == nil {
			continue
		}
		mores := elem.Extra.Mores

		// 继承（C++ base_class_clause）
		if elem.Kind == model.Class || elem.Kind == model.Struct {
			if supers, ok := mores[constants.ClassSuperClass].([]string); ok {
				for _, s := range supers {
					target := e.resolver.ResolveType(gc, fCtx, s, model.Class)
					if target == nil {
						continue
					}
					rels = append(rels, &model.DependencyRelation{
						Type: model.Extend, Source: elem, Target: target,
						Mores: map[string]interface{}{constants.RelRawText: s},
					})
				}
			}
		}

		// 函数签名（Parameter/Return）
		if isFunctionKind(elem.Kind) {
			rels = append(rels, e.extractSignatureRels(fCtx, gc, elem)...)
		}

		// 变量/字段类型（TypeArg + 类型 Use）
		rels = append(rels, e.extractTypeArgRels(fCtx, gc, elem)...)
	}
	return rels
}

// extractSignatureRels 提取 Parameter / Return 关系。
func (e *Extractor) extractSignatureRels(fCtx core.FileContext, gc core.GlobalContext, elem *model.CodeElement) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	mores := elem.Extra.Mores

	if params, ok := mores[constants.FunctionParameters].([]string); ok {
		qnParams, _ := mores[constants.FunctionParametersWithQN].([]string)
		for i, p := range params {
			if p == "..." {
				continue
			}
			var target *model.CodeElement
			if i < len(qnParams) {
				if t := e.lookupTypeQN(gc, qnParams[i]); t != nil {
					target = t
				}
			}
			if target == nil {
				typePart := paramTypeOnly(p)
				target = e.resolver.ResolveType(gc, fCtx, typePart, model.Class)
			}
			if target == nil {
				continue
			}
			rels = append(rels, &model.DependencyRelation{
				Type: model.Parameter, Source: elem, Target: target,
				Mores: map[string]interface{}{constants.RelRawText: p},
			})
		}
	}

	if rt, ok := mores[constants.FunctionReturnType].(string); ok && rt != "" && rt != "void" {
		var target *model.CodeElement
		if qn, ok := mores[constants.FunctionReturnTypeWithQN].(string); ok && qn != "" {
			target = e.lookupTypeQN(gc, qn)
		}
		if target == nil {
			target = e.resolver.ResolveType(gc, fCtx, rt, model.Class)
		}
		if target != nil {
			rels = append(rels, &model.DependencyRelation{
				Type: model.Return, Source: elem, Target: target,
				Mores: map[string]interface{}{constants.RelRawText: rt},
			})
		}
	}
	return rels
}

// extractTypeArgRels 提取变量/字段的类型依赖（TypeArg）与模板实参。
func (e *Extractor) extractTypeArgRels(fCtx core.FileContext, gc core.GlobalContext, elem *model.CodeElement) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	mores := elem.Extra.Mores

	var raw, qn string
	switch elem.Kind {
	case model.Variable:
		raw, _ = mores[constants.VariableRawType].(string)
		qn, _ = mores[constants.VariableTypeWithQN].(string)
	case model.Field:
		raw, _ = mores[constants.FieldRawType].(string)
		qn, _ = mores[constants.FieldTypeWithQN].(string)
	default:
		return nil
	}
	if raw == "" {
		return nil
	}

	target := e.lookupTypeQN(gc, qn)
	if target == nil {
		target = e.resolver.ResolveType(gc, fCtx, raw, model.Class)
	}
	if target == nil || target.Kind == model.Primitive {
		return nil
	}
	rels = append(rels, &model.DependencyRelation{
		Type: model.TypeArg, Source: elem, Target: target,
		Mores: map[string]interface{}{constants.RelRawText: raw},
	})

	// 模板实参：Foo<Bar, Baz> 的每个实参。
	for _, arg := range templateArgs(raw) {
		at := e.resolver.ResolveType(gc, fCtx, arg, model.Class)
		if at == nil || at.Kind == model.Primitive {
			continue
		}
		rels = append(rels, &model.DependencyRelation{
			Type: model.TypeArg, Source: elem, Target: at,
			Mores: map[string]interface{}{constants.RelRawText: arg},
		})
	}
	return rels
}

// lookupTypeQN 按 QN 查找类型元素（含声明 kind 容忍）。
func (e *Extractor) lookupTypeQN(gc core.GlobalContext, qn string) *model.CodeElement {
	if qn == "" {
		return nil
	}
	if entry, ok := gc.FindByQualifiedName(qn); ok {
		switch entry.Element.Kind {
		case model.Class, model.Struct, model.Enum, model.Type, model.Namespace:
			return entry.Element
		}
	}
	return nil
}

// =============================================================================
// 动作关系
// =============================================================================

// actionCtx 描述一次动作提取的上下文。
type actionCtx struct {
	resolver core.SymbolResolver
	fCtx     core.FileContext
	gc       core.GlobalContext
	rels     []*model.DependencyRelation
}

// extractActions 遍历 AST 提取动作关系。
func (e *Extractor) extractActions(fCtx core.FileContext, gc core.GlobalContext) []*model.DependencyRelation {
	ac := &actionCtx{resolver: e.resolver, fCtx: fCtx, gc: gc}
	ac.walk(fCtx.Root())
	return ac.rels
}

func (ac *actionCtx) walk(node coreast.Node) {
	if node == nil {
		return
	}
	switch node.Kind() {
	case "call_expression":
		ac.handleCall(node)
	case "new_expression":
		ac.handleNew(node)
	case "cast_expression":
		ac.handleCast(node)
	case "assignment_expression":
		ac.handleAssign(node)
	case "init_declarator":
		ac.handleInit(node)
	case "throw_statement":
		ac.handleThrow(node)
	case "field_initializer":
		ac.handleFieldInitializer(node)
	case "identifier":
		ac.handleUse(node)
	case "field_expression", "qualified_identifier":
		ac.handleUseComposite(node)
	}
	for i := 0; i < node.ChildCount(); i++ {
		ac.walk(node.Child(i))
	}
}

func (ac *actionCtx) add(relType model.DependencyType, source, target *model.CodeElement, node coreast.Node, mores map[string]interface{}) {
	if source == nil || target == nil {
		return
	}
	// 全局变量初始化（int g = 0;）的发起者即变量自身，自赋值不产关系。
	if source.QualifiedName == target.QualifiedName {
		return
	}
	rel := &model.DependencyRelation{
		Type:     relType,
		Source:   source,
		Target:   target,
		Location: helper.ExtractLocation(node, ac.fCtx.FilePath()),
		Mores:    mores,
	}
	ac.rels = append(ac.rels, rel)
}

// handleCall 处理函数调用。
func (ac *actionCtx) handleCall(node coreast.Node) {
	fn := node.ChildByFieldName("function")
	if fn == nil {
		return
	}
	source := ac.sourceOf(node)
	if source == nil {
		return
	}
	// static_cast<T>(x) 等显式转换：语义为 CAST 而非 CALL。
	if isCastCall(fn, *ac.fCtx.SourceBytes()) {
		target := ac.resolver.ResolveAction(ac.gc, ac.fCtx, fn, model.Call)
		if target != nil && target.Kind != model.Primitive {
			ac.add(model.Cast, source, target, fn, map[string]interface{}{
				constants.RelRawText: helper.GetNodeContent(fn, *ac.fCtx.SourceBytes()),
				constants.RelAstKind: "call_expression",
			})
		}
		return
	}
	target := ac.resolver.ResolveAction(ac.gc, ac.fCtx, fn, model.Call)
	ac.add(model.Call, source, target, fn, map[string]interface{}{
		constants.RelRawText: helper.GetNodeContent(fn, *ac.fCtx.SourceBytes()),
		constants.RelAstKind: "call_expression",
	})
}

// isCastCall 判断调用目标是否为 C++ 显式转换（static_cast 等）。
func isCastCall(fn coreast.Node, src []byte) bool {
	if fn == nil || fn.Kind() != "template_function" {
		return false
	}
	name := fn.ChildByFieldName("name")
	if name == nil {
		return false
	}
	switch strings.TrimSpace(name.Utf8Text(src)) {
	case "static_cast", "dynamic_cast", "const_cast", "reinterpret_cast":
		return true
	}
	return false
}

// handleNew 处理 C++ new 表达式。
func (ac *actionCtx) handleNew(node coreast.Node) {
	t := node.ChildByFieldName("type")
	if t == nil {
		return
	}
	source := ac.sourceOf(node)
	if source == nil {
		return
	}
	target := ac.resolver.ResolveAction(ac.gc, ac.fCtx, t, model.Create)
	ac.add(model.Create, source, target, t, map[string]interface{}{
		constants.RelRawText: helper.GetNodeContent(t, *ac.fCtx.SourceBytes()),
		constants.RelAstKind: "new_expression",
	})
	// new Foo(...) 的构造器调用。
	if args := node.ChildByFieldName("arguments"); args != nil {
		if target != nil {
			ac.add(model.Call, source, ac.constructorOf(target), node, map[string]interface{}{
				constants.RelRawText: target.Name,
				constants.RelAstKind: "new_expression",
			})
		}
	}
}

// constructorOf 构造构造器元素（类型 → Method 语义）。
func (ac *actionCtx) constructorOf(t *model.CodeElement) *model.CodeElement {
	if t == nil {
		return nil
	}
	if t.Kind == model.Class || t.Kind == model.Struct {
		return &model.CodeElement{
			Kind:           model.Method,
			Name:           t.Name,
			QualifiedName:  helper.BuildQN(t.QualifiedName, t.Name),
			PackageName:    t.PackageName,
			IsFormExternal: t.IsFormExternal,
		}
	}
	return t
}

// handleCast 处理强转。
func (ac *actionCtx) handleCast(node coreast.Node) {
	t := node.ChildByFieldName("type")
	if t == nil {
		return
	}
	source := ac.sourceOf(node)
	if source == nil {
		return
	}
	target := ac.resolver.ResolveAction(ac.gc, ac.fCtx, t, model.Cast)
	// (void)x 是抑制未使用告警的惯用法，不是真实类型依赖。
	if target != nil && target.QualifiedName == "void" {
		return
	}
	ac.add(model.Cast, source, target, t, map[string]interface{}{
		constants.RelRawText: helper.GetNodeContent(t, *ac.fCtx.SourceBytes()),
		constants.RelAstKind: "cast_expression",
	})
}

// handleAssign 处理赋值表达式。
func (ac *actionCtx) handleAssign(node coreast.Node) {
	left := node.ChildByFieldName("left")
	if left == nil {
		return
	}
	source := ac.sourceOf(node)
	if source == nil {
		return
	}
	target := ac.resolver.ResolveAction(ac.gc, ac.fCtx, left, model.Assign)
	op := ""
	if o := node.ChildByFieldName("operator"); o != nil {
		op = o.Utf8Text(*ac.fCtx.SourceBytes())
	}
	ac.add(model.Assign, source, target, left, map[string]interface{}{
		constants.RelRawText:        helper.GetNodeContent(left, *ac.fCtx.SourceBytes()),
		constants.RelAstKind:        "assignment_expression",
		constants.RelAssignOperator: op,
	})
}

// handleInit 处理变量初始化（int x = expr; / Point p{...}）。
func (ac *actionCtx) handleInit(node coreast.Node) {
	decl := node.ChildByFieldName("declarator")
	value := node.ChildByFieldName("value")
	if decl == nil || value == nil {
		return
	}
	source := ac.sourceOf(node)
	if source == nil {
		return
	}
	target := ac.resolver.ResolveAction(ac.gc, ac.fCtx, decl, model.Assign)
	ac.add(model.Assign, source, target, decl, map[string]interface{}{
		constants.RelRawText:             helper.GetNodeContent(decl, *ac.fCtx.SourceBytes()),
		constants.RelAstKind:             "init_declarator",
		constants.RelAssignIsInitializer: true,
	})
}

// handleThrow 处理 throw 语句。
func (ac *actionCtx) handleThrow(node coreast.Node) {
	var arg coreast.Node
	for i := 0; i < node.NamedChildCount(); i++ {
		arg = node.NamedChild(i)
		break
	}
	if arg == nil {
		return
	}
	source := ac.sourceOf(node)
	if source == nil {
		return
	}
	target := ac.resolver.ResolveAction(ac.gc, ac.fCtx, arg, model.Throw)
	ac.add(model.Throw, source, target, arg, map[string]interface{}{
		constants.RelRawText: helper.GetNodeContent(arg, *ac.fCtx.SourceBytes()),
		constants.RelAstKind: "throw_statement",
	})
}

// handleFieldInitializer 处理构造器成员初始化列表（: a(x), b(y)）。
// 语义为对成员字段的赋值。
func (ac *actionCtx) handleFieldInitializer(node coreast.Node) {
	field := node.NamedChild(0)
	if field == nil {
		return
	}
	source := ac.sourceOf(node)
	if source == nil {
		return
	}
	target := ac.resolver.ResolveAction(ac.gc, ac.fCtx, field, model.Assign)
	ac.add(model.Assign, source, target, field, map[string]interface{}{
		constants.RelRawText:             helper.GetNodeContent(field, *ac.fCtx.SourceBytes()),
		constants.RelAstKind:             "field_initializer",
		constants.RelAssignIsInitializer: true,
	})
}

// handleUse 处理标识符使用（排除定义点/调用名/字段名/类型位置）。
func (ac *actionCtx) handleUse(node coreast.Node) {
	if !isUsePosition(node) {
		return
	}
	source := ac.sourceOf(node)
	if source == nil {
		return
	}
	target := ac.resolver.ResolveAction(ac.gc, ac.fCtx, node, model.Use)
	if target == nil {
		return
	}
	switch target.Kind {
	case model.Variable, model.Field, model.EnumConstant:
	default:
		return
	}
	ac.add(model.Use, source, target, node, map[string]interface{}{
		constants.RelRawText: helper.GetNodeContent(node, *ac.fCtx.SourceBytes()),
		constants.RelAstKind: "identifier",
	})
}

// handleUseComposite 处理复合左值的使用（w.count / Color::RED）。
// 这类节点由 field_expression / qualified_identifier 整体解析，其内部 identifier 已被
// isUsePosition 排除，故需在此单独处理。
func (ac *actionCtx) handleUseComposite(node coreast.Node) {
	if !isCompositeUsePosition(node) {
		return
	}
	source := ac.sourceOf(node)
	if source == nil {
		return
	}
	target := ac.resolver.ResolveAction(ac.gc, ac.fCtx, node, model.Use)
	if target == nil {
		return
	}
	switch target.Kind {
	case model.Variable, model.Field, model.EnumConstant:
	default:
		return
	}
	if source.QualifiedName == target.QualifiedName {
		return
	}
	ac.add(model.Use, source, target, node, map[string]interface{}{
		constants.RelRawText: helper.GetNodeContent(node, *ac.fCtx.SourceBytes()),
		constants.RelAstKind: node.Kind(),
	})
}

// isCompositeUsePosition 判定复合左值节点是否处于「使用」位置。
func isCompositeUsePosition(node coreast.Node) bool {
	p := node.Parent()
	if p == nil {
		return false
	}
	switch p.Kind() {
	case "assignment_expression":
		// 赋值左侧由 handleAssign 处理，避免重复。
		if l := p.ChildByFieldName("left"); l != nil && l.Equal(node) {
			return false
		}
	case "init_declarator":
		// 声明器位置（变量名）非使用；value 位置（初始化表达式）是使用。
		if d := p.ChildByFieldName("declarator"); d != nil && d.Equal(node) {
			return false
		}
	case "declaration", "field_declaration", "parameter_declaration",
		"type_definition", "alias_declaration", "qualified_identifier",
		"call_expression", "field_expression", "template_argument_list",
		"base_class_clause", "namespace_definition", "using_declaration":
		return false
	}
	return true
}

// isUsePosition 判定 identifier 是否处于「使用」位置。
func isUsePosition(node coreast.Node) bool {
	p := node.Parent()
	if p == nil {
		return false
	}
	// 声明点（declarator 的 name）。
	if name := p.ChildByFieldName("name"); name != nil && name.Equal(node) {
		return false
	}
	switch p.Kind() {
	case "declaration", "field_declaration", "parameter_declaration", "function_declarator",
		"pointer_declarator", "reference_declarator", "array_declarator", "init_declarator",
		"type_definition", "alias_declaration", "preproc_def", "preproc_function_def",
		"qualified_identifier", "namespace_definition", "template_parameter_list",
		"struct_specifier", "class_specifier", "enum_specifier", "enumerator",
		"call_expression", "field_expression", "lambda_expression", "case_statement",
		"preproc_ifdef", "preproc_if", "preproc_elif", "preproc_defined", "preproc_call":
		return false
	}
	// 类型位置（type 字段）。
	if t := p.ChildByFieldName("type"); t != nil && t.Equal(node) {
		return false
	}
	return true
}

// sourceOf 向上定位关系发起者（函数/方法/Lambda/块/字段/全局变量）。
// 局部变量不作为发起者：其声明内的动作（初始化表达式）归属所在函数，
// 与 Java「局部变量不是 source kind」语义一致。
func (ac *actionCtx) sourceOf(node coreast.Node) *model.CodeElement {
	var best *model.CodeElement
	bestSpan := -1
	start, end := node.StartByte(), node.EndByte()
	for _, entry := range ac.fCtx.Definitions() {
		elem := entry.Element
		if !isSourceKind(elem) {
			continue
		}
		if entry.Node == nil {
			continue
		}
		s, e := entry.Node.StartByte(), entry.Node.EndByte()
		if s <= start && end <= e {
			if span := e - s; bestSpan < 0 || span < bestSpan {
				bestSpan, best = span, elem
			}
		}
	}
	return best
}

func isSourceKind(elem *model.CodeElement) bool {
	switch elem.Kind {
	case model.Function, model.Method, model.Lambda, model.ScopeBlock, model.Field:
		return true
	case model.Variable:
		// 仅全局变量可作为关系发起者（其初始化表达式归属变量自身）。
		if elem.Extra == nil {
			return false
		}
		isGlobal, _ := elem.Extra.Mores[constants.VariableIsGlobal].(bool)
		return isGlobal
	}
	return false
}

// =============================================================================
// 辅助
// =============================================================================

// paramTypeOnly 从 "int x" 提取 "int"。
func paramTypeOnly(p string) string {
	p = helper.NormalizeTypeText(p)
	if i := strings.LastIndexAny(p, " \t"); i > 0 {
		candidate := strings.TrimSpace(p[i+1:])
		if candidate != "" && !strings.ContainsAny(candidate, "*&[]<>,()") {
			return strings.TrimSpace(p[:i])
		}
	}
	return p
}

// templateArgs 提取类型文本中的模板实参列表（顶层逗号切分）。
func templateArgs(raw string) []string {
	start := strings.Index(raw, "<")
	if start < 0 {
		return nil
	}
	depth := 0
	end := -1
	for i := start; i < len(raw); i++ {
		switch raw[i] {
		case '<':
			depth++
		case '>':
			depth--
			if depth == 0 {
				end = i
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		return nil
	}
	inner := raw[start+1 : end]
	var out []string
	depth = 0
	last := 0
	for i := 0; i < len(inner); i++ {
		switch inner[i] {
		case '<':
			depth++
		case '>':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(inner[last:i]))
				last = i + 1
			}
		}
	}
	if last < len(inner) {
		out = append(out, strings.TrimSpace(inner[last:]))
	}
	return out
}

func isFunctionKind(k model.ElementKind) bool {
	switch k {
	case model.Function, model.Method, model.FunctionDecl, model.MethodDecl:
		return true
	}
	return false
}
