package test

import (
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/constants"
)

// coreGlobalContext 是 core.GlobalContext 的别名（测试辅助签名用）。
type coreGlobalContext = core.GlobalContext

// TestCollectorC 覆盖 C 语言元素收集：函数/声明/结构体/枚举/typedef/宏/全局与局部变量。
func TestCollectorC(t *testing.T) {
	paths := ResolvePaths(t, "collector/base", "*.c", "*.h")
	gc := RunCollection(t, "x/cpp/testdata", paths)

	// --- 函数定义与声明成对 ---
	AssertDef(t, gc, "add(int,int)", model.Function)
	AssertDef(t, gc, "internal_helper(int)", model.Function)
	AssertDef(t, gc, "process()", model.Function)

	// 声明（头文件原型）与定义共存：FindByQualifiedName 定义优先。
	if entry, ok := gc.FindByQualifiedName("add(int,int)"); !ok || entry.Element.Kind != model.Function {
		t.Errorf("add(int,int) should resolve to definition, got %v", entry)
	}

	// --- 结构体与字段 ---
	AssertDef(t, gc, "Node", model.Struct)
	AssertDef(t, gc, "Node::value", model.Field)
	AssertDef(t, gc, "Node::next", model.Field)

	// --- 匿名 struct 承载实体：typedef struct {...} Point → 字段挂 Point ---
	AssertDef(t, gc, "Point", model.Type)
	AssertDef(t, gc, "Point::x", model.Field)
	AssertDef(t, gc, "Point::y", model.Field)

	// --- typedef 别名 ---
	AssertDef(t, gc, "ulong_t", model.Type)
	AssertMores(t, gc, "ulong_t", constants.TypeUnderlying, "unsigned long")

	// --- 枚举 ---
	AssertDef(t, gc, "Color", model.Enum)
	AssertDef(t, gc, "Color::RED", model.EnumConstant)
	AssertDef(t, gc, "Color::GREEN", model.EnumConstant)
	AssertDef(t, gc, "Color::BLUE", model.EnumConstant)

	// --- 宏（含函数式宏） ---
	AssertDef(t, gc, "MAX_SIZE", model.Macro)
	AssertMores(t, gc, "MAX_SIZE", constants.MacroBody, "100")
	AssertMores(t, gc, "MAX_SIZE", constants.MacroIsFunctionLike, false)
	AssertDef(t, gc, "SQUARE", model.Macro)
	AssertMores(t, gc, "SQUARE", constants.MacroIsFunctionLike, true)

	// --- 全局变量与局部变量 ---
	// 注：头文件 extern 声明与源文件定义同 QN 同 kind，跨文件注册顺序不定，
	// 故按文件上下文分别校验（basic.c 的定义必须标记 is_global）。
	AssertDef(t, gc, "global_counter", model.Variable)
	if fc, ok := gc.FileContext("collector/base/basic.c"); ok {
		if entry, found := fc.FindByQualifiedName("global_counter"); found {
			if !isTrue(entry.Element.GetMores()[constants.VariableIsGlobal]) {
				t.Errorf("basic.c global_counter should be marked is_global")
			}
		} else {
			t.Errorf("basic.c global_counter not found")
		}
	}
	AssertDef(t, gc, "add(int,int)::local", model.Variable)

	// --- 局部类型引用不应产元素（struct Node n; / enum Color c;） ---
	AssertNoDef(t, gc, "process()::Node")
	AssertNoDef(t, gc, "process()::Color")

	// --- 参数作为变量元素 ---
	AssertDef(t, gc, "add(int,int)::a", model.Variable)
	AssertDef(t, gc, "add(int,int)::b", model.Variable)

	// --- static 函数不参与 ImplLink，但元素存在且带 static 修饰符 ---
	AssertDef(t, gc, "internal_helper(int)", model.Function)
	if elem := FindDef(t, gc, "internal_helper(int)"); elem.Extra == nil || !containsStr(elem.Extra.Modifiers, "static") {
		t.Errorf("internal_helper should carry static modifier, got %v", elem.Extra)
	}
}

// findDefInFile 在指定文件上下文中按 QN + kind 查找定义（同文件声明/定义共存时按 kind 区分）。
func findDefInFile(gc coreGlobalContext, filePath, qn string, kind model.ElementKind) *model.CodeElement {
	fc, ok := gc.FileContext(filePath)
	if !ok {
		return nil
	}
	for _, def := range fc.Definitions() {
		if def.Element.QualifiedName == qn && (kind == "" || def.Element.Kind == kind) {
			return def.Element
		}
	}
	return nil
}

// assertDeclInFile 在指定文件上下文中断言声明元素存在。
func assertDeclInFile(t *testing.T, gc coreGlobalContext, filePath, qn string, wantKind model.ElementKind) {
	t.Helper()
	if findDefInFile(gc, filePath, qn, wantKind) == nil {
		t.Errorf("%s declaration (kind=%s) not found in %s", qn, wantKind, filePath)
	}
}

