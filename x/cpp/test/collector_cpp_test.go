package test

import (
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/constants"
)

// =============================================================================
// 本文件覆盖 C++ 语言 Collector 的全量元素提取。
// 每组场景对应 testdata/collector/cpp/<group>/ 下的独立样例。
// 断言目标是「正确行为」而非当前实现现状。
// =============================================================================

// collectCpp 收集指定 C++ 场景目录下的全部源文件。
func collectCpp(t *testing.T, group string) core.GlobalContext {
	t.Helper()
	paths := ResolvePaths(t, "collector/cpp/"+group, "*.c", "*.h", "*.cpp", "*.hpp")
	return RunCollection(t, "x/cpp/testdata", paths)
}

// =============================================================================
// 1. 基准元素：16 种 ElementKind 冒烟
// =============================================================================

func TestCollectorCppBase(t *testing.T) {
	gc := collectCpp(t, "base")

	// --- File ---
	AssertDef(t, gc, "collector/cpp/base/base.h", model.File)
	AssertDef(t, gc, "collector/cpp/base/base.cpp", model.File)
	AssertDef(t, gc, "collector/cpp/base/cpp_base_dep.h", model.File)

	// --- Namespace ---
	AssertDef(t, gc, "base_ns", model.Namespace)

	// --- Struct / Class ---
	AssertDef(t, gc, "base_ns::BaseStruct", model.Struct)
	AssertFieldsOf(t, gc, "base_ns::BaseStruct", []string{"field"})
	AssertDef(t, gc, "base_ns::BaseClass", model.Class)
	AssertFieldsOf(t, gc, "base_ns::BaseClass", []string{"member"})

	// --- Enum / EnumConstant ---
	AssertDef(t, gc, "base_ns::BaseEnum", model.Enum)
	AssertDef(t, gc, "base_ns::BaseEnum::BE_A", model.EnumConstant)
	AssertDef(t, gc, "base_ns::BaseEnum::BE_B", model.EnumConstant)
	AssertDef(t, gc, "base_ns::ScopedEnum", model.Enum)
	AssertMores(t, gc, "base_ns::ScopedEnum", constants.EnumIsScoped, true)
	AssertDef(t, gc, "base_ns::ScopedEnum::SE_A", model.EnumConstant)

	// --- Function / FunctionDecl ---
	AssertDef(t, gc, "base_ns::base_func(int)", model.Function)
	AssertDef(t, gc, "base_ns::base_lambda_user()", model.Function)
	assertElemInFile(t, gc, "collector/cpp/base/base.h", "base_ns::base_func(int)", model.FunctionDecl)

	// --- Method / MethodDecl ---
	AssertDef(t, gc, "base_ns::BaseClass::method()", model.Method)
	AssertDef(t, gc, "base_ns::BaseClass::BaseClass()", model.Method)
	assertElemInFile(t, gc, "collector/cpp/base/base.h", "base_ns::BaseClass::method()", model.MethodDecl)
	assertElemInFile(t, gc, "collector/cpp/base/base.h", "base_ns::BaseClass::BaseClass()", model.MethodDecl)

	// --- Field ---
	AssertDef(t, gc, "base_ns::BaseClass::member", model.Field)
	AssertMores(t, gc, "base_ns::BaseClass::member", constants.FieldRawType, "int")

	// --- Type（typedef / using）---
	AssertDef(t, gc, "base_ns::base_ulong_t", model.Type)
	AssertMores(t, gc, "base_ns::base_ulong_t", constants.TypeUnderlying, "unsigned long")
	AssertDef(t, gc, "base_ns::BaseAlias", model.Type)
	AssertMores(t, gc, "base_ns::BaseAlias", constants.TypeUnderlying, "int")

	// --- Variable ---
	AssertDef(t, gc, "base_ns::base_global", model.Variable)
	AssertMores(t, gc, "base_ns::base_global", constants.VariableIsGlobal, true)
	AssertDef(t, gc, "base_ns::base_func(int)::a", model.Variable)
	AssertDef(t, gc, "base_ns::base_func(int)::sum", model.Variable)
	AssertNoMores(t, gc, "base_ns::base_func(int)::sum", constants.VariableIsGlobal)

	// --- Macro ---
	AssertDef(t, gc, "BASE_MACRO", model.Macro)
	AssertMores(t, gc, "BASE_MACRO", constants.MacroBody, "42")
	AssertMores(t, gc, "BASE_MACRO", constants.MacroIsFunctionLike, false)
	AssertDef(t, gc, "BASE_FUNC_MACRO", model.Macro)
	AssertMores(t, gc, "BASE_FUNC_MACRO", constants.MacroIsFunctionLike, true)

	// --- Lambda ---
	AssertDef(t, gc, "base_ns::base_lambda_user()::lambda$1", model.Lambda)
	AssertDef(t, gc, "base_ns::base_lambda_user()::lambda$1::x", model.Variable)
	AssertMores(t, gc, "base_ns::base_lambda_user()::lam", constants.VariableLambdaQN,
		"base_ns::base_lambda_user()::lambda$1")

	// --- ScopeBlock ---
	AssertDef(t, gc, "base_ns::BaseClass::method()::block$1", model.ScopeBlock)
	AssertDef(t, gc, "base_ns::BaseClass::method()::block$1::in_block", model.Variable)
}

// =============================================================================
// 2. 命名空间族
// =============================================================================

func TestCollectorCppNamespace(t *testing.T) {
	gc := collectCpp(t, "namespace")

	// 传统嵌套 namespace：逐级拆分
	AssertDef(t, gc, "outer", model.Namespace)
	AssertDef(t, gc, "outer::middle", model.Namespace)
	AssertDef(t, gc, "outer::middle::inner", model.Namespace)
	AssertDef(t, gc, "outer::middle::inner::Deep", model.Struct)

	// C++17 嵌套写法 namespace modern::sub
	AssertDef(t, gc, "modern::sub", model.Namespace)
	AssertDef(t, gc, "modern::sub::Modern", model.Struct)

	// inline namespace
	AssertDef(t, gc, "inl", model.Namespace)
	AssertDef(t, gc, "inl::v1", model.Namespace)
	AssertDef(t, gc, "inl::v1::inline_var", model.Variable)

	// 匿名 namespace 记名 $anon
	AssertDef(t, gc, "$anon", model.Namespace)
	AssertDef(t, gc, "$anon::anon_var", model.Variable)
	AssertDef(t, gc, "$anon::AnonStruct", model.Struct)

	// namespace alias / using 不产元素
	AssertNoDef(t, gc, "alias_ns")
	AssertNoDef(t, gc, "using")

	// 命名空间列表（NamespaceAware）
	fc, ok := gc.FileContext("collector/cpp/namespace/ns.cpp")
	if !ok {
		t.Fatal("file context not found")
	}
	nfc, ok := fc.(interface{ Namespaces() []string })
	if !ok {
		t.Fatal("file context should implement Namespaces()")
	}
	got := map[string]bool{}
	for _, ns := range nfc.Namespaces() {
		got[ns] = true
	}
	for _, want := range []string{"outer", "outer::middle", "outer::middle::inner", "modern::sub", "inl", "inl::v1", "$anon"} {
		if !got[want] {
			t.Errorf("Namespaces() missing %q, got %v", want, nfc.Namespaces())
		}
	}
}

