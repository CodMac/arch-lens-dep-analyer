package test

import (
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/constants"
)

// TestResolverCall 覆盖调用解析：自由函数/成员方法/静态方法/命名空间限定调用。
func TestResolverCall(t *testing.T) {
	paths := ResolvePaths(t, "resolver", "*.cpp")
	_, rels := RunExtraction(t, "x/cpp/testdata", paths)

	// 自由函数调用
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "caller()" &&
			r.Target.QualifiedName == "free_func(int)"
	})

	// 成员方法调用（obj.method()）
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "caller()" &&
			r.Target.QualifiedName == "Widget::draw()"
	})

	// 指针成员调用（ptr->method()）
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "caller()" &&
			r.Target.QualifiedName == "Widget::resize(int)"
	})

	// 静态方法调用（Class::staticMethod()）
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "caller()" &&
			r.Target.QualifiedName == "Widget::create()"
	})

	// 命名空间限定调用（ns::func()）
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "caller()" &&
			r.Target.QualifiedName == "util::helper()"
	})
}

// TestResolverCreate 覆盖 C++ new 表达式解析。
func TestResolverCreate(t *testing.T) {
	paths := ResolvePaths(t, "resolver", "*.cpp")
	_, rels := RunExtraction(t, "x/cpp/testdata", paths)

	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Create && r.Source.QualifiedName == "maker()" &&
			r.Target.QualifiedName == "Widget"
	})
	// new Widget() 同时产构造器 CALL。
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "maker()" &&
			r.Target.QualifiedName == "Widget::Widget"
	})
}

// TestResolverUse 覆盖变量/字段/枚举常量使用解析。
func TestResolverUse(t *testing.T) {
	paths := ResolvePaths(t, "resolver", "*.cpp")
	_, rels := RunExtraction(t, "x/cpp/testdata", paths)

	// 局部变量读取（作为实参）
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Use && r.Source.QualifiedName == "caller()" &&
			r.Target.QualifiedName == "caller()::local"
	})

	// 成员字段读取（w.count）
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Use && r.Source.QualifiedName == "caller()" &&
			r.Target.QualifiedName == "Widget::count"
	})

	// 枚举常量（Color::RED）
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Use && r.Source.QualifiedName == "caller()" &&
			r.Target.QualifiedName == "Color::RED"
	})
}

// TestResolverCast 覆盖 C/C++ 强转解析（C 风格与 static_cast）。
func TestResolverCast(t *testing.T) {
	paths := ResolvePaths(t, "resolver", "*.cpp")
	_, rels := RunExtraction(t, "x/cpp/testdata", paths)

	casts := RelFilter(rels, model.Cast)
	if len(casts) < 2 {
		t.Fatalf("CAST relations = %d, want >= 2", len(casts))
	}
	for _, c := range casts {
		if c.Source.QualifiedName != "caster(void*)" || c.Target.QualifiedName != "Widget" {
			t.Errorf("unexpected CAST: %s -> %s", c.Source.QualifiedName, c.Target.QualifiedName)
		}
	}
}

// TestResolverFieldInitializer 覆盖构造器成员初始化列表（: a(x)）的赋值解析。
func TestResolverFieldInitializer(t *testing.T) {
	paths := ResolvePaths(t, "resolver", "*.cpp")
	_, rels := RunExtraction(t, "x/cpp/testdata", paths)

	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Assign && r.Source.QualifiedName == "Pair2::Pair2(int,int)" &&
			r.Target.QualifiedName == "Pair2::a"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Assign && r.Source.QualifiedName == "Pair2::Pair2(int,int)" &&
			r.Target.QualifiedName == "Pair2::b"
	})
}

// TestResolverCapture 覆盖 Lambda 对外部局部变量的捕获（CAPTURE）。
func TestResolverCapture(t *testing.T) {
	paths := ResolvePaths(t, "resolver", "*.cpp")
	_, rels := RunExtraction(t, "x/cpp/testdata", paths)

	cap := FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Capture
	})
	if cap.Source.Kind != model.Lambda {
		t.Errorf("capture source kind = %s, want LAMBDA", cap.Source.Kind)
	}
	if cap.Target.QualifiedName != "capture_user(int)::base" {
		t.Errorf("capture target = %s, want capture_user(int)::base", cap.Target.QualifiedName)
	}
}

