package test

import (
	"sort"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/constants"
)

// =============================================================================
// 本文件覆盖 C 语言 Collector 的全量元素提取（11 组场景）。
// 每组场景对应 testdata/collector/c/<group>/ 下的独立样例。
// 断言目标是「正确行为」而非当前实现现状。
// =============================================================================

// collectC 收集指定 C 场景目录下的全部 .c/.h 文件。
func collectC(t *testing.T, group string) core.GlobalContext {
	t.Helper()
	paths := ResolvePaths(t, "collector/c/"+group, "*.c", "*.h")
	return RunCollection(t, "x/cpp/testdata", paths)
}

// AssertMods 断言元素携带指定修饰符（如 static/inline/extern）。
func AssertMods(t *testing.T, gc core.GlobalContext, qn, mod string) {
	t.Helper()
	elem := AssertDef(t, gc, qn, "")
	if elem == nil {
		return
	}
	if elem.Extra == nil || !containsStr(elem.Extra.Modifiers, mod) {
		var got []string
		if elem.Extra != nil {
			got = elem.Extra.Modifiers
		}
		t.Errorf("%s modifiers = %v, want contains %q", qn, got, mod)
	}
}

// AssertNoMores 断言元素不带指定 Mores 键（或值为 false 视为未标记）。
func AssertNoMores(t *testing.T, gc core.GlobalContext, qn, key string) {
	t.Helper()
	elem := AssertDef(t, gc, qn, "")
	if elem == nil {
		return
	}
	if v, ok := elem.GetMores()[key]; ok {
		if b, isBool := v.(bool); isBool && !b {
			return
		}
		t.Errorf("%s Mores[%s] should be absent, got %v", qn, key, v)
	}
}

// AssertNoKind 断言指定 kind 下不存在该 QN 的元素。
func AssertNoKind(t *testing.T, gc core.GlobalContext, qn string, kind model.ElementKind) {
	t.Helper()
	for _, def := range gc.Definitions() {
		if def.Element.QualifiedName == qn && def.Element.Kind == kind {
			t.Errorf("%s should NOT exist as kind %s", qn, kind)
			return
		}
	}
}

// AssertFieldsOf 断言某承载实体的字段名集合完全匹配（排序后比较）。
func AssertFieldsOf(t *testing.T, gc core.GlobalContext, parentQN string, want []string) {
	t.Helper()
	var got []string
	for _, def := range gc.Definitions() {
		if def.ParentQN == parentQN && def.Element.Kind == model.Field {
			got = append(got, def.Element.Name)
		}
	}
	sort.Strings(got)
	sortedWant := append([]string(nil), want...)
	sort.Strings(sortedWant)
	if !moresEqual(got, sortedWant) {
		t.Errorf("%s fields = %v, want %v", parentQN, got, sortedWant)
	}
}

// AssertSignature 断言函数元素的参数签名 Mores（类型 + 参数名列表）。
func AssertSignature(t *testing.T, gc core.GlobalContext, qn string, want []string) {
	t.Helper()
	AssertMores(t, gc, qn, constants.FunctionParameters, want)
}

// assertElemInFile 在指定文件上下文中按 QN + kind 查找元素（声明/定义共存时按 kind 区分）。
func assertElemInFile(t *testing.T, gc core.GlobalContext, filePath, qn string, kind model.ElementKind) *model.CodeElement {
	t.Helper()
	fc, ok := gc.FileContext(filePath)
	if !ok {
		t.Fatalf("file context not found: %s", filePath)
	}
	for _, def := range fc.Definitions() {
		if def.Element.QualifiedName == qn && def.Element.Kind == kind {
			return def.Element
		}
	}
	return nil
}

// AssertMoresInFile 断言指定文件上下文中某元素的 Mores。
func AssertMoresInFile(t *testing.T, gc core.GlobalContext, filePath, qn string, kind model.ElementKind, key string, want interface{}) {
	t.Helper()
	elem := assertElemInFile(t, gc, filePath, qn, kind)
	if elem == nil {
		t.Errorf("%s (kind=%s) not found in %s", qn, kind, filePath)
		return
	}
	got, exists := elem.GetMores()[key]
	if !exists {
		t.Errorf("%s Mores[%s] missing in %s", qn, key, filePath)
		return
	}
	if want != nil && !moresEqual(got, want) {
		t.Errorf("%s Mores[%s] = %v, want %v", qn, key, got, want)
	}
}