// =============================================================================
// 3. 声明 / 定义分离
// =============================================================================

func TestCollectorCppDeclDef(t *testing.T) {
	gc := collectCpp(t, "decl_def")

	// 同文件先声明后定义：共享 QN，不产生 $2
	AssertDef(t, gc, "local_proto(int)", model.Function)
	assertElemInFile(t, gc, "collector/cpp/decl_def/decl_def.cpp", "local_proto(int)", model.FunctionDecl)
	AssertNoDef(t, gc, "local_proto(int)$2")

	// 头文件原型 + 源文件定义
	AssertDef(t, gc, "header_func(int)", model.Function)
	assertElemInFile(t, gc, "collector/cpp/decl_def/decl_def.h", "header_func(int)", model.FunctionDecl)
	AssertNoDef(t, gc, "header_func(int)$2")

	// 前向声明 + 定义：定义覆盖，不产生 $2
	AssertDef(t, gc, "FwdStruct", model.Struct)
	AssertNoDef(t, gc, "FwdStruct$2")
	AssertFieldsOf(t, gc, "FwdStruct", []string{"fv"})
	AssertNoMores(t, gc, "FwdStruct", constants.StructIsDeclaration)

	AssertDef(t, gc, "FwdClass", model.Class)
	AssertNoDef(t, gc, "FwdClass$2")
	AssertFieldsOf(t, gc, "FwdClass", []string{"cv"})
	AssertNoMores(t, gc, "FwdClass", constants.ClassIsDeclaration)

	AssertDef(t, gc, "FwdEnum", model.Enum)
	AssertNoDef(t, gc, "FwdEnum$2")
	AssertDef(t, gc, "FwdEnum::FE_A", model.EnumConstant)

	// 类内声明 + 类外定义
	AssertDef(t, gc, "DeclaredClass::declared_method()", model.Method)
	assertElemInFile(t, gc, "collector/cpp/decl_def/decl_def.h", "DeclaredClass::declared_method()", model.MethodDecl)

	// extern 声明 + 定义：定义优先，不再保留 is_extern
	AssertDef(t, gc, "extern_var", model.Variable)
	AssertMores(t, gc, "extern_var", constants.VariableIsGlobal, true)
	AssertNoMores(t, gc, "extern_var", constants.VariableIsExtern)

	// 只声明不定义
	AssertDef(t, gc, "never_defined(int)", model.FunctionDecl)
	AssertNoKind(t, gc, "never_defined(int)", model.Function)

	// 头文件内完整定义的结构体
	AssertDef(t, gc, "DefinedInHeader", model.Struct)
	AssertFieldsOf(t, gc, "DefinedInHeader", []string{"hv"})
}

// =============================================================================
// 4. 类族：访问级别 / 嵌套类型 / friend / static / const / mutable / final
// =============================================================================

func TestCollectorCppClass(t *testing.T) {
	gc := collectCpp(t, "class")

	// 抽象类与纯虚
	AssertDef(t, gc, "Abstract", model.Class)
	AssertMores(t, gc, "Abstract::pure()", constants.FunctionIsPureVirtual, true)
	AssertMores(t, gc, "Abstract::pure()", constants.FunctionIsVirtual, true)
	AssertMores(t, gc, "Abstract::~Abstract()", constants.FunctionIsDestructor, true)
	AssertMores(t, gc, "Abstract::~Abstract()", constants.FunctionIsVirtual, true)

	// final 类
	AssertDef(t, gc, "FinalClass", model.Class)
	AssertFieldsOf(t, gc, "FinalClass", []string{"fv"})

	// 嵌套类型：QN 带父级前缀
	AssertDef(t, gc, "Outer::NestedStruct", model.Struct)
	AssertFieldsOf(t, gc, "Outer::NestedStruct", []string{"ns"})
	AssertDef(t, gc, "Outer::NestedClass", model.Class)
	AssertFieldsOf(t, gc, "Outer::NestedClass", []string{"nc"})
	AssertDef(t, gc, "Outer::NestedEnum", model.Enum)
	AssertDef(t, gc, "Outer::NestedEnum::NE_A", model.EnumConstant)
	AssertDef(t, gc, "Outer::NestedScoped", model.Enum)
	AssertMores(t, gc, "Outer::NestedScoped", constants.EnumIsScoped, true)
	AssertDef(t, gc, "Outer::nested_alias_t", model.Type)
	AssertMores(t, gc, "Outer::nested_alias_t", constants.TypeUnderlying, "int")
	AssertDef(t, gc, "Outer::Inner", model.Class)
	AssertFieldsOf(t, gc, "Outer::Inner", []string{"deep"})

	// 访问级别
	AssertMores(t, gc, "Outer::pub_method()", constants.Visibility, "public")
	AssertMores(t, gc, "Outer::prot_field", constants.Visibility, "protected")
	AssertMores(t, gc, "Outer::prot_method()", constants.Visibility, "protected")
	AssertMores(t, gc, "Outer::priv_field", constants.Visibility, "private")
	AssertMores(t, gc, "Outer::priv_method()", constants.Visibility, "private")

	// static / const / mutable 成员
	AssertMores(t, gc, "Outer::static_field", constants.FieldIsStatic, true)
	AssertMores(t, gc, "Outer::static_const_field", constants.FieldIsStatic, true)
	AssertMores(t, gc, "Outer::static_const_field", constants.FieldIsConst, true)
	AssertMores(t, gc, "Outer::const_field", constants.FieldIsConst, true)
	AssertFieldsOf(t, gc, "Outer", []string{
		"static_field", "static_const_field", "const_field", "mutable_field", "prot_field", "priv_field",
	})

	// friend 声明不是成员：不产 Outer::friend_func
	AssertNoDef(t, gc, "Outer::friend_func(Outer&)")
	AssertDef(t, gc, "friend_func(Outer&)", model.Function)
}

// =============================================================================
// 5. 构造 / 析构
// =============================================================================

