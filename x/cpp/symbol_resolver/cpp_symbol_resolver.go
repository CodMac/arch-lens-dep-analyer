// Package symbol_resolver 实现 C/C++ 语言的 core.SymbolResolver 接口。
// 职责：统一动作解析入口（Call/Create/Cast/Assign/Use/Throw），
// 以及类型/变量/函数的基础解析（供 binder/extractor 共享）。
//
// 与 Java 三段式的差异：C/C++ 表达式形态更简单（无链式方法调用语法糖），
// 采用「节点形态分类 + 作用域链查找」的两段式解析。
package symbol_resolver

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/helper"
)

// SymbolResolver 实现 core.SymbolResolver。
type SymbolResolver struct{}

// New 创建 C/C++ SymbolResolver。
func New() *SymbolResolver { return &SymbolResolver{} }

// =============================================================================
// core.SymbolResolver 接口实现
// =============================================================================

// BuildQualifiedName 拼接父符号 QN 与子符号名称为完整 QN（:: 分隔）。
func (r *SymbolResolver) BuildQualifiedName(parentQN, name string) string {
	return helper.BuildQN(parentQN, name)
}

// RegisterPackage 注册命名空间层级（C++ 按 :: 拆分；C 无 namespace 时为空操作）。
func (r *SymbolResolver) RegisterPackage(gc core.GlobalContext, packageName string) {
	if packageName == "" {
		return
	}
	parts := helper.SplitQN(packageName)
	var cur []string
	for _, p := range parts {
		if p == "" || p == "$anon" {
			continue
		}
		cur = append(cur, p)
		qn := strings.Join(cur, helper.QNSep)
		if _, ok := gc.FindByQualifiedName(qn); !ok {
			gc.AddDefinition(&core.DefinitionEntry{
				Element: &model.CodeElement{
					Kind: model.Namespace, Name: p, QualifiedName: qn, IsFormSource: true,
				},
			})
		}
	}
}

// IsPrimitive 判断类型名是否为 C/C++ 基本类型。
func (r *SymbolResolver) IsPrimitive(typeName string) bool {
	return helper.IsPrimitive(helper.BaseTypeName(typeName))
}

// ResolveType 解析类型符号；未找到返回外部实体。
func (r *SymbolResolver) ResolveType(gc core.GlobalContext, fc core.FileContext, symbol string, kind model.ElementKind) *model.CodeElement {
	raw := helper.NormalizeTypeText(symbol)
	if raw == "" || raw == "?" {
		return nil
	}
	base := helper.BaseTypeName(raw)
	if base == "" {
		return nil
	}
	if r.IsPrimitive(base) {
		return &model.CodeElement{Name: base, QualifiedName: base, Kind: model.Primitive, IsFormExternal: true}
	}
	if helper.IsTypeVariable(base) {
		return nil
	}

	if elem := r.lookupType(gc, fc, base); elem != nil {
		return elem
	}

	// 外部兜底：按 :: 分配命名空间归属。
	return helper.NewExternalElement(kind, base, base)
}

// ResolveVar 解析变量/字段/枚举常量。
func (r *SymbolResolver) ResolveVar(gc core.GlobalContext, fc core.FileContext, node coreast.Node, receiver, symbol string) *model.CodeElement {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return nil
	}
	if receiver != "" {
		if recvType := r.receiverType(gc, fc, node, receiver); recvType != nil {
			if elem := r.lookupMember(gc, fc, recvType, symbol, model.Field); elem != nil {
				return elem
			}
		}
	}
	return r.lookupScopedSymbol(gc, fc, node, symbol)
}