// =============================================================================
// 1. 基准元素
// =============================================================================

func TestCollectorCBase(t *testing.T) {
	gc := collectC(t, "base")

	// 文件元素
	AssertDef(t, gc, "collector/c/base/base.c", model.File)
	AssertDef(t, gc, "collector/c/base/base.h", model.File)

	// 函数定义 + 头文件原型（同 QN 不同 kind 共存）
	AssertDef(t, gc, "add(int,int)", model.Function)
	assertElemInFile(t, gc, "collector/c/base/base.h", "add(int,int)", model.FunctionDecl)
	AssertSignature(t, gc, "add(int,int)", []string{"int a", "int b"})
	AssertMores(t, gc, "add(int,int)", constants.FunctionReturnType, "int")

	// 全局变量
	AssertDef(t, gc, "counter", model.Variable)
	AssertMores(t, gc, "counter", constants.VariableIsGlobal, true)
	AssertMores(t, gc, "counter", constants.VariableRawType, "int")

	// 结构体与字段
	AssertDef(t, gc, "Point", model.Struct)
	AssertFieldsOf(t, gc, "Point", []string{"x", "y"})
	AssertMores(t, gc, "Point::x", constants.FieldRawType, "int")

	// 枚举与常量
	AssertDef(t, gc, "Color", model.Enum)
	AssertDef(t, gc, "Color::RED", model.EnumConstant)
	AssertDef(t, gc, "Color::GREEN", model.EnumConstant)

	// typedef
	AssertDef(t, gc, "ulong_t", model.Type)
	AssertMores(t, gc, "ulong_t", constants.TypeUnderlying, "unsigned long")

	// 宏
	AssertDef(t, gc, "ANSWER", model.Macro)
	AssertMores(t, gc, "ANSWER", constants.MacroBody, "42")
	AssertMores(t, gc, "ANSWER", constants.MacroIsFunctionLike, false)

	// 参数与局部变量
	AssertDef(t, gc, "add(int,int)::a", model.Variable)
	AssertDef(t, gc, "add(int,int)::b", model.Variable)
	AssertDef(t, gc, "add(int,int)::sum", model.Variable)
	AssertMores(t, gc, "add(int,int)::sum", constants.VariableRawType, "int")
	// 局部变量不是全局变量
	AssertNoMores(t, gc, "add(int,int)::sum", constants.VariableIsGlobal)
}

// =============================================================================
// 2. 函数族
// =============================================================================

func TestCollectorCFunc(t *testing.T) {
	gc := collectC(t, "func")

	// f(void) 与 f() 都归一到 ()
	AssertDef(t, gc, "no_args()", model.Function)
	AssertDef(t, gc, "empty_parens()", model.Function)
	AssertMores(t, gc, "no_args()", constants.FunctionReturnType, "void")

	// 可变参数
	AssertDef(t, gc, "variadic(int,...)", model.Function)
	AssertMores(t, gc, "variadic(int,...)", constants.FunctionIsVariadic, true)
	AssertSignature(t, gc, "variadic(int,...)", []string{"int first", "..."})

	// 存储类/限定符
	AssertDef(t, gc, "static_helper(int)", model.Function)
	AssertMods(t, gc, "static_helper(int)", "static")
	AssertDef(t, gc, "inline_helper(int)", model.Function)
	AssertMods(t, gc, "inline_helper(int)", "inline")

	// 递归：函数体作为独立作用域块
	AssertDef(t, gc, "fact(int)", model.Function)
	AssertDef(t, gc, "fact(int)::block$1", model.ScopeBlock)

	// 函数指针变量 → VARIABLE（不是函数）
	AssertDef(t, gc, "gfn", model.Variable)
	AssertNoKind(t, gc, "gfn", model.Function)
	AssertNoKind(t, gc, "gfn", model.FunctionDecl)

	// 返回函数指针的函数 → FUNCTION
	AssertDef(t, gc, "rfp(int)", model.Function)
	AssertNoKind(t, gc, "rfp(int)", model.Variable)
	AssertSignature(t, gc, "rfp(int)", []string{"int n"})

	// 返回数组指针的函数 → FUNCTION
	AssertDef(t, gc, "rap()", model.Function)
	AssertNoKind(t, gc, "rap()", model.Variable)

	// 返回函数指针的函数原型 → FUNCTION_DECL
	AssertDef(t, gc, "signal_like(int,void (*))", model.FunctionDecl)
	AssertMores(t, gc, "signal_like(int,void (*))", constants.FunctionIsDeclaration, true)
}