func TestCollectorCppCtorDtor(t *testing.T) {
	gc := collectCpp(t, "ctor_dtor")

	// 构造器族：默认 / 带参 / explicit / 拷贝 / 移动 / 双参
	for _, qn := range []string{
		"Widget::Widget()",
		"Widget::Widget(int)",
		"Widget::Widget(const char*)",
		"Widget::Widget(const Widget&)",
		"Widget::Widget(Widget&&)",
		"Widget::Widget(int,int)",
	} {
		assertElemInFile(t, gc, "collector/cpp/ctor_dtor/ctor.cpp", qn, model.MethodDecl)
		AssertMores(t, gc, qn, constants.FunctionIsConstructor, true)
	}

	// 析构器
	assertElemInFile(t, gc, "collector/cpp/ctor_dtor/ctor.cpp", "Widget::~Widget()", model.MethodDecl)
	AssertMores(t, gc, "Widget::~Widget()", constants.FunctionIsDestructor, true)

	// 赋值运算符（引用返回，需正确剥 reference_declarator）
	assertElemInFile(t, gc, "collector/cpp/ctor_dtor/ctor.cpp", "Widget::operator=(const Widget&)", model.MethodDecl)
	assertElemInFile(t, gc, "collector/cpp/ctor_dtor/ctor.cpp", "Widget::operator=(Widget&&)", model.MethodDecl)
	AssertMores(t, gc, "Widget::operator=(const Widget&)", constants.FunctionIsOperator, true)

	// = default：仍是定义语义（Kind=METHOD），额外记 is_declaration
	for _, qn := range []string{
		"Defaulted::Defaulted()",
		"Defaulted::~Defaulted()",
		"Defaulted::Defaulted(const Defaulted&)",
		"Defaulted::operator=(const Defaulted&)",
	} {
		AssertDef(t, gc, qn, model.Method)
		AssertMores(t, gc, qn, constants.IsDeclaration, true)
	}

	// = delete：同样产元素
	AssertDef(t, gc, "Deleted::Deleted()", model.Method)
	AssertDef(t, gc, "Deleted::Deleted(const Deleted&)", model.Method)
	AssertDef(t, gc, "Deleted::operator=(const Deleted&)", model.Method)

	// 委托构造：两个构造器各自独立
	AssertDef(t, gc, "Delegating::Delegating()", model.Method)
	AssertDef(t, gc, "Delegating::Delegating(int)", model.Method)

	// 类内定义的构造/析构
	AssertDef(t, gc, "Inline::Inline()", model.Method)
	AssertDef(t, gc, "Inline::~Inline()", model.Method)

	// 类外定义：定义优先命中 METHOD
	if elem := FindDef(t, gc, "Widget::Widget()"); elem.Kind != model.Method {
		t.Errorf("Widget::Widget() should resolve to definition METHOD, got %s", elem.Kind)
	}

	// 成员初始化列表的形参仍是变量元素
	AssertDef(t, gc, "Widget::Widget(int)::v", model.Variable)
}

// =============================================================================
// 6. 运算符重载
// =============================================================================

func TestCollectorCppOperator(t *testing.T) {
	gc := collectCpp(t, "operator")

	// 算术运算符
	for _, qn := range []string{
		"Num::operator+(const Num&)", "Num::operator-(const Num&)",
		"Num::operator*(int)", "Num::operator+=(const Num&)",
	} {
		AssertDef(t, gc, qn, model.MethodDecl)
		AssertMores(t, gc, qn, constants.FunctionIsOperator, true)
	}

	// 比较运算符
	for _, qn := range []string{
		"Num::operator==(const Num&)", "Num::operator!=(const Num&)",
		"Num::operator<(const Num&)", "Num::operator<=(const Num&)",
	} {
		AssertDef(t, gc, qn, model.MethodDecl)
		AssertMores(t, gc, qn, constants.FunctionIsOperator, true)
	}

	// 赋值运算符（引用返回）
	AssertDef(t, gc, "Num::operator=(const Num&)", model.MethodDecl)
	AssertDef(t, gc, "Num::operator=(Num&&)", model.MethodDecl)

	// 下标 / 调用运算符（可重载）
	AssertDef(t, gc, "Num::operator[](int)", model.MethodDecl)
	AssertDef(t, gc, "Num::operator()()", model.MethodDecl)
	AssertDef(t, gc, "Num::operator()(int,int)", model.MethodDecl)

	// 自增自减（前置/后置靠参数区分）
	AssertDef(t, gc, "Num::operator++()", model.MethodDecl)
	AssertDef(t, gc, "Num::operator++(int)", model.MethodDecl)
	AssertDef(t, gc, "Num::operator--()", model.MethodDecl)
	AssertDef(t, gc, "Num::operator--(int)", model.MethodDecl)

	// 转换运算符
	AssertDef(t, gc, "Num::operator bool()", model.MethodDecl)
	AssertDef(t, gc, "Num::operator int()", model.MethodDecl)
	AssertMores(t, gc, "Num::operator bool()", constants.FunctionIsOperator, true)

	// 内存分配运算符
	AssertDef(t, gc, "Num::operator new(unsigned long)", model.MethodDecl)
	AssertDef(t, gc, "Num::operator delete(void*)", model.MethodDecl)

	// 逻辑 / 位运算
	AssertDef(t, gc, "Num::operator!()", model.MethodDecl)
	AssertDef(t, gc, "Num::operator&(const Num&)", model.MethodDecl)
	AssertDef(t, gc, "Num::operator|(const Num&)", model.MethodDecl)
	AssertDef(t, gc, "Num::operator~()", model.MethodDecl)

	// 非成员运算符
	AssertDef(t, gc, "operator+(int,const Num&)", model.FunctionDecl)
	AssertDef(t, gc, "operator>(const Num&,const Num&)", model.FunctionDecl)
	AssertMores(t, gc, "operator+(int,const Num&)", constants.FunctionIsOperator, true)
}

// =============================================================================
// 7. 模板
// =============================================================================

func TestCollectorCppTemplate(t *testing.T) {
	gc := collectCpp(t, "template")

	// 类模板：模板参数原文记录
	AssertDef(t, gc, "Box", model.Struct)
	AssertMores(t, gc, "Box", constants.TemplateParams, "<typename T>")
	AssertFieldsOf(t, gc, "Box", []string{"value", "int_value", "ptr_value"})
	// 类内声明 + 类外定义：定义优先命中 METHOD，声明在文件上下文中
	assertElemInFile(t, gc, "collector/cpp/template/tpl.cpp", "Box::put(T)", model.MethodDecl)

	// 函数模板
	AssertDef(t, gc, "identity(T)", model.Function)
	AssertMores(t, gc, "identity(T)", constants.TemplateParams, "<typename T>")
	AssertMores(t, gc, "identity(T)", constants.FunctionParameters, []string{"T v"})

	// 多参数 + 默认模板参数
	AssertDef(t, gc, "Map", model.Struct)
	AssertMores(t, gc, "Map", constants.TemplateParams, "<typename K, typename V = int, int N = 8>")

	// 可变参数模板
	AssertDef(t, gc, "Tuple", model.Struct)
	AssertMores(t, gc, "Tuple", constants.TemplateParams, "<typename... Ts>")
	AssertDef(t, gc, "count_args(Ts... args)", model.Function)

	// 全特化 / 偏特化：QN 剥离模板实参，与主模板同 QN
	AssertNoDef(t, gc, "Box<int>")
	AssertNoDef(t, gc, "Box<T*>")
	AssertNoDef(t, gc, "Box<int>$2")

	// 函数模板显式特化：QN 剥离实参
	AssertDef(t, gc, "identity(int)", model.Function)
	AssertNoDef(t, gc, "identity<int>(int)")

	// 别名模板
	AssertDef(t, gc, "BoxAlias", model.Type)
	AssertMores(t, gc, "BoxAlias", constants.TypeUnderlying, "Box<T>")

	// concept / requires
	AssertDef(t, gc, "constrained_func(T)", model.Function)
	AssertMores(t, gc, "constrained_func(T)", constants.TemplateParams, "<Small T>")

	// 模板参数不是变量元素
	AssertNoDef(t, gc, "T")

	// 类模板成员类外定义
	AssertDef(t, gc, "Box::put(T)", model.Method)
}