// ResolveFunc 解析函数/方法。
func (r *SymbolResolver) ResolveFunc(gc core.GlobalContext, fc core.FileContext, node coreast.Node, receiver, symbol string) *model.CodeElement {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return nil
	}
	argCount := r.argCount(node)

	if receiver != "" {
		if recvType := r.receiverType(gc, fc, node, receiver); recvType != nil {
			if elem := r.lookupMemberFunc(gc, fc, recvType, symbol, argCount); elem != nil {
				return elem
			}
			// 外部接收者类型：产外部方法。
			return helper.NewExternalElement(model.Method, symbol, helper.BuildQN(recvType.QualifiedName, symbol))
		}
	}

	if elem := r.lookupScopedFunc(gc, fc, node, symbol, argCount); elem != nil {
		return elem
	}

	// 变量持有 lambda（auto f = [](){...}; f();）：解析到 Lambda 元素。
	if v := r.lookupScopedSymbol(gc, fc, node, symbol); v != nil && v.Extra != nil {
		if lambdaQN, ok := v.Extra.Mores[constants.VariableLambdaQN].(string); ok && lambdaQN != "" {
			if entry, found := gc.FindByQualifiedName(lambdaQN); found {
				return entry.Element
			}
		}
	}
	return nil
}

// ResolveAction 统一动作解析入口。
func (r *SymbolResolver) ResolveAction(gc core.GlobalContext, fc core.FileContext, node coreast.Node, relType model.DependencyType) *model.CodeElement {
	if node == nil {
		return nil
	}
	switch relType {
	case model.Call:
		return r.resolveCall(gc, fc, node)
	case model.Create:
		return r.resolveCreate(gc, fc, node)
	case model.Cast:
		return r.resolveCast(gc, fc, node)
	case model.Assign:
		return r.resolveAssign(gc, fc, node)
	case model.Use:
		return r.resolveUse(gc, fc, node)
	case model.Throw:
		return r.resolveThrow(gc, fc, node)
	}
	return nil
}

// =============================================================================
// 动作解析
// =============================================================================

// resolveCall 解析调用目标。node 为 call_expression 的 function 字段节点。
func (r *SymbolResolver) resolveCall(gc core.GlobalContext, fc core.FileContext, node coreast.Node) *model.CodeElement {
	src := *fc.SourceBytes()
	recv, name := splitCallee(node, src)
	if name == "" {
		return nil
	}
	// 构造函数调用（C++ new 的 function 位置不会出现；此处为普通调用）。
	// static_cast<T>(x) / dynamic_cast / const_cast / reinterpret_cast：类型转换语义。
	if recv == "" && isCastKeyword(name) {
		if t := castTemplateArg(node, *fc.SourceBytes()); t != "" {
			return r.ResolveType(gc, fc, t, model.Class)
		}
	}
	if elem := r.ResolveFunc(gc, fc, node, recv, name); elem != nil {
		return elem
	}
	// 兜底：外部函数。
	qn := name
	if recv != "" {
		qn = helper.BuildQN(recv, name)
	}
	return helper.NewExternalElement(model.Function, name, qn)
}

// isCastKeyword 判断是否为 C++ 显式转换关键字。
func isCastKeyword(name string) bool {
	switch name {
	case "static_cast", "dynamic_cast", "const_cast", "reinterpret_cast":
		return true
	}
	return false
}

// castTemplateArg 取 template_function 的模板实参类型文本。
func castTemplateArg(node coreast.Node, src []byte) string {
	if node == nil {
		return ""
	}
	if node.Kind() == "template_function" {
		if args := node.ChildByFieldName("arguments"); args != nil {
			for i := 0; i < args.NamedChildCount(); i++ {
				ch := args.NamedChild(i)
				if name := typeNodeName(ch, src); name != "" {
					return name
				}
			}
		}
	}
	return ""
}

// resolveCreate 解析实例创建目标。node 为 new_expression 的 type 字段节点。
func (r *SymbolResolver) resolveCreate(gc core.GlobalContext, fc core.FileContext, node coreast.Node) *model.CodeElement {
	name := typeNodeName(node, *fc.SourceBytes())
	if name == "" {
		return nil
	}
	return r.ResolveType(gc, fc, name, model.Class)
}

// resolveCast 解析强转目标类型。node 为 cast_expression 的 type 字段节点。
func (r *SymbolResolver) resolveCast(gc core.GlobalContext, fc core.FileContext, node coreast.Node) *model.CodeElement {
	name := typeNodeName(node, *fc.SourceBytes())
	if name == "" {
		return nil
	}
	return r.ResolveType(gc, fc, name, model.Class)
}

