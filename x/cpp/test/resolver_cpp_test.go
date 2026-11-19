package test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// =============================================================================
// Resolver 测试辅助：直接对动作节点调用 ResolveAction，断言精确 QN/Kind/外部性。
// =============================================================================

// cppWalk 深度优先遍历 AST。
func cppWalk(n coreast.Node, visit func(coreast.Node)) {
	if n == nil {
		return
	}
	visit(n)
	for i := 0; i < n.NamedChildCount(); i++ {
		cppWalk(n.NamedChild(i), visit)
	}
}

// cppParse 返回与该文件 FileContext 共享同一语法树的根节点与源码。
// 必须复用 fc 的树：元素定义记录的 Node 来自该树，节点身份判断依赖同一棵树。
func cppParse(t *testing.T, fc core.FileContext) (coreast.Node, []byte) {
	t.Helper()
	return fc.Root(), *fc.SourceBytes()
}

// cppFileContext 取源文件对应的 FileContext（相对 testdata 根）。
func cppFileContext(t *testing.T, gc core.GlobalContext, path string) core.FileContext {
	t.Helper()
	rel, err := filepath.Rel(absRoot("x/cpp/testdata"), path)
	if err != nil {
		t.Fatalf("rel %s: %v", path, err)
	}
	fc, ok := gc.FileContext(filepath.ToSlash(rel))
	if !ok {
		t.Fatalf("file context not found: %s", path)
	}
	return fc
}

// cppFindNode 查找首个完整文本匹配的节点。
func cppFindNode(t *testing.T, root coreast.Node, content []byte, want string) coreast.Node {
	t.Helper()
	return cppFindNodeOccurrence(t, root, content, want, 1)
}

// cppFindNodeOccurrence 查找第 occurrence 个完整文本匹配的节点（1 起始）。
func cppFindNodeOccurrence(t *testing.T, root coreast.Node, content []byte, want string, occurrence int) coreast.Node {
	t.Helper()
	var found coreast.Node
	idx := 0
	cppWalk(root, func(n coreast.Node) {
		if found != nil {
			return
		}
		if strings.TrimSpace(n.Utf8Text(content)) == want {
			idx++
			if idx == occurrence {
				found = n
			}
		}
	})
	if found == nil {
		t.Fatalf("no node with text %q (occurrence %d)", want, occurrence)
	}
	return found
}

// cppFindNodeKind 查找首个「kind + 完整文本」匹配的节点。
func cppFindNodeKind(t *testing.T, root coreast.Node, content []byte, kind, want string) coreast.Node {
	t.Helper()
	var found coreast.Node
	cppWalk(root, func(n coreast.Node) {
		if found == nil && n.Kind() == kind && strings.TrimSpace(n.Utf8Text(content)) == want {
			found = n
		}
	})
	if found == nil {
		t.Fatalf("no %s node with text %q", kind, want)
	}
	return found
}

// cppFindCall 查找首个完整文本匹配的 call_expression 节点。
func cppFindCall(t *testing.T, root coreast.Node, content []byte, want string) coreast.Node {
	t.Helper()
	var found coreast.Node
	cppWalk(root, func(n coreast.Node) {
		if found == nil && n.Kind() == "call_expression" &&
			strings.TrimSpace(n.Utf8Text(content)) == want {
			found = n
		}
	})
	if found == nil {
		t.Fatalf("no call_expression with text %q", want)
	}
	return found
}

// cppAssertResolve 调用 ResolveAction 并断言 QN / Kind / 外部性。
func cppAssertResolve(t *testing.T, gc core.GlobalContext, fc core.FileContext, node coreast.Node,
	relTyp model.DependencyType, wantQN string, wantKind model.ElementKind, wantExternal bool) {
	t.Helper()
	elem := gc.Resolver().ResolveAction(gc, fc, node, relTyp)
	if elem == nil {
		t.Errorf("ResolveAction(%s) returned nil (want %q)", relTyp, wantQN)
		return
	}
	if elem.QualifiedName != wantQN {
		t.Errorf("ResolveAction(%s) QN = %q, want %q (kind=%s ext=%v)",
			relTyp, elem.QualifiedName, wantQN, elem.Kind, elem.IsFormExternal)
	}
	if wantKind != "" && elem.Kind != wantKind {
		t.Errorf("ResolveAction(%s) kind = %s, want %s (QN=%q)", relTyp, elem.Kind, wantKind, elem.QualifiedName)
	}
	if elem.IsFormExternal != wantExternal {
		t.Errorf("ResolveAction(%s) external = %v, want %v (QN=%q)", relTyp, elem.IsFormExternal, wantExternal, elem.QualifiedName)
	}
}

