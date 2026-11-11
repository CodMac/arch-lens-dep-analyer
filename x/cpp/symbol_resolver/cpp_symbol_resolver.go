// Package symbol_resolver 实现 C/C++ 语言的 core.SymbolResolver 接口。
// 职责：统一动作解析入口（Call/Create/Cast/Assign/Use/Throw），
// 以及类型/变量/函数的基础解析（供 binder/extractor 共享）。
//
// 解析采用**两段式**（与 Java 的「Context → Segmenter → ChainResolver」三段式相对）：
//
//	段 1 归一化：把任意动作 AST 节点折叠为统一的 expr 描述
//	             （form + recv + name + typeText + argc），
//	             吸收 field_expression / qualified_identifier / call_expression /
//	             new_expression / cast_expression / template_function 等形态差异。
//	段 2 查找：只依赖 expr，沿「词法作用域链 + 类型继承链」查找目标符号；
//	             需要接收者类型时递归 typeOf(recv)，实现链式调用的类型传播。
//
// 之所以不需要 Java 的三段式：C/C++ 的 AST 已把 receiver 与成员名分开
// （call_expression.function 为 field_expression，其 argument 即 receiver），
// 且无 super / Xxx.class / 方法引用 / 泛型实例化等语法。
package symbol_resolver

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/constants"
	cppctx "github.com/CodMac/arch-lens-dep-analyer/x/cpp/context"
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
	e := &expr{form: formIdent, node: node, name: symbol}
	if receiver != "" {
		e.form = formMember
		e.recv = &expr{form: formText, node: node, text: receiver}
	}
	return r.resolveValue(gc, fc, e)
}

// ResolveFunc 解析函数/方法。
func (r *SymbolResolver) ResolveFunc(gc core.GlobalContext, fc core.FileContext, node coreast.Node, receiver, symbol string) *model.CodeElement {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return nil
	}
	e := &expr{form: formIdent, node: node, name: symbol, argc: r.argCount(node)}
	if receiver != "" {
		e.form = formMember
		e.recv = &expr{form: formText, node: node, text: receiver}
	}
	return r.resolveFunc(gc, fc, e)
}

// ResolveAction 统一动作解析入口：段 1 归一化 → 段 2 查找。
func (r *SymbolResolver) ResolveAction(gc core.GlobalContext, fc core.FileContext, node coreast.Node, relType model.DependencyType) *model.CodeElement {
	if node == nil {
		return nil
	}
	src := *fc.SourceBytes()
	e := r.normalize(node, src)
	if e == nil {
		return nil
	}

	switch relType {
	case model.Call:
		// static_cast<T>(x) 等显式转换：语义为 CAST 而非 CALL。
		if e.form == formCast && e.typeText != "" {
			return r.ResolveType(gc, fc, e.typeText, model.Class)
		}
		// extractor 传入的是 call_expression 的 function 字段（非整个调用节点），
		// 此时归一化结果不含实参个数，需从祖先调用节点补齐（重载挑选依赖它）。
		if e.form != formCall {
			e.argc = r.argCount(node)
		}
		if elem := r.resolveExpr(gc, fc, e, wantFunc); elem != nil {
			return elem
		}
		// 可调用对象（operator() 重载 / 函数指针变量）不建模为外部函数调用。
		if r.lookupLocalVariable(gc, fc, e.node, e.name) != nil {
			return nil
		}
		// 构造器调用（T(...) 临时对象）：按类型产构造器元素。
		if t := r.lookupType(gc, fc, e.name); t != nil {
			return &model.CodeElement{
				Kind:          model.Method,
				Name:          t.Name,
				QualifiedName: helper.BuildQN(t.QualifiedName, t.Name),
				PackageName:   t.PackageName,
			}
		}
		// 兜底：外部函数（保持原有 QN 形态：接收者原文::函数名）。
		qn := e.name
		if txt := exprText(e.recv, src); txt != "" {
			qn = helper.BuildQN(txt, e.name)
		}
		return helper.NewExternalElement(model.Function, e.name, qn)

	case model.Create, model.Cast:
		return r.resolveExpr(gc, fc, e, wantType)

	case model.Assign, model.Use:
		return r.resolveExpr(gc, fc, e, wantValue)

	case model.Throw:
		// throw new T() → 类型；throw f() → 函数；throw var → 变量声明类型。
		switch e.form {
		case formNew:
			return r.resolveExpr(gc, fc, e, wantType)
		case formCall:
			// throw T(...)：被调用者是类型名时取类型（临时对象）。
			if t := r.lookupType(gc, fc, e.name); t != nil {
				return t
			}
			return r.resolveExpr(gc, fc, e, wantFunc)
		default:
			if t := r.typeOf(gc, fc, e); t != nil {
				return t
			}
			// 变量声明类型不在源码中（外部类型）时仍产出外部类型元素。
			if v := r.resolveValue(gc, fc, e); v != nil && v.Extra != nil {
				for _, key := range []string{constants.VariableTypeWithQN, constants.FieldTypeWithQN} {
					if raw, ok := v.Extra.Mores[key].(string); ok && raw != "" {
						return r.ResolveType(gc, fc, raw, model.Class)
					}
				}
			}
			return nil
		}
	}
	return nil
}