// =============================================================================
// 3. 声明与定义分离
// =============================================================================

func TestCollectorCDeclDef(t *testing.T) {
	gc := collectC(t, "decl_def")

	// 同文件先声明后定义：共享同一 QN，不产生 $2 消歧
	AssertDef(t, gc, "proto(int)", model.Function)
	assertElemInFile(t, gc, "collector/c/decl_def/decl_def.c", "proto(int)", model.FunctionDecl)
	AssertNoDef(t, gc, "proto(int)$2")

	// 结构体前向声明 + 定义：定义覆盖声明，同一 QN 只有一个 STRUCT
	AssertDef(t, gc, "FwdDef", model.Struct)
	AssertNoDef(t, gc, "FwdDef$2")
	AssertNoMores(t, gc, "FwdDef", constants.StructIsDeclaration)
	AssertFieldsOf(t, gc, "FwdDef", []string{"v"})

	// 枚举前向声明 + 定义
	AssertDef(t, gc, "FwdEnum", model.Enum)
	AssertNoDef(t, gc, "FwdEnum$2")
	AssertDef(t, gc, "FwdEnum::FE_A", model.EnumConstant)

	// extern 变量声明 + 定义：定义优先，is_extern 不再保留
	AssertDef(t, gc, "shared_var", model.Variable)
	AssertMores(t, gc, "shared_var", constants.VariableIsGlobal, true)
	AssertNoMores(t, gc, "shared_var", constants.VariableIsExtern)

	// 只声明不定义
	AssertDef(t, gc, "never_defined(int)", model.FunctionDecl)
	AssertNoKind(t, gc, "never_defined(int)", model.Function)

	// 头文件原型
	AssertDef(t, gc, "header_only(int)", model.FunctionDecl)
	AssertMoresInFile(t, gc, "collector/c/decl_def/decl_def.h", "header_only(int)", model.FunctionDecl,
		constants.FunctionIsDeclaration, true)
}

// =============================================================================
// 4. struct / union
// =============================================================================

func TestCollectorCStruct(t *testing.T) {
	gc := collectC(t, "struct")

	// 具名结构体 + 自引用字段
	AssertDef(t, gc, "Node", model.Struct)
	AssertFieldsOf(t, gc, "Node", []string{"value", "next"})
	AssertMores(t, gc, "Node::next", constants.FieldRawType, "struct Node")

	// 嵌套具名结构体：QN 带父级前缀
	AssertDef(t, gc, "Outer", model.Struct)
	AssertDef(t, gc, "Outer::Inner", model.Struct)
	AssertFieldsOf(t, gc, "Outer::Inner", []string{"depth"})
	AssertFieldsOf(t, gc, "Outer", []string{"id", "inner", "payload"})

	// 匿名 union 成员：字段挂外层字段名下
	AssertFieldsOf(t, gc, "Outer::payload", []string{"as_int", "as_float"})

	// 位域
	AssertFieldsOf(t, gc, "Bits", []string{"flag", "count"})
	AssertMores(t, gc, "Bits::flag", constants.FieldRawType, "unsigned int")

	// 匿名 struct 多声明器：每个变量各挂一份字段（允许冗余）
	AssertDef(t, gc, "s1", model.Variable)
	AssertDef(t, gc, "s2", model.Variable)
	AssertFieldsOf(t, gc, "s1", []string{"m1", "m2"})
	AssertFieldsOf(t, gc, "s2", []string{"m1", "m2"})

	// 具名 union
	AssertDef(t, gc, "Value", model.Struct)
	AssertMores(t, gc, "Value", constants.StructIsUnion, true)
	AssertFieldsOf(t, gc, "Value", []string{"i", "d"})

	// 匿名 union 多声明器
	AssertDef(t, gc, "MU1", model.Variable)
	AssertDef(t, gc, "MU2", model.Variable)
	AssertFieldsOf(t, gc, "MU1", []string{"u1", "u2"})
	AssertFieldsOf(t, gc, "MU2", []string{"u1", "u2"})
}