// cppResolveGroup 跑一组样例并返回 (gc, 主源文件路径)。
func cppResolveGroup(t *testing.T, group, mainFile string) (core.GlobalContext, string) {
	t.Helper()
	paths := ResolveTree(t, group)
	gc, _ := RunExtraction(t, "x/cpp/testdata", paths)
	for _, p := range paths {
		if filepath.Base(p) == mainFile {
			return gc, p
		}
	}
	t.Fatalf("%s not found in %s", mainFile, group)
	return nil, ""
}

// TestResolverCppForms 覆盖段 1 的 6 种 exprForm 归一化与段 2 分派。
func TestResolverCppForms(t *testing.T) {
	gc, path := cppResolveGroup(t, "resolver/cpp/forms", "forms.cpp")
	fc := cppFileContext(t, gc, path)
	root, content := cppParse(t, fc)

	// formIdent：裸标识符变量
	cppAssertResolve(t, gc, fc, cppFindNode(t, root, content, "local"), model.Use,
		"frm::forms_user(TypeX*)::local", model.Variable, false)

	// formMember：指针成员字段 / 成员方法
	cppAssertResolve(t, gc, fc, cppFindNode(t, root, content, "p->field"), model.Use,
		"frm::TypeX::field", model.Field, false)
	cppAssertResolve(t, gc, fc, cppFindCall(t, root, content, "p->method()"), model.Call,
		"frm::TypeX::method()", model.Method, false)

	// formScoped：命名空间限定 / 静态成员
	cppAssertResolve(t, gc, fc, cppFindCall(t, root, content, "sub::sub_func()"), model.Call,
		"frm::sub::sub_func()", model.Function, false)
	cppAssertResolve(t, gc, fc, cppFindCall(t, root, content, "TypeX::static_method()"), model.Call,
		"frm::TypeX::static_method()", model.Method, false)

	// formCall：自由函数
	cppAssertResolve(t, gc, fc, cppFindCall(t, root, content, "free_func(a)"), model.Call,
		"frm::free_func(int)", model.Function, false)

	// formNew：new 表达式 → 类型
	cppAssertResolve(t, gc, fc, cppFindNode(t, root, content, "new TypeX()"), model.Create,
		"frm::TypeX", model.Struct, false)

	// formCast：C 风格 / static_cast → 类型
	cppAssertResolve(t, gc, fc, cppFindNode(t, root, content, "(TypeX *)p"), model.Cast,
		"frm::TypeX", model.Struct, false)
	cppAssertResolve(t, gc, fc, cppFindNode(t, root, content, "static_cast<TypeX *>(p)"), model.Cast,
		"frm::TypeX", model.Struct, false)
}

// TestResolverCppOverload 覆盖按实参个数挑选重载（自由函数 + 成员方法）。
func TestResolverCppOverload(t *testing.T) {
	gc, path := cppResolveGroup(t, "resolver/cpp/overload", "overload.cpp")
	fc := cppFileContext(t, gc, path)
	root, content := cppParse(t, fc)

	cases := []struct{ call, want string }{
		{"pick(1)", "ov::pick(int)"},
		{"pick(1, 2)", "ov::pick(int,int)"},
		{"pick()", "ov::pick()"},
		{"s->call(1)", "ov::Svc::call(int)"},
		{"s->call(1, 2)", "ov::Svc::call(int,int)"},
		{"s->call()", "ov::Svc::call()"},
	}
	for _, c := range cases {
		cppAssertResolve(t, gc, fc, cppFindCall(t, root, content, c.call), model.Call, c.want, "", false)
	}
}