// =============================================================================
// 8. 继承
// =============================================================================

func TestCollectorCppInherit(t *testing.T) {
	gc := collectCpp(t, "inherit")

	// 单继承
	AssertMores(t, gc, "Single", constants.ClassSuperClass, []string{"BaseA"})
	// 多继承：基类列表保序
	AssertMores(t, gc, "Multi", constants.ClassSuperClass, []string{"BaseA", "BaseB"})
	// virtual 继承
	AssertMores(t, gc, "VLeft", constants.ClassSuperClass, []string{"BaseA"})
	AssertMores(t, gc, "VRight", constants.ClassSuperClass, []string{"BaseA"})
	// 菱形继承
	AssertMores(t, gc, "Diamond", constants.ClassSuperClass, []string{"VLeft", "VRight"})
	// private / protected 继承
	AssertMores(t, gc, "PrivDerived", constants.ClassSuperClass, []string{"BaseB"})
	AssertMores(t, gc, "ProtDerived", constants.ClassSuperClass, []string{"BaseC"})
	// override / final
	AssertMores(t, gc, "Overrider", constants.ClassSuperClass, []string{"BaseA"})
	AssertMores(t, gc, "FinalDerived", constants.ClassSuperClass, []string{"BaseB"})
	// 继承 + 混合访问级别
	AssertMores(t, gc, "Rich", constants.ClassSuperClass, []string{"BaseA", "BaseC"})

	// 无继承的类不带 superclass
	AssertNoMores(t, gc, "BaseA", constants.ClassSuperClass)

	// 派生类的自有成员照常收集
	AssertFieldsOf(t, gc, "Multi", []string{"m"})
	AssertFieldsOf(t, gc, "Rich", []string{"rich_field"})
	AssertDef(t, gc, "Rich::rich_method()", model.MethodDecl)
	AssertDef(t, gc, "Overrider::fa()", model.MethodDecl)
	AssertDef(t, gc, "Overrider::extra()", model.MethodDecl)
	AssertDef(t, gc, "FinalDerived::fb()", model.MethodDecl)
}

// =============================================================================
// 9. 枚举
// =============================================================================

func TestCollectorCppEnum(t *testing.T) {
	gc := collectCpp(t, "enum")

	// 普通 enum
	AssertDef(t, gc, "Color", model.Enum)
	AssertNoMores(t, gc, "Color", constants.EnumIsScoped)
	for _, c := range []string{"RED", "GREEN", "BLUE"} {
		AssertDef(t, gc, "Color::"+c, model.EnumConstant)
	}

	// enum class
	AssertDef(t, gc, "Scoped", model.Enum)
	AssertMores(t, gc, "Scoped", constants.EnumIsScoped, true)
	AssertDef(t, gc, "Scoped::SA", model.EnumConstant)

	// 指定底层类型
	AssertDef(t, gc, "Small", model.Enum)
	AssertMores(t, gc, "Small", constants.EnumIsScoped, true)
	AssertDef(t, gc, "Small::MIN", model.EnumConstant)
	AssertDef(t, gc, "Small::MAX", model.EnumConstant)
	AssertDef(t, gc, "Tiny", model.Enum)
	AssertNoMores(t, gc, "Tiny", constants.EnumIsScoped)
	AssertDef(t, gc, "Tiny::T_NEG", model.EnumConstant)

	// 前向声明 + 定义：定义覆盖，不产生 $2
	AssertDef(t, gc, "FwdEnum", model.Enum)
	AssertNoDef(t, gc, "FwdEnum$2")
	AssertDef(t, gc, "FwdEnum::FE_A", model.EnumConstant)

	// 匿名 enum：常量挂文件级，不产 ENUM 元素
	AssertDef(t, gc, "ANON_X", model.EnumConstant)
	AssertDef(t, gc, "ANON_Y", model.EnumConstant)

	// 显式值
	AssertDef(t, gc, "Flags", model.Enum)
	for _, c := range []string{"F_NONE", "F_READ", "F_WRITE", "F_ALL"} {
		AssertDef(t, gc, "Flags::"+c, model.EnumConstant)
	}

	// 枚举类型变量
	AssertDef(t, gc, "current_color", model.Variable)
	AssertMores(t, gc, "current_color", constants.VariableRawType, "Color")
	AssertDef(t, gc, "current_scoped", model.Variable)
	AssertMores(t, gc, "current_scoped", constants.VariableRawType, "Scoped")
}

// =============================================================================
// 10. typedef / using
// =============================================================================