// resolveAssign 解析赋值目标。node 为 assignment_expression 的 left 字段节点。
func (r *SymbolResolver) resolveAssign(gc core.GlobalContext, fc core.FileContext, node coreast.Node) *model.CodeElement {
	return r.resolveLValue(gc, fc, node)
}

// resolveUse 解析使用目标。
func (r *SymbolResolver) resolveUse(gc core.GlobalContext, fc core.FileContext, node coreast.Node) *model.CodeElement {
	return r.resolveLValue(gc, fc, node)
}

// resolveLValue 解析可寻址表达式（identifier / field_expression / qualified_identifier / pointer_expression）。
func (r *SymbolResolver) resolveLValue(gc core.GlobalContext, fc core.FileContext, node coreast.Node) *model.CodeElement {
	if node == nil {
		return nil
	}
	src := *fc.SourceBytes()
	switch node.Kind() {
	case "identifier", "field_identifier":
		return r.ResolveVar(gc, fc, node, "", node.Utf8Text(src))
	case "field_expression":
		arg := node.ChildByFieldName("argument")
		field := node.ChildByFieldName("field")
		if arg == nil || field == nil {
			return nil
		}
		return r.ResolveVar(gc, fc, node, helper.GetNodeContent(arg, src), helper.GetNodeContent(field, src))
	case "qualified_identifier":
		recv, name := splitQualified(node, src)
		if name == "" {
			return nil
		}
		return r.ResolveVar(gc, fc, node, recv, name)
	case "pointer_expression", "parenthesized_expression":
		arg := node.ChildByFieldName("argument")
		if arg == nil {
			for i := 0; i < node.NamedChildCount(); i++ {
				arg = node.NamedChild(i)
				break
			}
		}
		return r.resolveLValue(gc, fc, arg)
	case "subscript_expression":
		arg := node.ChildByFieldName("argument")
		return r.resolveLValue(gc, fc, arg)
	}
	return nil
}

// resolveThrow 解析抛出异常的类型。node 为 throw_statement 的 argument 字段节点。
func (r *SymbolResolver) resolveThrow(gc core.GlobalContext, fc core.FileContext, node coreast.Node) *model.CodeElement {
	if node == nil {
		return nil
	}
	src := *fc.SourceBytes()
	// throw new Foo() / throw Foo(...) / throw var
	switch node.Kind() {
	case "new_expression":
		if t := node.ChildByFieldName("type"); t != nil {
			return r.ResolveType(gc, fc, typeNodeName(t, src), model.Class)
		}
	case "call_expression":
		if f := node.ChildByFieldName("function"); f != nil {
			if e := r.resolveCall(gc, fc, f); e != nil {
				return e
			}
		}
	}
	// 变量形态：取其声明类型。
	if v := r.resolveLValue(gc, fc, node); v != nil && v.Extra != nil {
		if t, ok := v.Extra.Mores[constants.VariableTypeWithQN].(string); ok && t != "" {
			return r.ResolveType(gc, fc, t, model.Class)
		}
		if t, ok := v.Extra.Mores[constants.FieldTypeWithQN].(string); ok && t != "" {
			return r.ResolveType(gc, fc, t, model.Class)
		}
	}
	return nil
}

// =============================================================================
// 类型/成员查找
// =============================================================================

// lookupType 按限定名/短名查找类型元素。
func (r *SymbolResolver) lookupType(gc core.GlobalContext, fc core.FileContext, name string) *model.CodeElement {
	if entry, ok := gc.FindByQualifiedName(name); ok && isTypeKind(entry.Element.Kind) {
		return entry.Element
	}
	// 相对限定名：namespace app 内的 core::Base 实为 app::core::Base。
	if strings.Contains(name, helper.QNSep) {
		for _, ns := range fc.Namespaces() {
			if cand := helper.BuildQN(ns, name); cand != name {
				if entry, ok := gc.FindByQualifiedName(cand); ok && isTypeKind(entry.Element.Kind) {
					return entry.Element
				}
			}
		}
	}
	// 文件内短名优先（C 无 ns 时尤其重要）。
	if entries, ok := fc.FindByShortName(name); ok {
		for _, e := range entries {
			if isTypeKind(e.Element.Kind) {
				return e.Element
			}
		}
	}
	// 全局短名（跨文件）。
	for _, def := range gc.Definitions() {
		if def.Element.Name == name && isTypeKind(def.Element.Kind) {
			return def.Element
		}
	}
	return nil
}

