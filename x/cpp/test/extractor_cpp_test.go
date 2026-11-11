package test

import (
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/constants"
)

// TestExtractorCppStructural Import / Contain / Extend / Parameter / Return / TypeArg 全形态。
func TestExtractorCppStructural(t *testing.T) {
	paths := ResolveTree(t, "extractor/structural")
	// File → 顶层元素的 Contain 由 Linker 产出，故跑完整流水线。
	ec := RunPipeline(t, "x/cpp/testdata", paths, core.LevelRaw)
	rels := ec.Relations

	// --- Import：项目内头文件（同目录）---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Import &&
			r.Source.QualifiedName == "extractor/structural/structural.h" &&
			r.Target.QualifiedName == "extractor/structural/structural_dep.h" &&
			!r.Target.IsFormExternal
	})
	// --- Import：系统头归外部符号 ---
	sys := FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Import &&
			r.Source.QualifiedName == "extractor/structural/structural.cpp" &&
			r.Target.QualifiedName == "vector"
	})
	if !sys.Target.IsFormExternal {
		t.Errorf("system include target should be external")
	}
	if v, _ := sys.Mores[constants.RelIncludeIsSystem].(bool); !v {
		t.Errorf("system include is_system = false, want true")
	}
	// --- Import：嵌套路径头文件（路径后缀匹配）---
	nested := FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Import &&
			r.Source.QualifiedName == "extractor/structural/structural.cpp" &&
			r.Target.QualifiedName == "extractor/structural/nested/deep.h"
	})
	if nested.Target.IsFormExternal {
		t.Errorf("nested/deep.h should resolve to project File element, got external")
	}

	// --- Contain：File → 顶层元素 / Namespace → 元素 / 类 → 字段 / 函数 → 局部变量 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Contain && r.Source.Kind == model.File &&
			r.Target.QualifiedName == "st::Base"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Contain && r.Source.QualifiedName == "st" &&
			r.Target.QualifiedName == "st::Leaf"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Contain && r.Source.QualifiedName == "st::Holder" &&
			r.Target.QualifiedName == "st::Holder::dep"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Contain && r.Source.QualifiedName == "st::compute(Dep,int)" &&
			r.Target.QualifiedName == "st::compute(Dep,int)::d"
	})
	// Contain 去重（单文件内）：声明/定义同 QN 只产一条。
	impl := CppImpl(t)
	oneFile, err := impl.Extractor.Extract("extractor/structural/structural.cpp", ec.GC)
	if err != nil {
		t.Fatalf("extract structural.cpp: %v", err)
	}
	count := 0
	for _, r := range oneFile {
		if r.Type == model.Contain && r.Source.QualifiedName == "st::compute(Dep,int)" &&
			r.Target.QualifiedName == "st::compute(Dep,int)::d" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Contain(compute -> d) count in one file = %d, want 1 (dedup by QN)", count)
	}

	// --- Extend：单继承 / 多继承 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Extend && r.Source.QualifiedName == "st::Mid" &&
			r.Target.QualifiedName == "st::Base"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Extend && r.Source.QualifiedName == "st::Leaf" &&
			r.Target.QualifiedName == "st::Mid"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Extend && r.Source.QualifiedName == "st::Leaf" &&
			r.Target.QualifiedName == "st::Dep"
	})

	// --- Parameter：源码类型 + 基本类型 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Parameter && r.Source.QualifiedName == "st::compute(Dep,int)" &&
			r.Target.QualifiedName == "st::Dep"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Parameter && r.Source.QualifiedName == "st::compute(Dep,int)" &&
			r.Target.QualifiedName == "int"
	})
	// --- Return：源码类型 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Return && r.Source.QualifiedName == "st::make_dep()" &&
			r.Target.QualifiedName == "st::Dep"
	})
	// --- Return：void 不产 ---
	if HasRel(rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Return && r.Source.QualifiedName == "st::no_return(Dep)"
	}) {
		t.Errorf("void return should not produce RETURN relation")
	}

	// --- TypeArg：字段类型 + 模板实参 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.TypeArg && r.Source.QualifiedName == "st::Holder::dep" &&
			r.Target.QualifiedName == "st::Dep"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.TypeArg && r.Source.QualifiedName == "st::Holder::nested" &&
			r.Target.QualifiedName == "st::Holder"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.TypeArg && r.Source.QualifiedName == "st::Holder::nested" &&
			r.Target.QualifiedName == "st::Dep"
	})
	// --- TypeArg：局部变量类型 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.TypeArg && r.Source.QualifiedName == "st::compute(Dep,int)::d" &&
			r.Target.QualifiedName == "st::Dep"
	})
	// --- TypeArg：全局变量类型 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.TypeArg && r.Source.QualifiedName == "st::g_dep" &&
			r.Target.QualifiedName == "st::Dep"
	})
	// --- TypeArg：基本类型不产 ---
	if HasRel(rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.TypeArg && r.Source.QualifiedName == "st::g_count"
	}) {
		t.Errorf("primitive variable type should not produce TYPE_ARG")
	}
}