func TestCollectorCppTypedefUsing(t *testing.T) {
	gc := collectCpp(t, "typedef_using")

	// 基础别名
	AssertDef(t, gc, "ulong_t", model.Type)
	AssertMores(t, gc, "ulong_t", constants.TypeUnderlying, "unsigned long")
	AssertDef(t, gc, "my_int", model.Type)
	AssertMores(t, gc, "my_int", constants.TypeUnderlying, "int")

	// 指针 / 引用 / 数组别名
	AssertDef(t, gc, "int_ptr_t", model.Type)
	AssertMores(t, gc, "int_ptr_t", constants.TypeUnderlying, "int*")
	AssertDef(t, gc, "int_ref_t", model.Type)
	AssertMores(t, gc, "int_ref_t", constants.TypeUnderlying, "int&")
	AssertDef(t, gc, "int_arr_t", model.Type)
	AssertMores(t, gc, "int_arr_t", constants.TypeUnderlying, "int[]")

	// 函数指针别名（底层文本不展开函数签名，仅断言元素存在）
	AssertDef(t, gc, "callback_t", model.Type)
	AssertDef(t, gc, "void_cb_t", model.Type)

	// 多声明器：每个都产 TYPE
	for _, n := range []string{"TI1", "TI2", "PTI3"} {
		AssertDef(t, gc, n, model.Type)
	}
	AssertMores(t, gc, "TI1", constants.TypeUnderlying, "int")
	AssertMores(t, gc, "TI2", constants.TypeUnderlying, "int")
	AssertMores(t, gc, "PTI3", constants.TypeUnderlying, "int*")

	// 链式别名
	AssertDef(t, gc, "my_ulong_t", model.Type)
	AssertMores(t, gc, "my_ulong_t", constants.TypeUnderlying, "ulong_t")

	// using 别名
	AssertDef(t, gc, "UInt", model.Type)
	AssertMores(t, gc, "UInt", constants.TypeUnderlying, "unsigned int")
	AssertDef(t, gc, "UIntPtr", model.Type)
	AssertMores(t, gc, "UIntPtr", constants.TypeUnderlying, "UInt*")
	AssertDef(t, gc, "Callback", model.Type)

	// 别名模板
	AssertDef(t, gc, "Vec", model.Type)
	AssertMores(t, gc, "Vec", constants.TemplateParams, "<typename T>")
	AssertMores(t, gc, "Vec", constants.TypeUnderlying, "T*")

	// 具名 struct + 别名：结构体与别名各自独立
	AssertDef(t, gc, "Named", model.Struct)
	AssertFieldsOf(t, gc, "Named", []string{"n"})
	AssertDef(t, gc, "Named_t", model.Type)
	AssertMores(t, gc, "Named_t", constants.TypeUnderlying, "Named")
	AssertDef(t, gc, "NamedAlias", model.Type)

	// 匿名 struct typedef：字段挂 typedef 名
	AssertDef(t, gc, "Point", model.Type)
	AssertFieldsOf(t, gc, "Point", []string{"x", "y"})

	// 别名使用
	for _, n := range []string{"g_ulong", "g_myint", "g_ptr", "g_cb", "g_uint"} {
		AssertDef(t, gc, n, model.Variable)
	}
	AssertMores(t, gc, "g_ulong", constants.VariableRawType, "ulong_t")
}

// =============================================================================
// 11. 变量与作用域
// =============================================================================

func TestCollectorCppVariable(t *testing.T) {
	gc := collectCpp(t, "variable")

	// 全局变量族
	AssertDef(t, gc, "g_counter", model.Variable)
	AssertMores(t, gc, "g_counter", constants.VariableIsGlobal, true)
	AssertDef(t, gc, "g_static", model.Variable)
	AssertMores(t, gc, "g_static", constants.VariableIsStatic, true)
	AssertDef(t, gc, "g_const", model.Variable)
	AssertMores(t, gc, "g_const", constants.VariableIsConst, true)
	// constexpr / thread_local / alignas：无对应 Mores 键，仅断言元素存在
	for _, n := range []string{"g_constexpr", "g_tl", "g_aligned"} {
		AssertDef(t, gc, n, model.Variable)
		AssertMores(t, gc, n, constants.VariableIsGlobal, true)
	}
	// extern 声明
	AssertDef(t, gc, "g_extern", model.Variable)
	AssertMores(t, gc, "g_extern", constants.VariableIsExtern, true)

	// auto / 引用 / 指针 / 数组
	AssertDef(t, gc, "g_auto", model.Variable)
	AssertMores(t, gc, "g_auto", constants.VariableRawType, "auto")
	for _, n := range []string{
		"g_ref", "g_rref", "g_ptr", "g_pptr", "g_ptr_to_const", "g_const_ptr", "g_arr", "g_arr2",
	} {
		AssertDef(t, gc, n, model.Variable)
	}
	AssertMores(t, gc, "g_ptr_to_const", constants.VariableIsConst, true)

	// 多声明器：每个都产 VARIABLE
	for _, n := range []string{"g_m1", "g_m2", "g_m3"} {
		AssertDef(t, gc, n, model.Variable)
	}

	// 初始化形式
	AssertDef(t, gc, "g_brace", model.Variable)
	AssertDef(t, gc, "g_paren", model.Variable)

	// 局部变量
	AssertDef(t, gc, "var_user(int)::local", model.Variable)
	AssertNoMores(t, gc, "var_user(int)::local", constants.VariableIsGlobal)
	AssertDef(t, gc, "var_user(int)::local_static", model.Variable)
	AssertMores(t, gc, "var_user(int)::local_static", constants.VariableIsStatic, true)
	AssertDef(t, gc, "var_user(int)::local_const", model.Variable)
	AssertMores(t, gc, "var_user(int)::local_const", constants.VariableIsConst, true)
	for _, n := range []string{"local_constexpr", "local_tl", "l_auto_int", "l_auto_ptr"} {
		AssertDef(t, gc, "var_user(int)::"+n, model.Variable)
	}

	// 引用与指针
	for _, n := range []string{"l_ref", "l_rref", "l_ptr", "l_pptr", "l_cptr"} {
		AssertDef(t, gc, "var_user(int)::"+n, model.Variable)
	}

	// 数组
	AssertDef(t, gc, "var_user(int)::l_arr", model.Variable)
	AssertDef(t, gc, "var_user(int)::l_arr2", model.Variable)

	// 多声明器
	for _, n := range []string{"l_m1", "l_m2", "l_m3"} {
		AssertDef(t, gc, "var_user(int)::"+n, model.Variable)
	}

	// 结构化绑定：每个绑定名各产一个变量
	AssertDef(t, gc, "var_user(int)::sb_first", model.Variable)
	AssertDef(t, gc, "var_user(int)::sb_second", model.Variable)

	// 参数作为变量元素
	AssertDef(t, gc, "var_user(int)::param", model.Variable)

	// 块作用域
	AssertDef(t, gc, "var_user(int)::block$1", model.ScopeBlock)
	AssertDef(t, gc, "var_user(int)::block$1::inner_block", model.Variable)
}

// =============================================================================
// 12. 函数
// =============================================================================