// TestResolverAssign 覆盖赋值目标解析（局部变量/成员字段）。
func TestResolverAssign(t *testing.T) {
	paths := ResolvePaths(t, "resolver", "*.cpp")
	_, rels := RunExtraction(t, "x/cpp/testdata", paths)

	// 局部变量初始化（int read = ...）
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Assign && r.Source.QualifiedName == "caller()" &&
			r.Target.QualifiedName == "caller()::read"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Assign && r.Source.QualifiedName == "caller()" &&
			r.Target.QualifiedName == "Widget::count"
	})
}

// TestResolverAnonymousTypedefField 覆盖匿名 struct 承载 typedef 的字段解析：
// typedef struct {...} Point; 的变量 p 其 type_with_qn 必须是 Point（而非匿名结构体文本），
// 否则 p.x 会退化为全局短名扫描，命中同名参数 x（并行收集顺序不定 → 结果不稳定）。
func TestResolverAnonymousTypedefField(t *testing.T) {
	paths := ResolvePaths(t, "collector/base")
	gc, rels := RunExtraction(t, "x/cpp/testdata", paths)

	// Binder 必须把 p 的类型绑定到 Point 本身，而不是匿名结构体文本。
	if entry, ok := gc.FindByQualifiedName("process()::p"); !ok {
		t.Fatal("process()::p not found")
	} else if got, _ := entry.Element.GetMores()[constants.VariableTypeWithQN].(string); got != "Point" {
		t.Errorf("process()::p type_with_qn = %q, want Point", got)
	}

	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Assign && r.Source.QualifiedName == "process()" &&
			r.Target.QualifiedName == "Point::x"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Use && r.Source.QualifiedName == "process()" &&
			r.Target.QualifiedName == "Point::x"
	})
}

// TestResolverChain 覆盖链式调用的接收者类型传播：
// f()->g() 取 f 的返回类型、a.b.c 取字段声明类型、this->f()->g() 组合。
func TestResolverChain(t *testing.T) {
	paths := ResolvePaths(t, "resolver", "chain.cpp")
	_, rels := RunExtraction(t, "x/cpp/testdata", paths)

	// 字段链 a.b.c：接收者类型逐级传播
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "chain_use(Holder*,Switcher*)" &&
			r.Target.QualifiedName == "L3::deep()"
	})

	// 调用链 f()->g()：取 f 的返回类型作接收者
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "chain_use(Holder*,Switcher*)" &&
			r.Target.QualifiedName == "L2::getL3()"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "chain_use(Holder*,Switcher*)" &&
			r.Target.QualifiedName == "L1::getL2()"
	})

	// 自由函数返回值作接收者：makeL1(h)->getL2()
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "chain_use(Holder*,Switcher*)" &&
			r.Target.QualifiedName == "makeL1(Holder*)"
	})

	// 同名方法区分：Alpha::pick 与 Beta::pick 各按接收者类型命中
	alphaCalls, betaCalls := 0, 0
	for _, r := range rels {
		if r.Type != model.Call || r.Source.QualifiedName != "chain_use(Holder*,Switcher*)" {
			continue
		}
		switch r.Target.QualifiedName {
		case "Alpha::pick()":
			alphaCalls++
		case "Beta::pick()":
			betaCalls++
		}
	}
	if alphaCalls != 2 {
		t.Errorf("Alpha::pick() calls = %d, want 2 (s->getAlpha()->pick / s->a->pick)", alphaCalls)
	}
	if betaCalls != 2 {
		t.Errorf("Beta::pick() calls = %d, want 2 (s->getBeta()->pick / s->b->pick)", betaCalls)
	}

	// this->f()->g()：隐式接收者 + 链式
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "Owner::run()" &&
			r.Target.QualifiedName == "Owner::self()"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "Owner::run()" &&
			r.Target.QualifiedName == "Alpha::pick()"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source.QualifiedName == "Owner::run()" &&
			r.Target.QualifiedName == "Beta::pick()"
	})
}