// =============================================================================
// 5. enum
// =============================================================================

func TestCollectorCEnum(t *testing.T) {
	gc := collectC(t, "enum")

	// 具名枚举与常量
	AssertDef(t, gc, "Color", model.Enum)
	AssertFieldsOf(t, gc, "Color", nil) // 枚举常量不是 FIELD
	for _, c := range []string{"RED", "GREEN", "BLUE"} {
		AssertDef(t, gc, "Color::"+c, model.EnumConstant)
	}

	// 显式值枚举
	AssertDef(t, gc, "Status", model.Enum)
	for _, c := range []string{"OK", "WARN", "FAIL"} {
		AssertDef(t, gc, "Status::"+c, model.EnumConstant)
	}

	// 匿名枚举：常量挂文件级，不产 ENUM 元素
	AssertDef(t, gc, "ANON_A", model.EnumConstant)
	AssertDef(t, gc, "ANON_B", model.EnumConstant)

	// 前向声明 + 定义：定义覆盖，常量挂在枚举下
	AssertDef(t, gc, "Forward", model.Enum)
	AssertNoDef(t, gc, "Forward$2")
	AssertDef(t, gc, "Forward::FWD_A", model.EnumConstant)

	// 枚举类型变量
	AssertDef(t, gc, "current", model.Variable)
	AssertMores(t, gc, "current", constants.VariableRawType, "enum Color")
}

// =============================================================================
// 6. typedef
// =============================================================================

func TestCollectorCTypedef(t *testing.T) {
	gc := collectC(t, "typedef")

	// 基础别名
	AssertDef(t, gc, "ulong_t", model.Type)
	AssertMores(t, gc, "ulong_t", constants.TypeUnderlying, "unsigned long")

	// 指针别名 / 数组别名 / 函数指针别名
	AssertDef(t, gc, "int_ptr_t", model.Type)
	AssertMores(t, gc, "int_ptr_t", constants.TypeUnderlying, "int*")
	AssertDef(t, gc, "int_arr_t", model.Type)
	AssertMores(t, gc, "int_arr_t", constants.TypeUnderlying, "int[]")
	AssertDef(t, gc, "callback_t", model.Type)

	// 多声明器：每个都产 TYPE
	for _, n := range []string{"TI1", "TI2", "TI3", "PTI3"} {
		AssertDef(t, gc, n, model.Type)
	}
	AssertMores(t, gc, "TI1", constants.TypeUnderlying, "int")
	AssertMores(t, gc, "TI2", constants.TypeUnderlying, "int")
	AssertMores(t, gc, "PTI3", constants.TypeUnderlying, "int*")

	// 链式别名
	AssertDef(t, gc, "my_ulong_t", model.Type)
	AssertMores(t, gc, "my_ulong_t", constants.TypeUnderlying, "ulong_t")

	// 匿名 struct typedef：字段挂 typedef 名
	AssertDef(t, gc, "Point", model.Type)
	AssertFieldsOf(t, gc, "Point", []string{"x", "y"})

	// 匿名 struct 多声明器 typedef：每个承载实体各挂一份字段
	AssertDef(t, gc, "TA", model.Type)
	AssertDef(t, gc, "PTA", model.Type)
	AssertFieldsOf(t, gc, "TA", []string{"a"})
	AssertFieldsOf(t, gc, "PTA", []string{"a"})
	AssertMores(t, gc, "PTA", constants.TypeUnderlying, "struct { int a; }*")

	// 具名 struct typedef：结构体与别名各自独立
	AssertDef(t, gc, "Named", model.Struct)
	AssertDef(t, gc, "Named_t", model.Type)
	AssertFieldsOf(t, gc, "Named", []string{"n"})
}

// =============================================================================
// 7. 宏与预处理
// =============================================================================