func TestCollectorCppFunction(t *testing.T) {
	gc := collectCpp(t, "function")

	// 无参 / void 参数：QN 均为 ()
	AssertDef(t, gc, "no_args()", model.Function)
	AssertDef(t, gc, "void_args()", model.Function)
	AssertNoMores(t, gc, "void_args()", constants.FunctionParameters)

	// 可变参数
	AssertDef(t, gc, "varargs(const char*)", model.Function)
	AssertMores(t, gc, "varargs(const char*)", constants.FunctionIsVariadic, true)
	AssertMores(t, gc, "varargs(const char*)", constants.FunctionParameters, []string{"const char* fmt", "..."})

	// noexcept（无对应 Mores 键，仅断言元素存在）
	AssertDef(t, gc, "noex()", model.Function)
	AssertDef(t, gc, "noex_cond()", model.Function)

	// constexpr
	AssertDef(t, gc, "cx(int)", model.Function)
	AssertMores(t, gc, "cx(int)", constants.FunctionReturnType, "int")

	// 尾置返回类型：返回类型取 auto（不解析 -> int）
	AssertDef(t, gc, "trailing(int)", model.Function)
	AssertMores(t, gc, "trailing(int)", constants.FunctionReturnType, "auto")

	// 默认实参不进入 QN 签名
	AssertDef(t, gc, "with_defaults(int,int,int)", model.Function)
	AssertMores(t, gc, "with_defaults(int,int,int)", constants.FunctionParameters, []string{"int a", "int b", "int c"})

	// 重载组：按参数类型区分 QN
	AssertDef(t, gc, "overloaded(int)", model.Function)
	AssertDef(t, gc, "overloaded(int,int)", model.Function)
	AssertDef(t, gc, "overloaded(double)", model.Function)

	// static / inline
	AssertDef(t, gc, "static_fn()", model.Function)
	AssertDef(t, gc, "inline_fn()", model.Function)

	// extern "C"：声明产 FunctionDecl，变量照常收集
	AssertDef(t, gc, "c_linkage_fn()", model.FunctionDecl)
	AssertDef(t, gc, "c_linkage_var", model.Variable)

	// 返回指针 / 引用（reference_declarator 需正确剥壳）
	AssertDef(t, gc, "ptr_ret()", model.Function)
	AssertMores(t, gc, "ptr_ret()", constants.FunctionReturnType, "int")
	AssertDef(t, gc, "ref_ret()", model.Function)
	AssertMores(t, gc, "ref_ret()", constants.FunctionReturnType, "int")

	// 递归
	AssertDef(t, gc, "factorial(int)", model.Function)
	AssertMores(t, gc, "factorial(int)", constants.FunctionParameters, []string{"int n"})
}

// =============================================================================
// 13. 参数形态
// =============================================================================

func TestCollectorCppParam(t *testing.T) {
	gc := collectCpp(t, "param")

	// 值参数
	AssertDef(t, gc, "by_value(int,double,char)", model.Function)
	AssertSignature(t, gc, "by_value(int,double,char)", []string{"int a", "double b", "char c"})

	// 指针 / 多级指针
	AssertDef(t, gc, "by_pointer(int*,int**,char***)", model.Function)
	AssertSignature(t, gc, "by_pointer(int*,int**,char***)", []string{"int* p", "int** pp", "char*** ppp"})

	// 引用 / 右值引用：QN 与签名都区分 & 与 &&
	AssertDef(t, gc, "by_ref(int&,const int&,int&&)", model.Function)
	AssertSignature(t, gc, "by_ref(int&,const int&,int&&)", []string{"int& r", "const int& cr", "int&& rr"})

	// const 各种位置
	AssertDef(t, gc, "by_const(const int,const int*,int*,const int&)", model.Function)
	AssertSignature(t, gc, "by_const(const int,const int*,int*,const int&)",
		[]string{"const int a", "const int* p", "int* cp", "const int& r"})

	// 数组参数：退化为 []
	AssertDef(t, gc, "by_array(int[],int[],int[][])", model.Function)
	AssertSignature(t, gc, "by_array(int[],int[],int[][])", []string{"int[] arr", "int[] arr2", "int[][] arr3"})

	// 函数指针参数：记 (*)
	AssertDef(t, gc, "by_funcptr(int (*),void (*))", model.Function)
	AssertSignature(t, gc, "by_funcptr(int (*),void (*))", []string{"int (*) cb", "void (*) cb2"})

	// 默认值不进入签名
	AssertDef(t, gc, "by_default(int,int,const char*)", model.Function)
	AssertSignature(t, gc, "by_default(int,int,const char*)", []string{"int a", "int b", "const char* s"})

	// 可变参数
	AssertDef(t, gc, "by_variadic(int)", model.Function)
	AssertMores(t, gc, "by_variadic(int)", constants.FunctionIsVariadic, true)
	AssertSignature(t, gc, "by_variadic(int)", []string{"int first", "..."})

	// 无名参数：保留类型
	AssertDef(t, gc, "by_unnamed(int,double,int*)", model.Function)
	AssertSignature(t, gc, "by_unnamed(int,double,int*)", []string{"int", "double", "int*"})

	// struct / enum 类型参数
	AssertDef(t, gc, "by_user_type(PStruct,PEnum,PStruct*)", model.Function)
	AssertSignature(t, gc, "by_user_type(PStruct,PEnum,PStruct*)", []string{"PStruct s", "PEnum e", "PStruct* ps"})

	// 模板参数
	AssertDef(t, gc, "by_template(T,T*,T&)", model.Function)
	AssertSignature(t, gc, "by_template(T,T*,T&)", []string{"T v", "T* pv", "T& rv"})
}

// =============================================================================
// 14. Lambda
// =============================================================================

func TestCollectorCppLambda(t *testing.T) {
	gc := collectCpp(t, "lambda")

	// Lambda 作字段初始值：挂字段所属类下
	AssertDef(t, gc, "Holder::lambda$1", model.Lambda)
	AssertDef(t, gc, "Holder::lambda$1::x", model.Variable)

	// 变量初始化 lambda：每个变量都关联到对应 Lambda 元素
	want := map[string]string{
		"no_cap":       "lambda_user()::lambda$1",
		"by_val":       "lambda_user()::lambda$2",
		"by_ref":       "lambda_user()::lambda$3",
		"all_val":      "lambda_user()::lambda$4",
		"all_ref":      "lambda_user()::lambda$5",
		"mut":          "lambda_user()::lambda$6",
		"generic":      "lambda_user()::lambda$7",
		"generic_pack": "lambda_user()::lambda$8",
		"outer_lam":    "lambda_user()::lambda$9",
		"call_with":    "lambda_user()::lambda$10",
	}
	for name, lambdaQN := range want {
		AssertDef(t, gc, "lambda_user()::"+name, model.Variable)
		AssertMores(t, gc, "lambda_user()::"+name, constants.VariableLambdaQN, lambdaQN)
		AssertDef(t, gc, lambdaQN, model.Lambda)
	}

	// generic lambda 的形参
	AssertDef(t, gc, "lambda_user()::lambda$7::a", model.Variable)
	AssertDef(t, gc, "lambda_user()::lambda$7::b", model.Variable)

	// 嵌套 lambda：内层挂在外层 Lambda 下
	AssertDef(t, gc, "lambda_user()::lambda$9::lambda$1", model.Lambda)
	AssertDef(t, gc, "lambda_user()::lambda$9::inner_lam", model.Variable)
	AssertMores(t, gc, "lambda_user()::lambda$9::inner_lam", constants.VariableLambdaQN,
		"lambda_user()::lambda$9::lambda$1")

	// Lambda 作返回值（无变量绑定）
	AssertDef(t, gc, "make_adder(int)::lambda$1", model.Lambda)

	// 立即调用 lambda：call_expression 形态，不产变量绑定
	AssertDef(t, gc, "lambda_user()::immediate", model.Variable)
	AssertNoMores(t, gc, "lambda_user()::immediate", constants.VariableLambdaQN)
}

// =============================================================================
// 15. 作用域
// =============================================================================