// =============================================================================
// 段 1：表达式归一化
// =============================================================================

// exprForm 是动作表达式归一化后的形态。
type exprForm int

const (
	formIdent  exprForm = iota // 裸标识符：name
	formMember                 // 成员访问：recv.name / recv->name
	formScoped                 // 限定访问：scope::name
	formCall                   // 调用：callee(...)
	formNew                    // new T(...)
	formCast                   // (T)x / static_cast<T>(x)
	formText                   // 纯文本接收者（公开 SPI 传入 receiver 字符串时）
)

// expr 是动作节点的归一化描述（段 1 产出）。
// 段 2 只依赖本结构，不再关心原始 AST 形态。
type expr struct {
	form     exprForm
	node     coreast.Node // 表达式节点（可为 nil）
	recv     *expr        // 接收者表达式（formMember / formScoped / formCall）
	name     string       // 成员名 / 函数名 / 标识符
	typeText string       // formNew / formCast 的类型文本
	text     string       // formText 的接收者原文
	argc     int          // formCall 的实参个数
}

// normalize 将任意动作节点折叠为统一的 expr（段 1）。
func (r *SymbolResolver) normalize(node coreast.Node, src []byte) *expr {
	if node == nil {
		return nil
	}
	switch node.Kind() {
	case "identifier", "field_identifier", "type_identifier", "namespace_identifier",
		"operator_name", "destructor_name", "primitive_type", "sized_type_specifier":
		return &expr{form: formIdent, node: node, name: strings.TrimSpace(node.Utf8Text(src))}

	case "field_expression":
		field := node.ChildByFieldName("field")
		if field == nil {
			return nil
		}
		return &expr{
			form: formMember,
			node: node,
			recv: r.normalize(node.ChildByFieldName("argument"), src),
			name: helper.GetNodeContent(field, src),
		}

	case "qualified_identifier":
		nameNode := node.ChildByFieldName("name")
		if nameNode == nil {
			return nil
		}
		return &expr{
			form: formScoped,
			node: node,
			recv: r.normalize(node.ChildByFieldName("scope"), src),
			name: strings.TrimSpace(nameNode.Utf8Text(src)),
		}

	case "call_expression":
		fnNode := node.ChildByFieldName("function")
		callee := r.normalize(fnNode, src)
		if callee == nil {
			return nil
		}
		// 立即调用的 lambda：保留 lambda 节点本身，供段 2 直接命中 Lambda 元素。
		if fnNode != nil && fnNode.Kind() == "lambda_expression" {
			callee.form = formCall
			callee.node = fnNode
			callee.name = "lambda"
			if args := node.ChildByFieldName("arguments"); args != nil {
				callee.argc = args.NamedChildCount()
			}
			return callee
		}
		// static_cast<T>(x) 等：保持转换语义，不当作普通调用。
		if callee.form == formCast {
			return callee
		}
		callee.form = formCall
		callee.node = node
		if args := node.ChildByFieldName("arguments"); args != nil {
			callee.argc = args.NamedChildCount()
		}
		return callee

	case "new_expression":
		return &expr{form: formNew, node: node, typeText: typeNodeName(node.ChildByFieldName("type"), src)}

	case "cast_expression":
		return &expr{form: formCast, node: node, typeText: typeNodeName(node.ChildByFieldName("type"), src)}

	case "template_function", "template_method":
		nameNode := node.ChildByFieldName("name")
		name := ""
		if nameNode != nil {
			name = strings.TrimSpace(nameNode.Utf8Text(src))
		}
		if isCastKeyword(name) {
			return &expr{form: formCast, node: node, typeText: castTemplateArg(node, src)}
		}
		if nameNode != nil {
			return r.normalize(nameNode, src)
		}
		return nil

	case "pointer_expression", "parenthesized_expression":
		arg := node.ChildByFieldName("argument")
		if arg == nil {
			for i := 0; i < node.NamedChildCount(); i++ {
				arg = node.NamedChild(i)
				break
			}
		}
		return r.normalize(arg, src)

	case "subscript_expression":
		return r.normalize(node.ChildByFieldName("argument"), src)
	}

	// 未知形态：退化为裸标识符（名称取原文）。
	return &expr{form: formIdent, node: node, name: strings.TrimSpace(node.Utf8Text(src))}
}