// lookupMember 在类型（含基类）中查找字段/枚举常量。
func (r *SymbolResolver) lookupMember(gc core.GlobalContext, fc core.FileContext, owner *model.CodeElement, name string, want model.ElementKind) *model.CodeElement {
	for _, o := range r.typeChain(gc, fc, owner, 0) {
		qn := helper.BuildQN(o.QualifiedName, name)
		if entry, ok := gc.FindByQualifiedName(qn); ok {
			switch entry.Element.Kind {
			case model.Field, model.Variable, model.EnumConstant:
				return entry.Element
			}
		}
	}
	return nil
}

// lookupMemberFunc 在类型（含基类）中查找方法。
func (r *SymbolResolver) lookupMemberFunc(gc core.GlobalContext, fc core.FileContext, owner *model.CodeElement, name string, argCount int) *model.CodeElement {
	for _, o := range r.typeChain(gc, fc, owner, 0) {
		qn := helper.BuildQN(o.QualifiedName, name)
		if entries, ok := gc.FindMethodByNoParamsQN(qn); ok {
			if e := pickOverload(entries, argCount); e != nil {
				return e.Element
			}
		}
	}
	return nil
}

// typeChain 返回类型自身及其基类链（广度优先，去环）。
func (r *SymbolResolver) typeChain(gc core.GlobalContext, fc core.FileContext, owner *model.CodeElement, depth int) []*model.CodeElement {
	if owner == nil || depth > 8 {
		return nil
	}
	out := []*model.CodeElement{owner}
	if owner.Extra == nil {
		return out
	}
	supers, _ := owner.Extra.Mores[constants.ClassSuperClass].([]string)
	for _, s := range supers {
		base := r.lookupType(gc, fc, helper.BaseTypeName(s))
		if base == nil || base.IsFormExternal || base.QualifiedName == owner.QualifiedName {
			continue
		}
		out = append(out, r.typeChain(gc, fc, base, depth+1)...)
	}
	return out
}

// receiverType 解析接收者表达式对应的类型元素。
func (r *SymbolResolver) receiverType(gc core.GlobalContext, fc core.FileContext, node coreast.Node, receiver string) *model.CodeElement {
	receiver = strings.TrimSpace(receiver)
	if receiver == "" {
		return nil
	}
	// this / *this
	if receiver == "this" || receiver == "*this" {
		if owner := enclosingType(fc, node); owner != nil {
			return owner
		}
		return nil
	}
	// 类型名静态访问：Type::member / Type::staticMethod
	if elem := r.lookupType(gc, fc, helper.BaseTypeName(receiver)); elem != nil {
		return elem
	}
	// 变量/字段：取其声明类型。
	if v := r.ResolveVar(gc, fc, node, "", receiver); v != nil && v.Extra != nil {
		for _, key := range []string{constants.VariableTypeWithQN, constants.FieldTypeWithQN, constants.VariableRawType, constants.FieldRawType} {
			if t, ok := v.Extra.Mores[key].(string); ok && t != "" {
				if elem := r.lookupType(gc, fc, helper.BaseTypeName(t)); elem != nil {
					return elem
				}
			}
		}
	}
	return nil
}

// enclosingType 返回节点所属的类/结构体元素。
func enclosingType(fc core.FileContext, node coreast.Node) *model.CodeElement {
	if node == nil {
		return nil
	}
	start, end := node.StartByte(), node.EndByte()
	var best *model.CodeElement
	bestSpan := -1
	for _, entry := range fc.Definitions() {
		k := entry.Element.Kind
		if k != model.Class && k != model.Struct {
			continue
		}
		if entry.Node == nil {
			continue
		}
		s, e := entry.Node.StartByte(), entry.Node.EndByte()
		if s <= start && end <= e {
			if span := e - s; bestSpan < 0 || span < bestSpan {
				bestSpan, best = span, entry.Element
			}
		}
	}
	return best
}