func TestCollectorCppScope(t *testing.T) {
	gc := collectCpp(t, "scope")

	// 函数级变量
	AssertDef(t, gc, "scope_user(int)::n", model.Variable)
	AssertDef(t, gc, "scope_user(int)::fn_level", model.Variable)
	AssertDef(t, gc, "scope_user(int)::arr", model.Variable)
	AssertDef(t, gc, "scope_user(int)::i", model.Variable)

	// 各类语句体各产一个 ScopeBlock，局部变量挂其下
	cases := map[string]string{
		"block$1":  "in_bare",
		"block$2":  "in_if",
		"block$3":  "in_else",
		"block$4":  "in_for",
		"block$5":  "in_range",
		"block$6":  "in_while",
		"block$7":  "in_do",
		"block$9":  "in_try",
		"block$10": "in_catch",
		"block$11": "outer_block",
	}
	for block, v := range cases {
		AssertDef(t, gc, "scope_user(int)::"+block, model.ScopeBlock)
		AssertDef(t, gc, "scope_user(int)::"+block+"::"+v, model.Variable)
	}

	// switch 的 case 体各自成块
	AssertDef(t, gc, "scope_user(int)::block$8", model.ScopeBlock)
	AssertDef(t, gc, "scope_user(int)::block$8::block$1", model.ScopeBlock)
	AssertDef(t, gc, "scope_user(int)::block$8::block$1::in_case", model.Variable)
	AssertDef(t, gc, "scope_user(int)::block$8::block$2", model.ScopeBlock)
	AssertDef(t, gc, "scope_user(int)::block$8::block$2::in_default", model.Variable)

	// 嵌套 block
	AssertDef(t, gc, "scope_user(int)::block$11::block$1", model.ScopeBlock)
	AssertDef(t, gc, "scope_user(int)::block$11::block$1::inner_block", model.Variable)

	// 块内重名：不同作用域各自独立（QN 不同）
	AssertDef(t, gc, "scope_user(int)::block$12::dup_name", model.Variable)
	AssertDef(t, gc, "scope_user(int)::block$13::dup_name", model.Variable)

	// 函数内局部类型
	AssertDef(t, gc, "scope_user(int)::LocalStruct", model.Struct)
	AssertFieldsOf(t, gc, "scope_user(int)::LocalStruct", []string{"lv"})
	AssertDef(t, gc, "scope_user(int)::LocalClass", model.Class)
	AssertFieldsOf(t, gc, "scope_user(int)::LocalClass", []string{"lc"})
	AssertDef(t, gc, "scope_user(int)::LocalEnum", model.Enum)
	AssertDef(t, gc, "scope_user(int)::LocalEnum::LE_A", model.EnumConstant)
}

// =============================================================================
// 16. 宏与预处理
// =============================================================================

func TestCollectorCppMacro(t *testing.T) {
	gc := collectCpp(t, "macro")

	// 对象宏
	AssertDef(t, gc, "OBJ_MACRO", model.Macro)
	AssertMores(t, gc, "OBJ_MACRO", constants.MacroBody, "100")
	AssertMores(t, gc, "OBJ_MACRO", constants.MacroIsFunctionLike, false)
	AssertMores(t, gc, "OBJ_MACRO", constants.MacroName, "OBJ_MACRO")

	// 函数式宏（单参 / 多参 / 可变参数）
	AssertDef(t, gc, "FUNC_MACRO", model.Macro)
	AssertMores(t, gc, "FUNC_MACRO", constants.MacroIsFunctionLike, true)
	AssertDef(t, gc, "ADD_MACRO", model.Macro)
	AssertMores(t, gc, "ADD_MACRO", constants.MacroIsFunctionLike, true)
	AssertDef(t, gc, "VAR_MACRO", model.Macro)
	AssertMores(t, gc, "VAR_MACRO", constants.MacroIsFunctionLike, true)

	// 多行宏
	AssertDef(t, gc, "MULTI_LINE", model.Macro)
	AssertMores(t, gc, "MULTI_LINE", constants.MacroIsFunctionLike, true)

	// 空宏：有元素、无 body
	AssertDef(t, gc, "EMPTY_MACRO", model.Macro)
	AssertMores(t, gc, "EMPTY_MACRO", constants.MacroIsFunctionLike, false)
	AssertNoMores(t, gc, "EMPTY_MACRO", constants.MacroBody)

	// 字符串化 / 连接
	AssertDef(t, gc, "STR_MACRO", model.Macro)
	AssertDef(t, gc, "CAT_MACRO", model.Macro)

	// 条件分支内同名宏：全分支保留，各自标注 condition
	AssertDef(t, gc, "FEATURE_VAL", model.Macro)
	AssertMores(t, gc, "FEATURE_VAL", constants.Condition, "#ifdef FEATURE_Y")
	AssertMores(t, gc, "FEATURE_VAL", constants.MacroBody, "1")
	AssertDef(t, gc, "FEATURE_VAL$2", model.Macro)
	AssertMores(t, gc, "FEATURE_VAL$2", constants.Condition, "#else")
	AssertMores(t, gc, "FEATURE_VAL$2", constants.MacroBody, "0")

	// #undef 不产元素（OBJ_MACRO 只出现一次）
	AssertNoDef(t, gc, "OBJ_MACRO$2")
}

// =============================================================================
// 17. 条件编译
// =============================================================================

func TestCollectorCppConditional(t *testing.T) {
	gc := collectCpp(t, "cond")

	// 各分支的 condition 标签
	cases := map[string]string{
		"cond_a":  "#ifdef FEATURE_A",
		"cond_b":  "#ifndef FEATURE_B",
		"cond_c":  "#if defined(FEATURE_C) && FEATURE_C > 0",
		"cond_d1": "#if FEATURE_D == 1",
		"cond_d2": "#elif FEATURE_D == 2",
		"cond_d3": "#else",
		"cond_e1": "#ifdef FEATURE_E",
		"cond_e2": "#else",
	}
	for qn, cond := range cases {
		AssertDef(t, gc, qn, model.Variable)
		AssertMores(t, gc, qn, constants.Condition, cond)
	}

	// 嵌套条件取最内层标签
	AssertMores(t, gc, "cond_nested", constants.Condition, "#ifdef INNER_F")
	AssertMores(t, gc, "cond_outer_only", constants.Condition, "#ifdef OUTER_F")

	// 条件分支内的类型与函数照常收集
	AssertDef(t, gc, "CondStruct", model.Struct)
	AssertMores(t, gc, "CondStruct", constants.Condition, "#ifdef FEATURE_G")
	AssertFieldsOf(t, gc, "CondStruct", []string{"cv"})
	AssertDef(t, gc, "cond_func()", model.FunctionDecl)
	AssertMores(t, gc, "cond_func()", constants.Condition, "#ifdef FEATURE_G")
}

// =============================================================================
// 18. 边界与负例
// =============================================================================