// assertDeclMores 在指定文件上下文中断言声明元素的 Mores。
func assertDeclMores(t *testing.T, gc coreGlobalContext, filePath, qn, key string, want interface{}) {
	t.Helper()
	elem := findDefInFile(gc, filePath, qn, model.MethodDecl)
	if elem == nil {
		t.Errorf("declaration not found in %s: %s", filePath, qn)
		return
	}
	got, exists := elem.GetMores()[key]
	if !exists {
		t.Errorf("%s Mores[%s] missing", qn, key)
		return
	}
	if !moresEqual(got, want) {
		t.Errorf("%s Mores[%s] = %v, want %v", qn, key, got, want)
	}
}

func isTrue(v interface{}) bool {
	b, _ := v.(bool)
	return b
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// TestCollectorCpp 覆盖 C++ 元素收集：namespace/class/继承/模板/纯虚/访问级别。
func TestCollectorCpp(t *testing.T) {
	paths := ResolvePaths(t, "collector/base", "*.cpp", "*.h")
	gc := RunCollection(t, "x/cpp/testdata", paths)

	// --- 命名空间层级 ---
	AssertDef(t, gc, "app", model.Namespace)
	AssertDef(t, gc, "app::core", model.Namespace)

	// --- 类与成员 ---
	AssertDef(t, gc, "app::core::Base", model.Class)
	AssertDef(t, gc, "app::core::Base::K", model.Field)
	AssertDef(t, gc, "app::core::Base::m_val", model.Field)
	// 声明元素（qnPrimary 定义优先，故直接查文件上下文）
	assertDeclInFile(t, gc, "collector/base/basic.cpp", "app::core::Base::run(int)", model.MethodDecl)
	assertDeclInFile(t, gc, "collector/base/basic.cpp", "app::core::Base::helper()", model.MethodDecl)

	// 纯虚函数
	AssertMores(t, gc, "app::core::Base::run(int)", constants.FunctionIsPureVirtual, true)
	AssertMores(t, gc, "app::core::Base::run(int)", constants.FunctionIsVirtual, true)

	// 访问级别
	AssertMores(t, gc, "app::core::Base::K", constants.Visibility, "public")
	AssertMores(t, gc, "app::core::Base::m_val", constants.Visibility, "protected")
	assertDeclMores(t, gc, "collector/base/basic.cpp", "app::core::Base::helper()", constants.Visibility, "private")

	// static 字段
	AssertMores(t, gc, "app::core::Base::K", constants.FieldIsStatic, true)

	// --- 类外定义（out-of-line）挂到类下：定义优先命中 METHOD ---
	if elem := FindDef(t, gc, "app::core::Base::helper()"); elem.Kind != model.Method {
		t.Errorf("Base::helper() should resolve to definition METHOD, got %s", elem.Kind)
	}
	AssertDef(t, gc, "app::core::Base::helper_impl", model.Variable)

	// --- 继承 ---
	AssertMores(t, gc, "app::Box", constants.ClassSuperClass, []string{"core::Base"})

	// --- 模板 ---
	AssertMores(t, gc, "app::Box", constants.TemplateParams, "<typename T>")
	AssertDef(t, gc, "app::Box::value", model.Field)
	AssertDef(t, gc, "app::Box::run(int)", model.Method)

	// --- enum class ---
	AssertDef(t, gc, "app::Scoped", model.Enum)
	AssertMores(t, gc, "app::Scoped", constants.EnumIsScoped, true)
	AssertDef(t, gc, "app::Scoped::A", model.EnumConstant)

	// --- 普通 struct ---
	AssertDef(t, gc, "app::Point3", model.Struct)
	AssertDef(t, gc, "app::Point3::x", model.Field)

	// --- 命名空间列表 ---
	fc, ok := gc.FileContext("collector/base/basic.cpp")
	if !ok {
		t.Fatalf("file context not found")
	}
	ns := fc.Namespaces()
	if len(ns) != 2 || ns[0] != "app" || ns[1] != "app::core" {
		t.Errorf("Namespaces = %v, want [app app::core]", ns)
	}
}

// TestCollectorConditional 覆盖条件编译分支标注。
func TestCollectorConditional(t *testing.T) {
	paths := ResolvePaths(t, "collector/cond", "*.c")
	gc := RunCollection(t, "x/cpp/testdata", paths)

	AssertMores(t, gc, "cond_a", constants.Condition, "#ifdef FOO")
	AssertMores(t, gc, "cond_b", constants.Condition, "#elif defined(BAR)")
	AssertMores(t, gc, "cond_c", constants.Condition, "#else")
	AssertMores(t, gc, "cond_d()", constants.Condition, "#ifndef BAZ")
}