// =============================================================================
// 作用域链查找
// =============================================================================

// scopeChain 返回节点的词法作用域 QN 链（最内层 → 最外层 → ""）。
func (r *SymbolResolver) scopeChain(fc core.FileContext, node coreast.Node) []string {
	var chain []string
	owner := enclosingScope(fc, node)
	if owner != nil {
		chain = append(chain, owner.QualifiedName)
		for scope := helper.PkgOf(owner.QualifiedName); scope != ""; {
			chain = append(chain, scope)
			next := helper.PkgOf(scope)
			if next == scope {
				break
			}
			scope = next
		}
	}
	chain = append(chain, "")
	return chain
}

// enclosingScope 返回节点所属的最内层作用域元素（函数/方法/类/Lambda/块）。
func enclosingScope(fc core.FileContext, node coreast.Node) *model.CodeElement {
	if node == nil {
		return nil
	}
	start, end := node.StartByte(), node.EndByte()
	var best *model.CodeElement
	bestSpan := -1
	for _, entry := range fc.Definitions() {
		if !helper.IsScopeContainer(entry.Element.Kind) {
			continue
		}
		if entry.Node == nil {
			continue
		}
		s, e := entry.Node.StartByte(), entry.Node.EndByte()
		if s <= start && end <= e {
			if span := e - s; bestSpan < 0 || span < bestSpan {
				bestSpan, best = span, entry.Element
			}
		}
	}
	return best
}

// lookupScopedSymbol 沿作用域链查找变量/字段/枚举常量。
func (r *SymbolResolver) lookupScopedSymbol(gc core.GlobalContext, fc core.FileContext, node coreast.Node, name string) *model.CodeElement {
	for _, scope := range r.scopeChain(fc, node) {
		qn := helper.BuildQN(scope, name)
		if entry, ok := gc.FindByQualifiedName(qn); ok {
			switch entry.Element.Kind {
			case model.Variable, model.Field, model.EnumConstant:
				return entry.Element
			}
		}
	}
	// 文件内短名（同文件全局变量/字段）。
	if entries, ok := fc.FindByShortName(name); ok {
		for _, e := range entries {
			switch e.Element.Kind {
			case model.Variable, model.Field, model.EnumConstant:
				return e.Element
			}
		}
	}
	// 全局短名（跨文件全局变量/枚举常量）。
	for _, def := range gc.Definitions() {
		if def.Element.Name != name {
			continue
		}
		switch def.Element.Kind {
		case model.EnumConstant, model.Field, model.Variable:
			return def.Element
		}
	}
	return nil
}

// lookupScopedFunc 沿作用域链查找函数/方法。
func (r *SymbolResolver) lookupScopedFunc(gc core.GlobalContext, fc core.FileContext, node coreast.Node, name string, argCount int) *model.CodeElement {
	for _, scope := range r.scopeChain(fc, node) {
		qn := helper.BuildQN(scope, name)
		if entries, ok := gc.FindMethodByNoParamsQN(qn); ok {
			if e := pickOverload(entries, argCount); e != nil {
				return e.Element
			}
		}
	}
	// 文件内短名（static 函数）。
	if entries, ok := fc.FindByShortName(name); ok {
		for _, e := range entries {
			if isFunctionKind(e.Element.Kind) {
				return e.Element
			}
		}
	}
	// 全局短名。
	for _, def := range gc.Definitions() {
		if def.Element.Name == name && isFunctionKind(def.Element.Kind) {
			return def.Element
		}
	}
	return nil
}

// pickOverload 按实参个数挑选重载（定义优先）。
func pickOverload(entries []*core.DefinitionEntry, argCount int) *core.DefinitionEntry {
	if len(entries) == 0 {
		return nil
	}
	var fallback *core.DefinitionEntry
	for _, e := range entries {
		params, _ := e.Element.Extra.Mores[constants.FunctionParameters].([]string)
		if len(params) == argCount {
			if !isDeclKind(e.Element.Kind) {
				return e
			}
			if fallback == nil {
				fallback = e
			}
		}
	}
	if fallback != nil {
		return fallback
	}
	// 无参数信息（或数量不匹配）：定义优先，否则首个。
	for _, e := range entries {
		if !isDeclKind(e.Element.Kind) {
			return e
		}
	}
	return entries[0]
}