// TestResolverCppInherit 覆盖继承链成员查找：一级/二级基类、多继承、覆盖。
func TestResolverCppInherit(t *testing.T) {
	gc, path := cppResolveGroup(t, "resolver/cpp/inherit", "inherit.cpp")
	fc := cppFileContext(t, gc, path)
	root, content := cppParse(t, fc)

	cases := []struct {
		expr     string
		relTyp   model.DependencyType
		wantQN   string
		wantKind model.ElementKind
	}{
		// 直接成员
		{"leaf->leaf_field", model.Use, "inh::Leaf::leaf_field", model.Field},
		{"leaf->leaf_method()", model.Call, "inh::Leaf::leaf_method()", model.Method},
		// 一级基类
		{"leaf->mid_a_field", model.Use, "inh::MidA::mid_a_field", model.Field},
		{"leaf->mid_a_method()", model.Call, "inh::MidA::mid_a_method()", model.Method},
		{"leaf->mid_b_field", model.Use, "inh::MidB::mid_b_field", model.Field},
		{"leaf->mid_b_method()", model.Call, "inh::MidB::mid_b_method()", model.Method},
		// 二级基类（菱形）
		{"leaf->root_field", model.Use, "inh::Root::root_field", model.Field},
		{"leaf->root_method()", model.Call, "inh::Root::root_method()", model.Method},
		// 覆盖：命中派生类
		{"ov->root_method()", model.Call, "inh::Overrider::root_method()", model.Method},
	}
	for _, c := range cases {
		cppAssertResolve(t, gc, fc, cppFindNode(t, root, content, c.expr), c.relTyp, c.wantQN, c.wantKind, false)
	}
}

// TestResolverCppScope 覆盖作用域链：相对限定名补全、嵌套类、块内遮蔽。
func TestResolverCppScope(t *testing.T) {
	gc, path := cppResolveGroup(t, "resolver/cpp/scope", "scope.cpp")
	fc := cppFileContext(t, gc, path)
	root, content := cppParse(t, fc)

	// 相对限定名：outer 内的 inner::scoped_func 补全为 outer::inner::scoped_func
	cppAssertResolve(t, gc, fc, cppFindNode(t, root, content, "inner::scoped_func()"), model.Call,
		"outer::inner::scoped_func()", model.Function, false)
	// 短名上溯：outer_func 在 outer 命名空间
	cppAssertResolve(t, gc, fc, cppFindCall(t, root, content, "outer_func()"), model.Call,
		"outer::outer_func()", model.Function, false)
	// 嵌套类成员
	cppAssertResolve(t, gc, fc, cppFindNode(t, root, content, "n->nested_field"), model.Use,
		"outer::OuterType::NestedType::nested_field", model.Field, false)
	cppAssertResolve(t, gc, fc, cppFindNode(t, root, content, "n->nested_method()"), model.Call,
		"outer::OuterType::NestedType::nested_method()", model.Method, false)

	// 块作用域遮蔽：块内读取 v（第 3 个 v 标识符）解析为 block$1::v。
	cppAssertResolve(t, gc, fc, cppFindNodeOccurrence(t, root, content, "v", 3), model.Use,
		"outer::shadowing(int)::block$1::v", model.Variable, false)
	// 块外读取 v（第 4 个 v）解析为函数参数 v。
	cppAssertResolve(t, gc, fc, cppFindNodeOccurrence(t, root, content, "v", 4), model.Use,
		"outer::shadowing(int)::v", model.Variable, false)
}

// TestResolverCppExternal 覆盖外部符号兜底：外部类型/方法、模板变量不产依赖。
func TestResolverCppExternal(t *testing.T) {
	gc, path := cppResolveGroup(t, "resolver/cpp/external", "external.cpp")
	fc := cppFileContext(t, gc, path)
	root, content := cppParse(t, fc)

	// 外部类型变量调用：s.size() → std::string::size（外部方法，按接收者类型归属）
	cppAssertResolve(t, gc, fc, cppFindCall(t, root, content, "s.size()"), model.Call,
		"std::string::size", model.Method, true)
	// 外部类型容器：vec.push_back / vec.size
	cppAssertResolve(t, gc, fc, cppFindCall(t, root, content, "vec.push_back(1)"), model.Call,
		"std::vector::push_back", model.Method, true)
	cppAssertResolve(t, gc, fc, cppFindCall(t, root, content, "vec.size()"), model.Call,
		"std::vector::size", model.Method, true)
	// 源码内自由函数仍正常解析
	cppAssertResolve(t, gc, fc, cppFindCall(t, root, content, "make_string()"), model.Call,
		"ext::make_string()", model.Function, false)

	// ResolveType：基本类型 → PRIMITIVE
	prim := gc.Resolver().ResolveType(gc, fc, "int", model.Class)
	if prim == nil || prim.Kind != model.Primitive {
		t.Errorf("ResolveType(int) = %v, want PRIMITIVE", prim)
	}
	// 模板类型变量 T → nil（不是真实依赖）
	if got := gc.Resolver().ResolveType(gc, fc, "T", model.Class); got != nil {
		t.Errorf("ResolveType(T) = %v, want nil", got)
	}
	// 外部类型 → 外部元素，QN 保留限定名
	ext := gc.Resolver().ResolveType(gc, fc, "std::string", model.Class)
	if ext == nil || !ext.IsFormExternal || ext.QualifiedName != "std::string" {
		t.Errorf("ResolveType(std::string) = %v, want external std::string", ext)
	}
	if ext != nil && ext.NamespaceQN != "std" {
		t.Errorf("ResolveType(std::string) NamespaceQN = %q, want std", ext.NamespaceQN)
	}
}