func TestCollectorCMacro(t *testing.T) {
	gc := collectC(t, "macro")

	// 对象宏
	AssertDef(t, gc, "MAX_SIZE", model.Macro)
	AssertMores(t, gc, "MAX_SIZE", constants.MacroBody, "100")
	AssertMores(t, gc, "MAX_SIZE", constants.MacroIsFunctionLike, false)

	// 函数式宏
	AssertDef(t, gc, "SQUARE", model.Macro)
	AssertMores(t, gc, "SQUARE", constants.MacroIsFunctionLike, true)
	AssertDef(t, gc, "MULTI_LINE", model.Macro)
	AssertMores(t, gc, "MULTI_LINE", constants.MacroIsFunctionLike, true)
	AssertDef(t, gc, "STR", model.Macro)
	AssertDef(t, gc, "CAT", model.Macro)

	// 空宏：有元素、无 body
	AssertDef(t, gc, "EMPTY_MACRO", model.Macro)
	AssertMores(t, gc, "EMPTY_MACRO", constants.MacroIsFunctionLike, false)

	// 条件分支内的同名宏：全分支保留，各自标注 condition
	AssertDef(t, gc, "FEATURE_VALUE", model.Macro)
	AssertMores(t, gc, "FEATURE_VALUE", constants.Condition, "#ifdef FEATURE_X")
	AssertMores(t, gc, "FEATURE_VALUE", constants.MacroBody, "1")
	AssertDef(t, gc, "FEATURE_VALUE$2", model.Macro)
	AssertMores(t, gc, "FEATURE_VALUE$2", constants.Condition, "#else")
	AssertMores(t, gc, "FEATURE_VALUE$2", constants.MacroBody, "0")

	// 头文件宏
	AssertDef(t, gc, "HEADER_MACRO", model.Macro)
}

// =============================================================================
// 8. 变量与作用域
// =============================================================================

func TestCollectorCVariable(t *testing.T) {
	gc := collectC(t, "variable")

	// 全局变量族
	AssertDef(t, gc, "g_counter", model.Variable)
	AssertMores(t, gc, "g_counter", constants.VariableIsGlobal, true)
	AssertDef(t, gc, "g_static", model.Variable)
	AssertMores(t, gc, "g_static", constants.VariableIsStatic, true)
	AssertDef(t, gc, "g_const", model.Variable)
	AssertMores(t, gc, "g_const", constants.VariableIsConst, true)
	AssertDef(t, gc, "g_arr", model.Variable)
	AssertDef(t, gc, "g_init", model.Variable)

	// 多声明器：每个都产 VARIABLE
	AssertDef(t, gc, "g_multi1", model.Variable)
	AssertDef(t, gc, "g_multi2", model.Variable)

	// 局部变量
	AssertDef(t, gc, "scope_fn(int)::local", model.Variable)
	AssertNoMores(t, gc, "scope_fn(int)::local", constants.VariableIsGlobal)
	AssertDef(t, gc, "scope_fn(int)::local_static", model.Variable)
	AssertMores(t, gc, "scope_fn(int)::local_static", constants.VariableIsStatic, true)
	AssertDef(t, gc, "scope_fn(int)::local_const", model.Variable)
	AssertMores(t, gc, "scope_fn(int)::local_const", constants.VariableIsConst, true)
	AssertDef(t, gc, "scope_fn(int)::local_multi_a", model.Variable)
	AssertDef(t, gc, "scope_fn(int)::local_multi_b", model.Variable)
	AssertDef(t, gc, "scope_fn(int)::local_arr", model.Variable)

	// 参数作为变量元素
	AssertDef(t, gc, "scope_fn(int)::param", model.Variable)

	// 块作用域
	AssertDef(t, gc, "scope_fn(int)::block$1", model.ScopeBlock)
	AssertDef(t, gc, "scope_fn(int)::block$1::inner", model.Variable)
	AssertDef(t, gc, "scope_fn(int)::block$2", model.ScopeBlock)
	AssertDef(t, gc, "scope_fn(int)::block$2::loop_body", model.Variable)

	// 混合声明器：函数 + 变量共存，变量不被丢弃
	AssertDef(t, gc, "mixf(int)", model.FunctionDecl)
	AssertDef(t, gc, "mixv", model.Variable)
	AssertMores(t, gc, "mixv", constants.VariableIsGlobal, true)
}

// =============================================================================
// 9. 参数形态
// =============================================================================