// argCount 返回调用节点的实参个数。
func (r *SymbolResolver) argCount(node coreast.Node) int {
	call := findAncestorOrSelf(node, "call_expression", "new_expression")
	if call == nil {
		return 0
	}
	args := call.ChildByFieldName("arguments")
	if args == nil {
		return 0
	}
	return args.NamedChildCount()
}

// =============================================================================
// 节点工具
// =============================================================================

// splitCallee 从调用目标节点拆出接收者与函数名。
func splitCallee(node coreast.Node, src []byte) (recv, name string) {
	if node == nil {
		return "", ""
	}
	switch node.Kind() {
	case "identifier", "field_identifier", "operator_name", "destructor_name":
		return "", strings.TrimSpace(node.Utf8Text(src))
	case "qualified_identifier":
		return splitQualified(node, src)
	case "field_expression":
		arg := node.ChildByFieldName("argument")
		field := node.ChildByFieldName("field")
		if field == nil {
			return "", ""
		}
		return helper.GetNodeContent(arg, src), helper.GetNodeContent(field, src)
	case "template_function", "template_method":
		if n := node.ChildByFieldName("name"); n != nil {
			return splitCallee(n, src)
		}
	case "parenthesized_expression":
		for i := 0; i < node.NamedChildCount(); i++ {
			return splitCallee(node.NamedChild(i), src)
		}
	}
	return "", strings.TrimSpace(node.Utf8Text(src))
}

// splitQualified 拆解 qualified_identifier 的 scope 与 name。
func splitQualified(node coreast.Node, src []byte) (recv, name string) {
	scope := node.ChildByFieldName("scope")
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return "", ""
	}
	if scope != nil {
		recv = helper.GetNodeContent(scope, src)
	}
	if nameNode.Kind() == "qualified_identifier" {
		innerRecv, innerName := splitQualified(nameNode, src)
		if recv == "" {
			recv = innerRecv
		} else if innerRecv != "" {
			recv = helper.BuildQN(recv, innerRecv)
		}
		return recv, innerName
	}
	return recv, strings.TrimSpace(nameNode.Utf8Text(src))
}

// typeNodeName 提取类型节点的名称文本（template_type 取模板名）。
func typeNodeName(node coreast.Node, src []byte) string {
	if node == nil {
		return ""
	}
	switch node.Kind() {
	case "type_descriptor":
		if t := node.ChildByFieldName("type"); t != nil {
			return typeNodeName(t, src)
		}
	case "template_type":
		if n := node.ChildByFieldName("name"); n != nil {
			return helper.GetNodeContent(n, src)
		}
	case "qualified_identifier", "type_identifier", "primitive_type", "sized_type_specifier":
		return helper.GetNodeContent(node, src)
	}
	return helper.GetNodeContent(node, src)
}

// findAncestorOrSelf 向上查找指定 kind 的节点。
func findAncestorOrSelf(node coreast.Node, kinds ...string) coreast.Node {
	for curr := node; curr != nil; curr = curr.Parent() {
		for _, k := range kinds {
			if curr.Kind() == k {
				return curr
			}
		}
		if strings.HasSuffix(curr.Kind(), "_statement") {
			return nil
		}
	}
	return nil
}

func isTypeKind(k model.ElementKind) bool {
	switch k {
	case model.Class, model.Struct, model.Enum, model.Type, model.Namespace, model.Interface:
		return true
	}
	return false
}

func isFunctionKind(k model.ElementKind) bool {
	switch k {
	case model.Function, model.Method, model.FunctionDecl, model.MethodDecl:
		return true
	}
	return false
}

func isDeclKind(k model.ElementKind) bool {
	return k == model.FunctionDecl || k == model.MethodDecl
}