// TestExtractorCppActions Call / Create / Cast / Assign / Use / Throw 全形态。
func TestExtractorCppActions(t *testing.T) {
	paths := ResolveTree(t, "extractor/actions")
	_, rels := RunExtraction(t, "x/cpp/testdata", paths)

	// --- Call：自由函数重载按实参个数 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "act::caller(int)" &&
			r.Target.QualifiedName == "act::free_fn(int)"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "act::caller(int)" &&
			r.Target.QualifiedName == "act::free_fn(int,int)"
	})
	// --- Call：当前命名空间 / 限定命名空间 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "act::caller(int)" &&
			r.Target.QualifiedName == "act::ns_free()"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "act::caller(int)" &&
			r.Target.QualifiedName == "act::inner::ns_free()"
	})
	// --- Call：成员 / 指针成员 / 静态成员 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "act::caller(int)" &&
			r.Target.QualifiedName == "act::Widget::draw()"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "act::caller(int)" &&
			r.Target.QualifiedName == "act::Widget::create()"
	})
	// --- Call：可调用对象 obj(args) → operator() ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "act::caller(int)" &&
			r.Target.QualifiedName == "act::Widget::operator()(int)"
	})
	// --- Call：const 成员 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "act::caller(int)" &&
			r.Target.QualifiedName == "act::Target::get()"
	})

	// --- Create + 构造器 CALL ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Create && r.Source.QualifiedName == "act::maker()" &&
			r.Target.QualifiedName == "act::Widget"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "act::maker()" &&
			r.Target.QualifiedName == "act::Widget::Widget"
	})

	// --- Cast：C 风格 / static_cast / dynamic_cast / const_cast / reinterpret_cast ---
	castTargets := map[string]bool{}
	for _, r := range rels {
		if r.Type == model.Cast && r.Source.QualifiedName == "act::caster(void*,Base*)" {
			castTargets[r.Target.QualifiedName] = true
		}
	}
	for _, want := range []string{"act::Target", "act::Base"} {
		if !castTargets[want] {
			t.Errorf("CAST target %s not found (got %v)", want, castTargets)
		}
	}
	// (void)x 抑制不产 CAST
	if HasRel(rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Cast && r.Target.QualifiedName == "void"
	}) {
		t.Errorf("(void)x should not produce CAST")
	}

	// --- Assign：初始化 / 赋值 / 字段赋值 / 复合赋值 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Assign && r.Source.QualifiedName == "act::assigner(Target*,int)" &&
			r.Target.QualifiedName == "act::assigner(Target*,int)::local"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Assign && r.Source.QualifiedName == "act::assigner(Target*,int)" &&
			r.Target.QualifiedName == "act::Target::field"
	})
	// 复合赋值运算符记录
	compound := FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Assign && r.Source.QualifiedName == "act::assigner(Target*,int)" &&
			r.Target.QualifiedName == "act::assigner(Target*,int)::local" &&
			r.Mores[constants.RelAssignOperator] == "+="
	})
	if compound == nil {
		t.Errorf("compound assignment operator not recorded")
	}
	// 初始化标记
	init := FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Assign && r.Source.QualifiedName == "act::assigner(Target*,int)" &&
			r.Target.QualifiedName == "act::assigner(Target*,int)::local" &&
			r.Mores[constants.RelAssignIsInitializer] == true
	})
	if init == nil {
		t.Errorf("initializer flag not recorded")
	}

	// --- Use：字段 / 变量 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Use && r.Source.QualifiedName == "act::user(Target*,int)" &&
			r.Target.QualifiedName == "act::Target::field"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Use && r.Source.QualifiedName == "act::user(Target*,int)" &&
			r.Target.QualifiedName == "act::user(Target*,int)::a"
	})

	// --- Throw：throw new T / throw T(...) / throw var ---
	throwTargets := map[string]bool{}
	for _, r := range rels {
		if r.Type == model.Throw {
			throwTargets[r.Target.QualifiedName] = true
		}
	}
	if !throwTargets["act::MyError"] {
		t.Errorf("THROW target act::MyError not found (got %v)", throwTargets)
	}
	// 三种形态各一条
	throwCount := len(RelFilter(rels, model.Throw))
	if throwCount != 3 {
		t.Errorf("THROW count = %d, want 3 (new/call/var)", throwCount)
	}
}