// exprText 返回表达式的源码文本（外部兜底 QN 用）。
func exprText(e *expr, src []byte) string {
	if e == nil {
		return ""
	}
	if e.form == formText {
		return e.text
	}
	if e.node != nil {
		return helper.GetNodeContent(e.node, src)
	}
	return e.text
}

// =============================================================================
// 段 2：查找
// =============================================================================

// wantKind 是段 2 的期望解析类别。
type wantKind int

const (
	wantValue wantKind = iota // 变量/字段/枚举常量
	wantFunc                  // 函数/方法
	wantType                  // 类型
)

// resolveExpr 按期望类别解析归一化表达式（段 2 入口）。
func (r *SymbolResolver) resolveExpr(gc core.GlobalContext, fc core.FileContext, e *expr, want wantKind) *model.CodeElement {
	if e == nil {
		return nil
	}
	switch e.form {
	case formNew, formCast:
		if e.typeText == "" {
			return nil
		}
		return r.ResolveType(gc, fc, e.typeText, model.Class)

	case formCall:
		if want == wantType {
			return r.typeOf(gc, fc, e)
		}
		return r.resolveFunc(gc, fc, e)

	case formIdent, formMember, formScoped, formText:
		switch want {
		case wantFunc:
			return r.resolveFunc(gc, fc, e)
		case wantType:
			if t := r.typeOf(gc, fc, e); t != nil {
				return t
			}
			// 未命中源码类型：按原文产外部类型（Create/Cast 语义）。
			raw := e.name
			if e.node != nil {
				raw = helper.GetNodeContent(e.node, *fc.SourceBytes())
			}
			return r.ResolveType(gc, fc, raw, model.Class)
		default:
			return r.resolveValue(gc, fc, e)
		}
	}
	return nil
}