func TestCollectorCParam(t *testing.T) {
	gc := collectC(t, "param")

	// 具名参数
	AssertDef(t, gc, "named(int,char*)", model.Function)
	AssertSignature(t, gc, "named(int,char*)", []string{"int a", "char* p"})

	// 无名参数：保留类型，保留 struct/enum 关键字
	AssertDef(t, gc, "unnamed(int,char*,struct S)", model.Function)
	AssertSignature(t, gc, "unnamed(int,char*,struct S)", []string{"int", "char*", "struct S"})

	// 多级指针
	AssertDef(t, gc, "multilevel(int**,char***)", model.Function)
	AssertSignature(t, gc, "multilevel(int**,char***)", []string{"int** pp", "char*** ppp"})

	// 数组参数
	AssertDef(t, gc, "arrays(int[],char[])", model.Function)
	AssertSignature(t, gc, "arrays(int[],char[])", []string{"int[] arr", "char[] buf"})

	// 函数指针参数
	AssertDef(t, gc, "func_ptr_param(cb_t,int (*))", model.Function)
	AssertSignature(t, gc, "func_ptr_param(cb_t,int (*))", []string{"cb_t cb", "int (*) direct"})

	// const 与 struct/enum 关键字
	AssertDef(t, gc, "qualified(const int*,const struct S*,enum E)", model.Function)
	AssertSignature(t, gc, "qualified(const int*,const struct S*,enum E)",
		[]string{"const int* cp", "const struct S* sp", "enum E e"})

	// 可变参数
	AssertDef(t, gc, "mixed(int,...)", model.Function)
	AssertMores(t, gc, "mixed(int,...)", constants.FunctionIsVariadic, true)

	// 无名指针参数不丢 *
	AssertDef(t, gc, "unnamed(int,char*,struct S)", model.Function)
}

// =============================================================================
// 10. 条件编译
// =============================================================================

func TestCollectorCConditional(t *testing.T) {
	gc := collectC(t, "cond")

	AssertMores(t, gc, "cond_a", constants.Condition, "#ifdef FOO")
	AssertMores(t, gc, "cond_ifndef", constants.Condition, "#ifndef FOO")
	AssertMores(t, gc, "cond_if", constants.Condition, "#if defined(BAR)")
	AssertMores(t, gc, "cond_elif", constants.Condition, "#elif defined(BAZ)")
	AssertMores(t, gc, "cond_else", constants.Condition, "#else")
	// 嵌套分支取最内层
	AssertMores(t, gc, "cond_nested", constants.Condition, "#ifdef INNER")
	// 条件分支内的函数
	AssertDef(t, gc, "cond_fn()", model.Function)
	AssertMores(t, gc, "cond_fn()", constants.Condition, "#ifdef FOO")
}

// =============================================================================
// 11. 边界与负例
// =============================================================================

func TestCollectorCEdge(t *testing.T) {
	gc := collectC(t, "edge")

	// 类型引用不产元素：struct Tag t; 只产变量，不产新的 STRUCT
	AssertDef(t, gc, "use_type_ref()::t", model.Variable)
	AssertNoKind(t, gc, "use_type_ref()::Tag", model.Struct)
	// 局部匿名 enum 的常量挂当前作用域，不产 ENUM 元素
	AssertDef(t, gc, "use_type_ref()::LOCAL_A", model.EnumConstant)

	// 不同块内同名局部变量：各自独立 QN
	AssertDef(t, gc, "dup_blocks()::block$1::dup", model.Variable)
	AssertDef(t, gc, "dup_blocks()::block$2::dup", model.Variable)

	// 同名不同签名函数
	AssertDef(t, gc, "overload_like(int)", model.Function)
	AssertDef(t, gc, "overload_like2(int,int)", model.Function)

	// 无名指针参数
	AssertDef(t, gc, "no_name_ptr(char*)", model.Function)
	AssertSignature(t, gc, "no_name_ptr(char*)", []string{"char*"})

	// 嵌套具名结构体
	AssertDef(t, gc, "NestedOuter", model.Struct)
	AssertDef(t, gc, "NestedOuter::NestedInner", model.Struct)
	AssertFieldsOf(t, gc, "NestedOuter::NestedInner", []string{"deep"})
	AssertFieldsOf(t, gc, "NestedOuter", []string{"inner"})

	// 顶层具名结构体只产一次
	AssertDef(t, gc, "Tag", model.Struct)
	AssertFieldsOf(t, gc, "Tag", []string{"v"})
}