// TestExtractorCppCapture Lambda 捕获推导全形态。
func TestExtractorCppCapture(t *testing.T) {
	paths := ResolveTree(t, "extractor/capture")
	_, rels := RunExtraction(t, "x/cpp/testdata", paths)

	caps := RelFilter(rels, model.Capture)
	// 值捕获 / 引用捕获 / 全捕获 / 嵌套 lambda / 赋值推导
	want := map[string]bool{
		"capture_cases(int)::by_value":      false,
		"capture_cases(int)::by_ref":        false,
		"capture_cases(int)::assigned":      false,
		"capture_cases(int)::lambda$6::mid": false,
	}
	for _, c := range caps {
		if c.Source.Kind != model.Lambda {
			t.Errorf("CAPTURE source kind = %s, want LAMBDA", c.Source.Kind)
		}
		if _, ok := want[c.Target.QualifiedName]; ok {
			want[c.Target.QualifiedName] = true
		}
	}
	for qn, found := range want {
		if !found {
			t.Errorf("CAPTURE target %s not found", qn)
		}
	}
	// 无捕获 lambda 不产 CAPTURE
	for _, c := range caps {
		if c.Source.QualifiedName == "capture_cases(int)::lambda$4" {
			t.Errorf("captureless lambda should not produce CAPTURE")
		}
	}
	// lambda 自身参数/局部变量不构成捕获
	for _, c := range caps {
		if c.Target.QualifiedName == "capture_cases(int)::lambda$5::p" ||
			c.Target.QualifiedName == "capture_cases(int)::lambda$5::inner" {
			t.Errorf("lambda-internal variable %s should not be CAPTURE", c.Target.QualifiedName)
		}
	}
	// CAPTURE 去重：同一 (lambda, target) 只一条
	seen := map[string]bool{}
	for _, c := range caps {
		key := c.Source.QualifiedName + "->" + c.Target.QualifiedName
		if seen[key] {
			t.Errorf("duplicate CAPTURE %s", key)
		}
		seen[key] = true
	}
}

// TestExtractorCppImplLink 声明-实现链接：1 头多源 / static / 只声明 / 类外定义。
func TestExtractorCppImplLink(t *testing.T) {
	paths := ResolveTree(t, "extractor/impl_link")
	ec := RunPipeline(t, "x/cpp/testdata", paths, core.LevelRaw)

	// 1 头 ↔ 多源：split_a 与 split_b 分散在两个源文件
	a := FindRel(t, ec.Relations, func(r *model.DependencyRelation) bool {
		return r.Type == model.ImplLink && r.Source.QualifiedName == "impl::split_a(int)"
	})
	if a.Source.Path != "extractor/impl_link/impl_a.cpp" ||
		a.Target.Path != "extractor/impl_link/impl.h" {
		t.Errorf("split_a ImplLink paths = %s -> %s", a.Source.Path, a.Target.Path)
	}
	b := FindRel(t, ec.Relations, func(r *model.DependencyRelation) bool {
		return r.Type == model.ImplLink && r.Source.QualifiedName == "impl::split_b(int)"
	})
	if b.Source.Path != "extractor/impl_link/impl_b.cpp" ||
		b.Target.Path != "extractor/impl_link/impl.h" {
		t.Errorf("split_b ImplLink paths = %s -> %s", b.Source.Path, b.Target.Path)
	}

	// 正常配对 + 签名
	paired := FindRel(t, ec.Relations, func(r *model.DependencyRelation) bool {
		return r.Type == model.ImplLink && r.Source.QualifiedName == "impl::paired(int)"
	})
	if sig, _ := paired.Mores[constants.RelImplLinkSignature].(string); sig != "impl::paired(int)(int)" {
		t.Errorf("paired signature = %q", sig)
	}

	// 类外定义配对
	FindRel(t, ec.Relations, func(r *model.DependencyRelation) bool {
		return r.Type == model.ImplLink && r.Source.QualifiedName == "impl::Service::member(int)"
	})

	// static 函数不配对
	if HasRel(ec.Relations, func(r *model.DependencyRelation) bool {
		return r.Type == model.ImplLink && r.Source.QualifiedName == "impl::file_local(int)"
	}) {
		t.Errorf("static function should not be ImplLink-paired")
	}
	// 只声明不定义不建链接
	if HasRel(ec.Relations, func(r *model.DependencyRelation) bool {
		return r.Type == model.ImplLink && r.Source.QualifiedName == "impl::declared_only(int)"
	}) {
		t.Errorf("declaration-only function should not be ImplLink-paired")
	}
	// 类内 inline 定义不配对
	if HasRel(ec.Relations, func(r *model.DependencyRelation) bool {
		return r.Type == model.ImplLink && r.Source.QualifiedName == "impl::Service::inline_member(int)"
	}) {
		t.Errorf("inline member should not be ImplLink-paired")
	}
}
