package test

import (
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/constants"
)

// TestExtractorStructural 覆盖结构关系：Import / Contain / Extend / Parameter / Return / TypeArg。
func TestExtractorStructural(t *testing.T) {
	paths := ResolvePaths(t, "collector/base", "*.c", "*.h", "*.cpp")
	gc, rels := RunExtraction(t, "x/cpp/testdata", paths)

	// --- Import：项目内头文件解析为 File 元素 ---
	imp := FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Import && r.Target != nil &&
			r.Target.QualifiedName == "collector/base/basic.h"
	})
	if imp.Source.QualifiedName != "collector/base/basic.c" {
		t.Errorf("import source = %s, want collector/base/basic.c", imp.Source.QualifiedName)
	}

	// --- Import：系统头归外部符号 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Import && r.Target != nil &&
			r.Target.QualifiedName == "stdio.h" && r.Target.IsFormExternal
	})

	// --- Contain：类 → 字段 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Contain &&
			r.Source.QualifiedName == "Node" && r.Target.QualifiedName == "Node::value"
	})

	// --- Extend：C++ 继承（相对命名空间解析为绝对 QN） ---
	ext := FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Extend && r.Source.QualifiedName == "app::Box"
	})
	if ext.Target.QualifiedName != "app::core::Base" {
		t.Errorf("Box extend target = %s, want app::core::Base", ext.Target.QualifiedName)
	}

	// --- Parameter / Return ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Parameter && r.Source.QualifiedName == "add(int,int)" &&
			r.Target.QualifiedName == "int"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Return && r.Source.QualifiedName == "add(int,int)" &&
			r.Target.QualifiedName == "int"
	})

	// --- TypeArg：字段类型 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.TypeArg && r.Source.QualifiedName == "Node::next" &&
			r.Target.QualifiedName == "Node"
	})
	// --- TypeArg：局部变量类型 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.TypeArg && r.Source.QualifiedName == "process()::p" &&
			r.Target.QualifiedName == "Point"
	})

	_ = gc
}

// TestExtractorActions 覆盖动作关系：Call / Assign / Use / Cast。
func TestExtractorActions(t *testing.T) {
	paths := ResolvePaths(t, "collector/base", "*.c", "*.h", "*.cpp")
	_, rels := RunExtraction(t, "x/cpp/testdata", paths)

	// --- Call：函数调用解析到定义 ---
	call := FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "process()" &&
			r.Target.Name == "add"
	})
	if call.Target.QualifiedName != "add(int,int)" {
		t.Errorf("call target = %s, want add(int,int)", call.Target.QualifiedName)
	}

	// --- Call：static 函数 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "process()" &&
			r.Target.QualifiedName == "internal_helper(int)"
	})

	// --- Assign：字段赋值（p.x = ...） ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Assign && r.Source.QualifiedName == "process()" &&
			r.Target.QualifiedName == "Point::x"
	})

	// --- Assign：变量初始化（int local = ...） ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Assign && r.Source.QualifiedName == "add(int,int)" &&
			r.Target.QualifiedName == "add(int,int)::local"
	})

	// --- Use：局部变量读取 ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Use && r.Source.QualifiedName == "add(int,int)" &&
			r.Target.QualifiedName == "add(int,int)::local"
	})

	// --- Use：成员字段读取（m_val） ---
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Use && r.Source.QualifiedName == "app::core::Base::helper()" &&
			r.Target.QualifiedName == "app::core::Base::m_val"
	})

	// --- Cast：真实强转（非 (void)x 抑制形态） ---
	if HasRel(rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Cast && r.Target != nil && r.Target.QualifiedName == "void"
	}) {
		t.Errorf("(void)x should not produce CAST relation")
	}

}

// TestExtractorImplLink 覆盖声明-实现链接（Linker 产出）：跨文件签名配对。
func TestExtractorImplLink(t *testing.T) {
	paths := ResolvePaths(t, "collector/base", "*.c", "*.h", "*.cpp")
	ec := RunPipeline(t, "x/cpp/testdata", paths, core.LevelRaw)

	// add: basic.h 声明 + basic.c 定义 → ImplLink（定义 → 声明）
	link := FindRel(t, ec.Relations, func(r *model.DependencyRelation) bool {
		return r.Type == model.ImplLink && r.Source.QualifiedName == "add(int,int)"
	})
	if link.Source.Path != "collector/base/basic.c" {
		t.Errorf("ImplLink source path = %s, want collector/base/basic.c", link.Source.Path)
	}
	if link.Target.Path != "collector/base/basic.h" {
		t.Errorf("ImplLink target path = %s, want collector/base/basic.h", link.Target.Path)
	}
	if sig, _ := link.Mores[constants.RelImplLinkSignature].(string); sig != "add(int,int)(int,int)" {
		t.Errorf("ImplLink signature = %q", sig)
	}

	// static 函数不参与配对
	if HasRel(ec.Relations, func(r *model.DependencyRelation) bool {
		return r.Type == model.ImplLink && r.Source.QualifiedName == "internal_helper(int)"
	}) {
		t.Errorf("static function should not be ImplLink-paired")
	}

	// 类外定义：Box::twice 成员声明 + 类外定义 → ImplLink
	FindRel(t, ec.Relations, func(r *model.DependencyRelation) bool {
		return r.Type == model.ImplLink && r.Source.QualifiedName == "app::Box::twice(int)"
	})
}

// TestExtractorContainNamespace 覆盖 namespace 层级拓扑。
func TestExtractorContainNamespace(t *testing.T) {
	paths := ResolvePaths(t, "collector/base", "*.cpp", "*.h")
	ec := RunPipeline(t, "x/cpp/testdata", paths, core.LevelRaw)

	// Namespace → SubNamespace
	FindRel(t, ec.Relations, func(r *model.DependencyRelation) bool {
		return r.Type == model.Contain && r.Source.Kind == model.Namespace &&
			r.Source.QualifiedName == "app" && r.Target.QualifiedName == "app::core"
	})
	// Namespace → File
	FindRel(t, ec.Relations, func(r *model.DependencyRelation) bool {
		return r.Type == model.Contain && r.Source.Kind == model.Namespace &&
			r.Source.QualifiedName == "app" && r.Target.Kind == model.File
	})
	// File → 顶层元素
	FindRel(t, ec.Relations, func(r *model.DependencyRelation) bool {
		return r.Type == model.Contain && r.Source.Kind == model.File &&
			r.Target.QualifiedName == "app::Box"
	})
}