// TestResolverCppLambdaVar 覆盖 lambda 变量调用与立即调用 lambda。
func TestResolverCppLambdaVar(t *testing.T) {
	gc, path := cppResolveGroup(t, "resolver/cpp/lambda_var", "lambda_var.cpp")
	fc := cppFileContext(t, gc, path)
	root, content := cppParse(t, fc)

	// auto f = [...](){...}; f(2) → Lambda 元素
	cppAssertResolve(t, gc, fc, cppFindCall(t, root, content, "f(2)"), model.Call,
		"lam::lambda_caller(int)::lambda$1", model.Lambda, false)
	// 立即调用 lambda：[](int v){...}(3) → 该 Lambda 元素
	cppAssertResolve(t, gc, fc, cppFindCall(t, root, content, "[](int v) { return v * 2; }(3)"), model.Call,
		"lam::lambda_caller(int)::lambda$2", model.Lambda, false)
}

// TestResolverCppCast 覆盖 C 风格与四种显式转换关键字。
func TestResolverCppCast(t *testing.T) {
	gc, path := cppResolveGroup(t, "resolver/cpp/cast", "cast.cpp")
	fc := cppFileContext(t, gc, path)
	root, content := cppParse(t, fc)

	cases := []struct{ expr, want string }{
		{"(CastA *)raw", "cst::CastA"},
		{"static_cast<CastA *>(raw)", "cst::CastA"},
		{"dynamic_cast<CastPolyD *>(poly)", "cst::CastPolyD"},
		{"const_cast<const CastA *>(c_style)", "cst::CastA"},
		{"reinterpret_cast<CastB *>(raw)", "cst::CastB"},
	}
	for _, c := range cases {
		cppAssertResolve(t, gc, fc, cppFindNode(t, root, content, c.expr), model.Cast, c.want, "", false)
	}
	// (void)x 抑制：ResolveAction(Cast) 返回 void，由 extractor 过滤
	voidCast := gc.Resolver().ResolveAction(gc, fc, cppFindNode(t, root, content, "(void)c_style"), model.Cast)
	if voidCast == nil || voidCast.QualifiedName != "void" {
		t.Errorf("(void)c_style cast = %v, want void", voidCast)
	}
}

// TestResolverCppThrow 覆盖 throw new T / throw T(...) / throw var。
func TestResolverCppThrow(t *testing.T) {
	gc, path := cppResolveGroup(t, "resolver/cpp/throw", "throw.cpp")
	fc := cppFileContext(t, gc, path)
	root, content := cppParse(t, fc)

	cases := []struct{ expr, kind, want string }{
		{"new ErrA()", "new_expression", "thr::ErrA"},
		{"ErrB()", "call_expression", "thr::ErrB"},
		{"e", "identifier", "thr::ErrA"},
	}
	for _, c := range cases {
		cppAssertResolve(t, gc, fc, cppFindNodeKind(t, root, content, c.kind, c.expr), model.Throw, c.want, "", false)
	}
}

// TestResolverCall 覆盖调用解析：自由函数/成员方法/静态方法/命名空间限定调用。
func TestResolverCall(t *testing.T) {
	paths := ResolvePaths(t, "resolver/cpp", "*.cpp")
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
	paths := ResolvePaths(t, "resolver/cpp", "*.cpp")
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
	paths := ResolvePaths(t, "resolver/cpp", "*.cpp")
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
	paths := ResolvePaths(t, "resolver/cpp", "*.cpp")
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
	paths := ResolvePaths(t, "resolver/cpp", "*.cpp")
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
	paths := ResolvePaths(t, "resolver/cpp", "*.cpp")
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
	paths := ResolvePaths(t, "resolver/cpp", "*.cpp")
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

// TestResolverChain 覆盖链式调用的接收者类型传播：
// f()->g() 取 f 的返回类型、a.b.c 取字段声明类型、this->f()->g() 组合。
func TestResolverChain(t *testing.T) {
	paths := ResolvePaths(t, "resolver/cpp", "chain.cpp")
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