func TestCollectorCppEdge(t *testing.T) {
	gc := collectCpp(t, "edge")

	// 同名不同 kind 共存
	AssertDef(t, gc, "DupTag", model.Struct)
	AssertFieldsOf(t, gc, "DupTag", []string{"a"})
	AssertDef(t, gc, "DupTag2", model.Class)
	AssertFieldsOf(t, gc, "DupTag2", []string{"b"})

	// 同名变量在不同作用域：QN 不同，不消歧
	AssertDef(t, gc, "dup_var", model.Variable)
	AssertDef(t, gc, "dup_ns::dup_var", model.Variable)

	// 同名函数重载：按参数区分，不产生 $N
	AssertDef(t, gc, "dup_fn(int)", model.Function)
	AssertDef(t, gc, "dup_fn(int,int)", model.Function)
	AssertNoDef(t, gc, "dup_fn(int)$2")

	// 匿名 struct 承载：typedef 名
	AssertDef(t, gc, "AnonPoint", model.Type)
	AssertFieldsOf(t, gc, "AnonPoint", []string{"tx", "ty"})

	// 匿名 struct 承载：变量名
	AssertDef(t, gc, "anon_obj", model.Variable)
	AssertFieldsOf(t, gc, "anon_obj", []string{"vx"})

	// 匿名 union 承载：外层字段名
	AssertFieldsOf(t, gc, "WithAnonUnion", []string{"payload"})
	AssertFieldsOf(t, gc, "WithAnonUnion::payload", []string{"as_int", "as_float"})

	// 类型引用不产元素（RefTarget 只出现一次，不产生 $2）
	AssertDef(t, gc, "RefTarget", model.Struct)
	AssertNoDef(t, gc, "RefTarget$2")
	AssertFieldsOf(t, gc, "RefTarget", []string{"rv"})
	AssertDef(t, gc, "use_ref()::local", model.Variable)
	AssertDef(t, gc, "use_ref()::ptr", model.Variable)

	// 前向声明：产元素，无字段
	AssertDef(t, gc, "FwdOnly", model.Struct)
	AssertNoMores(t, gc, "FwdOnly", constants.StructIsUnion)
	AssertFieldsOf(t, gc, "FwdOnly", nil)

	// 空 struct / 空函数
	AssertDef(t, gc, "EmptyStruct", model.Struct)
	AssertFieldsOf(t, gc, "EmptyStruct", nil)
	AssertDef(t, gc, "empty_fn()", model.Function)

	// 只声明不定义
	AssertDef(t, gc, "declared_only(int)", model.FunctionDecl)
	AssertNoKind(t, gc, "declared_only(int)", model.Function)

	// 无名参数：保留类型
	AssertDef(t, gc, "unnamed_params(int,char*)", model.FunctionDecl)
	AssertSignature(t, gc, "unnamed_params(int,char*)", []string{"int", "char*"})

	// 负例：宏体中的类型名不产元素
	AssertDef(t, gc, "FAKE_CLASS", model.Macro)
	AssertNoDef(t, gc, "FakeClass")
}

// =============================================================================
// 19. 展示层字段（NamespaceQN / EnclosingQN / TypeQN）
// =============================================================================

// TestCollectorCppDisplayFields 覆盖展示层三字段：
// L1 NamespaceQN（最近外层 namespace）、L2 EnclosingQN（直接词法外层）、
// L2 TypeQN（最外层具名宿主类型）。
func TestCollectorCppDisplayFields(t *testing.T) {
	t.Run("namespace 与嵌套类型", func(t *testing.T) {
		gc := collectCpp(t, "namespace")

		// namespace 元素：NamespaceQN 取父 namespace
		AssertDisplayFields(t, gc, "outer", "", "", "")
		AssertDisplayFields(t, gc, "outer::middle", "outer", "outer", "")
		AssertDisplayFields(t, gc, "outer::middle::inner", "outer::middle", "outer::middle", "")
		// namespace 内类型：L1=最近 namespace，L2 宿主=自身
		AssertDisplayFields(t, gc, "outer::middle::inner::Deep",
			"outer::middle::inner", "outer::middle::inner", "outer::middle::inner::Deep")
		AssertDisplayFields(t, gc, "modern::sub::Modern",
			"modern::sub", "modern::sub", "modern::sub::Modern")
	})

	t.Run("嵌套类与成员", func(t *testing.T) {
		gc := collectCpp(t, "class")

		// 顶层类型：TypeQN 取自身
		AssertDisplayFields(t, gc, "Outer", "", "", "Outer")
		// 嵌套类型：TypeQN 取最外层具名类型
		AssertDisplayFields(t, gc, "Outer::Inner", "", "Outer", "Outer")
		AssertDisplayFields(t, gc, "Outer::NestedStruct", "", "Outer", "Outer")
		// 成员：EnclosingQN=直接宿主，TypeQN=最外层
		AssertDisplayFields(t, gc, "Outer::Inner::deep", "", "Outer::Inner", "Outer")
		AssertDisplayFields(t, gc, "Outer::pub_method()", "", "Outer", "Outer")
		AssertDisplayFields(t, gc, "Outer::static_field", "", "Outer", "Outer")
	})

	t.Run("自由函数与全局变量无宿主类型", func(t *testing.T) {
		gc := collectCpp(t, "function")

		AssertDisplayFields(t, gc, "factorial(int)", "", "", "")
		// 局部变量：EnclosingQN=函数，TypeQN 空
		elem := FindDef(t, gc, "factorial(int)")
		if elem.TypeQN != "" {
			t.Errorf("factorial TypeQN = %q, want empty", elem.TypeQN)
		}
		// 函数参数变量：EnclosingQN=函数
		AssertDisplayFields(t, gc, "factorial(int)::n", "", "factorial(int)", "")
	})

	t.Run("跨文件类外定义", func(t *testing.T) {
		paths := ResolveTree(t, "extractor/impl_link")
		gc, _ := RunExtraction(t, "x/cpp/testdata", paths)

		// 类外定义在 impl_a.cpp：父类型定义在头文件，TypeQN 仍按 QN 结构推导
		AssertDisplayFields(t, gc, "impl::Service::member(int)", "impl", "impl::Service", "impl::Service")
		// 普通自由函数：无宿主类型
		AssertDisplayFields(t, gc, "impl::split_a(int)", "impl", "impl", "")
	})

	t.Run("File 元素三字段为空", func(t *testing.T) {
		gc := collectCpp(t, "base")
		for _, def := range gc.Definitions() {
			if def.Element.Kind != model.File {
				continue
			}
			if def.Element.NamespaceQN != "" || def.Element.EnclosingQN != "" || def.Element.TypeQN != "" {
				t.Errorf("File %s display fields should be empty, got NS=%q ENC=%q TYPE=%q",
					def.Element.QualifiedName, def.Element.NamespaceQN, def.Element.EnclosingQN, def.Element.TypeQN)
			}
		}
	})
}