// resolveFunc 解析函数/方法：接收者类型 → 成员方法；否则作用域链 → 自由函数。
func (r *SymbolResolver) resolveFunc(gc core.GlobalContext, fc core.FileContext, e *expr) *model.CodeElement {
	if e == nil || e.name == "" {
		return nil
	}
	// 立即调用的 lambda（[](int v){...}(3)）：直接命中该 Lambda 元素。
	if e.node != nil && e.node.Kind() == "lambda_expression" {
		if elem := lambdaElementOf(fc, e.node); elem != nil {
			return elem
		}
	}
	if recvType := r.typeOf(gc, fc, e.recv); recvType != nil {
		if elem := r.lookupMemberFunc(gc, fc, recvType, e.name, e.argc); elem != nil {
			return elem
		}
		// 外部接收者类型：产外部方法。
		return helper.NewExternalElement(model.Method, e.name, helper.BuildQN(recvType.QualifiedName, e.name))
	}
	if elem := r.lookupScopedFunc(gc, fc, e.node, e.name, e.argc); elem != nil {
		return elem
	}
	// 变量持有 lambda（auto f = [](){...}; f();）：解析到 Lambda 元素。
	if v := r.lookupScopedSymbol(gc, fc, e.node, e.name); v != nil && v.Extra != nil {
		if lambdaQN, ok := v.Extra.Mores[constants.VariableLambdaQN].(string); ok && lambdaQN != "" {
			if entry, found := gc.FindByQualifiedName(lambdaQN); found {
				return entry.Element
			}
		}
		// 可调用对象（obj(args)）：解析到其类型的 operator() 重载。
		if owner := r.declaredTypeOf(gc, fc, v); owner != nil {
			if elem := r.lookupMemberFunc(gc, fc, owner, "operator()", e.argc); elem != nil {
				return elem
			}
		}
	}
	return nil
}

// resolveValue 解析变量/字段/枚举常量：接收者类型 → 成员；否则作用域链。
func (r *SymbolResolver) resolveValue(gc core.GlobalContext, fc core.FileContext, e *expr) *model.CodeElement {
	if e == nil || e.name == "" {
		return nil
	}
	if recvType := r.typeOf(gc, fc, e.recv); recvType != nil {
		if elem := r.lookupMember(gc, fc, recvType, e.name, model.Field); elem != nil {
			return elem
		}
	}
	return r.lookupScopedSymbol(gc, fc, e.node, e.name)
}

// typeOf 返回表达式的静态类型（链式接收者传播的递归入口）。
func (r *SymbolResolver) typeOf(gc core.GlobalContext, fc core.FileContext, e *expr) *model.CodeElement {
	if e == nil {
		return nil
	}
	switch e.form {
	case formNew, formCast:
		if e.typeText == "" {
			return nil
		}
		return r.lookupType(gc, fc, helper.BaseTypeName(e.typeText))

	case formCall:
		return r.returnTypeOf(gc, fc, r.resolveFunc(gc, fc, e))

	case formText:
		return r.textType(gc, fc, e)

	case formScoped:
		// A::B 先按完整限定名当类型（namespace/嵌套类），再按成员变量。
		if e.node != nil {
			if t := r.lookupType(gc, fc, helper.BaseTypeName(helper.GetNodeContent(e.node, *fc.SourceBytes()))); t != nil {
				return t
			}
		}
		if t := r.declaredTypeOf(gc, fc, r.resolveValue(gc, fc, e)); t != nil {
			return t
		}
		return r.lookupType(gc, fc, helper.BaseTypeName(e.name))

	case formIdent, formMember:
		// 裸标识符/成员作接收者时通常是变量（obj.method()），变量优先于同名类型。
		if t := r.declaredTypeOf(gc, fc, r.resolveValue(gc, fc, e)); t != nil {
			return t
		}
		return r.lookupType(gc, fc, helper.BaseTypeName(e.name))
	}
	return nil
}

// textType 解析纯文本接收者的类型（公开 SPI 传入 receiver 字符串时）。
func (r *SymbolResolver) textType(gc core.GlobalContext, fc core.FileContext, e *expr) *model.CodeElement {
	recv := strings.TrimSpace(e.text)
	if recv == "" {
		return nil
	}
	if recv == "this" || recv == "*this" {
		return enclosingType(fc, e.node)
	}
	if t := r.lookupType(gc, fc, helper.BaseTypeName(recv)); t != nil {
		return t
	}
	return r.declaredTypeOf(gc, fc, r.lookupScopedSymbol(gc, fc, e.node, recv))
}

// returnTypeOf 取函数元素的返回类型元素（优先 QN 版本，退回原始文本）。
func (r *SymbolResolver) returnTypeOf(gc core.GlobalContext, fc core.FileContext, fn *model.CodeElement) *model.CodeElement {
	if fn == nil || fn.Extra == nil {
		return nil
	}
	for _, key := range []string{constants.FunctionReturnTypeWithQN, constants.FunctionReturnType} {
		if t, ok := fn.Extra.Mores[key].(string); ok && t != "" {
			if elem := r.lookupType(gc, fc, helper.BaseTypeName(t)); elem != nil {
				return elem
			}
		}
	}
	return nil
}

// declaredTypeKeys 是变量/字段声明类型的 Mores 键（优先 QN 版本，退回原始文本）。
var declaredTypeKeys = []string{
	constants.VariableTypeWithQN, constants.FieldTypeWithQN,
	constants.VariableRawType, constants.FieldRawType,
}

// declaredTypeText 取变量/字段元素的声明类型原文。
func declaredTypeText(v *model.CodeElement) string {
	if v == nil || v.Extra == nil {
		return ""
	}
	for _, key := range declaredTypeKeys {
		if t, ok := v.Extra.Mores[key].(string); ok && t != "" {
			return t
		}
	}
	return ""
}

// declaredTypeOf 取变量/字段元素的声明类型元素。
// 源码内类型优先；外部类型（std::string 等）也返回外部类型元素，
// 使 `s.size()` 能按接收者类型解析为 std::string::size 而非退化为 s::size。
func (r *SymbolResolver) declaredTypeOf(gc core.GlobalContext, fc core.FileContext, v *model.CodeElement) *model.CodeElement {
	raw := declaredTypeText(v)
	if raw == "" {
		return nil
	}
	base := helper.BaseTypeName(raw)
	if t := r.lookupType(gc, fc, base); t != nil {
		return t
	}
	if base == "" || r.IsPrimitive(base) || helper.IsTypeVariable(base) {
		return nil
	}
	return helper.NewExternalElement(model.Class, base, base)
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
		var namespaces []string
		if nfc, ok := fc.(cppctx.NamespaceAware); ok {
			namespaces = nfc.Namespaces()
		}
		for _, ns := range namespaces {
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
		if entries, ok := gc.FindOverloadCandidates(qn); ok {
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

// lambdaElementOf 返回与给定 lambda_expression 节点位置一致的 Lambda 元素。
func lambdaElementOf(fc core.FileContext, node coreast.Node) *model.CodeElement {
	if node == nil {
		return nil
	}
	for _, entry := range fc.Definitions() {
		if entry.Element.Kind != model.Lambda || entry.Node == nil {
			continue
		}
		if entry.Node.Equal(node) {
			return entry.Element
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

// lookupLocalVariable 仅沿词法作用域链查找局部变量（不跨文件、不查全局短名）。
// 用于判定「被调用者其实是局部可调用对象」，避免误产外部函数。
func (r *SymbolResolver) lookupLocalVariable(gc core.GlobalContext, fc core.FileContext, node coreast.Node, name string) *model.CodeElement {
	for _, scope := range r.scopeChain(fc, node) {
		qn := helper.BuildQN(scope, name)
		if entry, ok := gc.FindByQualifiedName(qn); ok && entry.Element.Kind == model.Variable {
			return entry.Element
		}
	}
	return nil
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
		if entries, ok := gc.FindOverloadCandidates(qn); ok {
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
				if name := typeNodeName(args.NamedChild(i), src); name != "" {
					return name
				}
			}
		}
	}
	return ""
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
