// Package test 解析层（第 2 层）单元测试：验证 Java SymbolResolver 的三段式动作解析。
//
// 场景化组织结构（R1-R10）：
//
//	R1 基础调用    R2 外部符号    R3 链式调用
//	R4 泛型传播    R5 重载匹配    R6 继承查找
//	R7 Use 解析    R8 Create/Cast/Throw   R9 合拍性校正   R10 可见性
//
// 策略：对每个动作点调用 ResolveAction，断言返回元素的精确 QN / Kind / IsFormExternal。
// 断言值均来自 debug_dump 实测（与 extractor 注释期望、e2e 基线一致）。
package test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// walkNodes 深度优先遍历 AST，对每个节点调用 visit。
func walkNodes(n coreast.Node, visit func(coreast.Node)) {
	if n == nil {
		return
	}
	visit(n)
	for i := 0; i < n.NamedChildCount(); i++ {
		walkNodes(n.NamedChild(i), visit)
	}
}

// nodesByKind 收集 AST 中指定 kind 的全部节点（按出现顺序）。
func nodesByKind(t *testing.T, src coreast.SourceFile, kind string) []coreast.Node {
	t.Helper()
	var out []coreast.Node
	walkNodes(src.Root(), func(n coreast.Node) {
		if n.Kind() == kind {
			out = append(out, n)
		}
	})
	return out
}

// findNodeByText 在 AST 中查找首个完整文本等于 want 的节点（不含空白）。
func findNodeByText(t *testing.T, src coreast.SourceFile, want string) coreast.Node {
	t.Helper()
	content := src.Content()
	var found coreast.Node
	walkNodes(src.Root(), func(n coreast.Node) {
		if found != nil {
			return
		}
		if strings.TrimSpace(n.Utf8Text(content)) == want {
			found = n
		}
	})
	if found == nil {
		t.Fatalf("no node with text %q found", want)
	}
	return found
}

// assertResolve 调用 ResolveAction 并断言返回元素的精确 QN / Kind / IsFormExternal。
// wantQN 传空串表示只断言非空；wantKind 传空串表示不校验 kind。
func assertResolve(t *testing.T, gc *core.GlobalContext, fc *core.FileContext, node coreast.Node, relTyp model.DependencyType, wantQN string, wantKind model.ElementKind, wantExternal bool) {
	t.Helper()
	elem := resolveAction(t, gc, fc, node, relTyp)
	if elem == nil {
		t.Errorf("ResolveAction(%s) returned nil", relTyp)
		return
	}
	if wantQN != "" && elem.QualifiedName != wantQN {
		t.Errorf("ResolveAction(%s) QN = %q, want %q (kind=%s external=%v)", relTyp, elem.QualifiedName, wantQN, elem.Kind, elem.IsFormExternal)
	}
	if wantKind != "" && elem.Kind != wantKind {
		t.Errorf("ResolveAction(%s) kind = %s, want %s (QN=%q)", relTyp, elem.Kind, wantKind, elem.QualifiedName)
	}
	if elem.IsFormExternal != wantExternal {
		t.Errorf("ResolveAction(%s) IsFormExternal = %v, want %v (QN=%q)", relTyp, elem.IsFormExternal, wantExternal, elem.QualifiedName)
	}
}

// resolveAtLine 按行号返回该行上的指定 kind 动作节点（该行首个匹配）。
// 用于精确定位存在重复文本的动作点（如多行相同调用）。
func resolveAtLine(t *testing.T, src coreast.SourceFile, kind string, line int) coreast.Node {
	t.Helper()
	var found coreast.Node
	walkNodes(src.Root(), func(n coreast.Node) {
		if found != nil || n.Kind() != kind {
			return
		}
		if n.StartPoint().Row+1 == line {
			found = n
		}
	})
	if found == nil {
		t.Fatalf("no %s node at line %d", kind, line)
	}
	return found
}

// actionKinds 动作节点 kind 与对应关系类型的映射。
var actionKinds = []struct {
	kind   string
	relTyp model.DependencyType
}{
	{"method_invocation", model.Call},
	{"method_reference", model.Call},
	{"object_creation_expression", model.Create},
	{"cast_expression", model.Cast},
	{"throw_statement", model.Throw},
	{"assignment_expression", model.Assign},
}

// TestResolverCall 验证 Call 动作的解析：所有 method_invocation 节点应解析出方法目标。
func TestResolverCall(t *testing.T) {
	paths := ResolvePaths(t, "extractor/call")
	gc, _ := RunExtraction(t, "x/java/testdata", paths)

	for _, path := range paths {
		src := parseSource(t, path)
		fc, ok := gc.FileContext(relOf(t, "x/java/testdata", path))
		if !ok {
			t.Fatalf("file context not found: %s", path)
		}

		var invocations int
		walkNodes(src.Root(), func(n coreast.Node) {
			if n.Kind() != "method_invocation" {
				return
			}
			invocations++
			elem := resolveAction(t, gc, fc, n, model.Call)
			if elem == nil {
				t.Errorf("%s:%d: ResolveAction(Call) returned nil for %q", path, n.StartPoint().Row+1, n.Utf8Text(src.Content()))
				return
			}
			if elem.QualifiedName == "" {
				t.Errorf("%s:%d: ResolveAction(Call) empty QN for %q", path, n.StartPoint().Row+1, n.Utf8Text(src.Content()))
			}
		})
		if invocations == 0 {
			t.Errorf("%s: no method_invocation nodes found", path)
		}
	}
}

// TestResolverAllActions 遍历全部动作类型，验证每个动作点都能解析出非空目标。
func TestResolverAllActions(t *testing.T) {
	paths := ResolvePaths(t, "extractor/call")
	gc, _ := RunExtraction(t, "x/java/testdata", paths)

	for _, path := range paths {
		src := parseSource(t, path)
		fc, ok := gc.FileContext(relOf(t, "x/java/testdata", path))
		if !ok {
			t.Fatalf("file context not found: %s", path)
		}

		for _, ak := range actionKinds {
			count := 0
			walkNodes(src.Root(), func(n coreast.Node) {
				if n.Kind() != ak.kind {
					return
				}
				count++
				elem := resolveAction(t, gc, fc, n, ak.relTyp)
				if elem == nil {
					t.Errorf("%s:%d: ResolveAction(%s) nil for kind=%s text=%q",
						path, n.StartPoint().Row+1, ak.relTyp, ak.kind, n.Utf8Text(src.Content()))
				}
			})
			if count == 0 {
				t.Logf("%s: no %s nodes (skipped)", path, ak.kind)
			}
		}
	}
}

// =============================================================================
// R1-R3 调用解析（extractor/call/CallRelationSuite.java）
// 断言值均来自 debug_dump 实测（与 CallRelationSuite.java 注释期望一致）。
// =============================================================================

// callSuite 解析 CallRelationSuite 并返回共享上下文（gc/fc/src 已就绪）。
type callSuite struct {
	gc  *core.GlobalContext
	fc  *core.FileContext
	src coreast.SourceFile
}

func newCallSuite(t *testing.T) *callSuite {
	t.Helper()
	paths := ResolvePaths(t, "extractor/call")
	gc, _ := RunExtraction(t, "x/java/testdata", paths)
	path := paths[0]
	src := parseSource(t, path)
	fc, ok := gc.FileContext(relOf(t, "x/java/testdata", path))
	if !ok {
		t.Fatalf("file context not found: %s", path)
	}
	return &callSuite{gc: gc, fc: fc, src: src}
}

// resolveLine 按行号定位 method_invocation 节点并断言解析结果。
func (s *callSuite) resolveLine(t *testing.T, line int, wantQN string, wantKind model.ElementKind, wantExternal bool) {
	t.Helper()
	node := resolveAtLine(t, s.src, "method_invocation", line)
	assertResolve(t, s.gc, s.fc, node, model.Call, wantQN, wantKind, wantExternal)
}

// TestResolverR1BasicCall 基础调用：隐式 this / 显式 this / 类名静态 / super / 继承 / 枚举。
func TestResolverR1BasicCall(t *testing.T) {
	s := newCallSuite(t)

	// 1. 隐式 this 调用（L21）-> 本类方法
	s.resolveLine(t, 21, "com.example.rel.CallRelationSuite.simpleMethod()", model.Method, false)

	// 2. 类名静态调用（L31）-> 本类静态方法
	s.resolveLine(t, 31, "com.example.rel.CallRelationSuite.staticMethod()", model.Method, false)

	// 3. 显式 super 调用（L60）-> 父类方法（super 接收者已上溯到父类，内部解析成功）
	s.resolveLine(t, 60, "com.example.rel.BaseClass.baseMethod()", model.Method, false)

	// 4. 隐式继承调用（L69）-> 父类方法（内部解析成功，继承查找生效）
	s.resolveLine(t, 69, "com.example.rel.BaseClass.baseMethod()", model.Method, false)

	// 5. 枚举静态方法调用（L148）-> 枚举隐式 values（内部符号，desugar 生成的 values() 定义）
	s.resolveLine(t, 148, "com.example.rel.CallRelationSuite.MyEnum.values()", model.Method, false)

	// 6. Lambda 内调用（L89）-> 隐式 this 上溯 lambda 外层类解析为本类方法
	s.resolveLine(t, 89, "com.example.rel.CallRelationSuite.simpleMethod()", model.Method, false)

	// 7. 变长参数调用（L127）-> 解析为本类 customLog(String, Object...)
	s.resolveLine(t, 127, "com.example.rel.CallRelationSuite.customLog(String,Object...)", model.Method, false)

	// 8. 方法引用目标（L100 this::simpleMethod）-> 提取层捕获 identifier，解析为本类方法
	// tree-sitter 中 method_reference 无 name 字段，方法名是第 2 个具名子节点（identifier）。
	mrNode := resolveAtLine(t, s.src, "method_reference", 100)
	nameNode := mrNode.NamedChild(1)
	if nameNode == nil || nameNode.Kind() != "identifier" {
		t.Fatalf("L100 method_reference second named child is not identifier: %v", nameNode)
	}
	assertResolve(t, s.gc, s.fc, nameNode, model.Call, "com.example.rel.CallRelationSuite.simpleMethod()", model.Method, false)

	// 9. 显式构造器调用 super()（L163）-> 解析为父类构造器 BaseClass.BaseClass()（内部符号）
	s.resolveLineKind(t, 163, "explicit_constructor_invocation", model.Call, "com.example.rel.BaseClass.BaseClass()", model.Method, false)
}

// TestResolverR2External 外部符号：未 import / JDK 类型 / 泛型显式调用 / 匿名类内调用 / cast 后调用。
func TestResolverR2External(t *testing.T) {
	s := newCallSuite(t)

	// 1. JDK 静态调用（L41）-> 外部符号（未 import，保持原样）
	s.resolveLine(t, 41, "System.currentTimeMillis", model.Method, true)

	// 2. 链式调用外层 add（L50）-> 外部 List.add
	s.resolveLine(t, 50, "com.example.rel.CallRelationSuite.getList().add", model.Method, true)

	// 3. 泛型方法显式调用（L109）-> this.<String>genericMethod 显式接收者已识别，解析为本类泛型方法
	s.resolveLine(t, 109, "com.example.rel.CallRelationSuite.genericMethod(T)", model.Method, false)

	// 4. 匿名类内调用（L121）-> 已上溯到外部类，解析为本类方法
	s.resolveLine(t, 121, "com.example.rel.CallRelationSuite.simpleMethod()", model.Method, false)

	// 5. cast 后调用（L134）-> cast 接收者已解析（((CallRelationSuite)obj)），解析为本类方法
	s.resolveLine(t, 134, "com.example.rel.CallRelationSuite.simpleMethod()", model.Method, false)
}

// TestResolverR3Chained 链式调用：内层方法解析成功，外层断链回退。
func TestResolverR3Chained(t *testing.T) {
	s := newCallSuite(t)

	// 1. 链式内层 getList()（L50）-> 本类方法（findNodeByText 精确匹配内层节点）
	inner := findNodeByText(t, s.src, "getList()")
	assertResolve(t, s.gc, s.fc, inner, model.Call, "com.example.rel.CallRelationSuite.getList()", model.Method, false)

	// 2. 链式外层 add（L50）-> 外部 List.add（断链回退）
	outer := findNodeByText(t, s.src, `getList().add("item")`)
	assertResolve(t, s.gc, s.fc, outer, model.Call, "com.example.rel.CallRelationSuite.getList().add", model.Method, true)

	// 3. Builder 模式链式（case1 L41 new Builder().name().age().build()）-> 每段均解析为内部方法
	paths := ResolvePaths(t, "extractor/call/case1")
	gc, _ := RunExtraction(t, "x/java/testdata", paths)
	src := parseSource(t, paths[0])
	fc := mustFileContext(t, gc, paths[0])

	// 4. 普通链式（case1 L49 obj1.getName().toUpperCase()）-> 内层 getName 内部，外层 toUpperCase 断链外部
	getNameCall := findNodeByText(t, src, "obj1.getName()")
	assertResolve(t, gc, fc, getNameCall, model.Call, "com.example.chained.ChainedCallExample.getName()", model.Method, false)

	// Builder 链式各段：method_invocation 文本含换行，用 name identifier 定位后上溯到调用节点
	for _, want := range []struct {
		line   int
		name   string
		wantQN string
	}{
		{42, "name", "com.example.chained.ChainedCallExample.Builder.name(String)"},
		{43, "age", "com.example.chained.ChainedCallExample.Builder.age(int)"},
	} {
		nameNode := findIdentAtLine(t, src, want.line, want.name)
		callNode := nameNode.Parent()
		if callNode == nil || callNode.Kind() != "method_invocation" {
			t.Fatalf("L%d %s parent not method_invocation", want.line, want.name)
		}
		assertResolve(t, gc, fc, callNode, model.Call, want.wantQN, model.Method, false)
	}
}

// =============================================================================
// R4-R6 重载 / 泛型 / 继承（x/java/testdata/resolver 与 extractor/use/case2）
// 断言值均来自 debug_dump 实测。
// =============================================================================

// resolverSuite 解析 resolver 目录下的测试套件并返回共享上下文。
func resolverSuite(t *testing.T, dirs ...string) *callSuite {
	t.Helper()
	paths := ResolvePaths(t, dirs...)
	gc, _ := RunExtraction(t, "x/java/testdata", paths)
	return &callSuite{gc: gc, fc: mustFileContext(t, gc, paths[0]), src: parseSource(t, paths[0])}
}

// mustFileContext 按路径取 FileContext，找不到则 fatal。
func mustFileContext(t *testing.T, gc *core.GlobalContext, path string) *core.FileContext {
	t.Helper()
	fc, ok := gc.FileContext(relOf(t, "x/java/testdata", path))
	if !ok {
		t.Fatalf("file context not found: %s", path)
	}
	return fc
}

// resolveLineKind 按行号定位指定 kind 的动作节点并断言解析结果。
func (s *callSuite) resolveLineKind(t *testing.T, line int, kind string, relTyp model.DependencyType, wantQN string, wantKind model.ElementKind, wantExternal bool) {
	t.Helper()
	node := resolveAtLine(t, s.src, kind, line)
	assertResolve(t, s.gc, s.fc, node, relTyp, wantQN, wantKind, wantExternal)
}

// TestResolverR4Generic 泛型元素类型传播：方法返回 List<Order>，数组下标消费传播元素类型。
func TestResolverR4Generic(t *testing.T) {
	s := resolverSuite(t, "resolver/generic")

	// 1. L19 类名静态调用作为 segment（GenericPropagationSuite.getOrders()[0]）
	//    -> 传播 List<Order> 元素类型，array_access 消费得 Order 类型。
	s.resolveLineKind(t, 19, "array_access", model.Use, "com.example.resolver.GenericPropagationSuite.Order", model.Class, false)

	// 2. L22 隐式调用作为 head（getOrders()[0]）：head 不经 resolveSegment，不触发传播
	//    -> 返回方法本身。
	s.resolveLineKind(t, 22, "array_access", model.Use, "com.example.resolver.GenericPropagationSuite.getOrders()", model.Method, false)

	// 3. L19/L22 方法调用本身（getOrders）-> 均解析为内部方法
	s.resolveLine(t, 19, "com.example.resolver.GenericPropagationSuite.getOrders()", model.Method, false)
	s.resolveLine(t, 22, "com.example.resolver.GenericPropagationSuite.getOrders()", model.Method, false)
}

// TestResolverR5Overload 重载打分：参数数量 -> 类型精确 -> 变长参数。
func TestResolverR5Overload(t *testing.T) {
	s := resolverSuite(t, "resolver/overload")

	// L15 handle("text") -> String 精确匹配 handle(String)
	s.resolveLine(t, 15, "com.example.resolver.OverloadSuite.handle(String)", model.Method, false)
	// L16 handle(42) -> int 精确匹配 handle(int)
	s.resolveLine(t, 16, "com.example.resolver.OverloadSuite.handle(int)", model.Method, false)
	// L17 handle("a", List.of()) -> (String,List) 匹配 handle(String,List)
	s.resolveLine(t, 17, "com.example.resolver.OverloadSuite.handle(String,List)", model.Method, false)
	// L18 handle(1,2,3) -> 变长参数匹配 handle(Object...)
	s.resolveLine(t, 18, "com.example.resolver.OverloadSuite.handle(Object...)", model.Method, false)
}

// TestResolverR6Inheritance 继承查找：字段遮蔽 / 继承静态字段 / 继承方法。
func TestResolverR6Inheritance(t *testing.T) {
	// use/case2：Child extends Parent，验证字段遮蔽与继承静态字段。
	paths := ResolvePaths(t, "extractor/use/case2")
	gc, _ := RunExtraction(t, "x/java/testdata", paths)

	childPath := findPathByBase(t, paths, "Child.java")
	src := parseSource(t, childPath)
	fc := mustFileContext(t, gc, childPath)

	// 1. L9 count -> 本类遮蔽字段（count=20）
	node := findIdentAtLine(t, src, 9, "count")
	assertResolve(t, gc, fc, node, model.Use, "com.example.rel.use.case2.Child.count", model.Field, false)

	// 2. L12 TAG -> 跨类继承解析为 Parent 静态字段
	node = findIdentAtLine(t, src, 12, "TAG")
	assertResolve(t, gc, fc, node, model.Use, "com.example.rel.use.case2.Parent.TAG", model.Field, false)

	// 3. CallRelationSuite L69 隐式继承调用 -> BaseClass 方法（内部解析成功）
	s := newCallSuite(t)
	s.resolveLine(t, 69, "com.example.rel.BaseClass.baseMethod()", model.Method, false)
}

// findIdentAtLine 在指定行查找文本匹配 want 的 identifier 节点。
func findIdentAtLine(t *testing.T, src coreast.SourceFile, line int, want string) coreast.Node {
	t.Helper()
	content := src.Content()
	var found coreast.Node
	walkNodes(src.Root(), func(n coreast.Node) {
		if found != nil || n.Kind() != "identifier" {
			return
		}
		if n.StartPoint().Row+1 == line && n.Utf8Text(content) == want {
			found = n
		}
	})
	if found == nil {
		t.Fatalf("no identifier %q at line %d", want, line)
	}
	return found
}

// findPathByBase 在路径列表中查找 basename 匹配的文件。
func findPathByBase(t *testing.T, paths []string, base string) string {
	t.Helper()
	for _, p := range paths {
		if filepath.Base(p) == base {
			return p
		}
	}
	t.Fatalf("no path with base %q in %v", base, paths)
	return ""
}

// findTypeIdentAtLine 在指定行查找文本匹配 want 的 type_identifier 节点。
func findTypeIdentAtLine(t *testing.T, src coreast.SourceFile, line int, want string) coreast.Node {
	t.Helper()
	content := src.Content()
	var found coreast.Node
	walkNodes(src.Root(), func(n coreast.Node) {
		if found != nil || n.Kind() != "type_identifier" {
			return
		}
		if n.StartPoint().Row+1 == line && n.Utf8Text(content) == want {
			found = n
		}
	})
	if found == nil {
		t.Fatalf("no type_identifier %q at line %d", want, line)
	}
	return found
}

// =============================================================================
// R7-R10 关系类型 / 合拍性 / 可见性（Use/Create/Cast/Throw suite）
// 断言值均来自 debug_dump 实测。
// =============================================================================

// TestResolverR7Use Use 解析：局部变量 / 字段 / 参数 / 静态常量 / 数组 / lambda 捕获。
func TestResolverR7Use(t *testing.T) {
	s := resolverSuite(t, "extractor/use")

	// 1. 局部变量读取（L16 local）
	node := findIdentAtLine(t, s.src, 16, "local")
	assertResolve(t, s.gc, s.fc, node, model.Use, "com.example.rel.UseRelationSuite.testUseCases(int).local", model.Variable, false)

	// 2. 显式 this 成员变量（L22 fieldVar）
	node = findIdentAtLine(t, s.src, 22, "fieldVar")
	assertResolve(t, s.gc, s.fc, node, model.Use, "com.example.rel.UseRelationSuite.fieldVar", model.Field, false)

	// 3. 参数读取（L29 param）
	node = findIdentAtLine(t, s.src, 29, "param")
	assertResolve(t, s.gc, s.fc, node, model.Use, "com.example.rel.UseRelationSuite.testUseCases(int).param", model.Variable, false)

	// 4. 静态常量访问（L35 CONSTANT）
	node = findIdentAtLine(t, s.src, 35, "CONSTANT")
	assertResolve(t, s.gc, s.fc, node, model.Use, "com.example.rel.UseRelationSuite.CONSTANT", model.Field, false)

	// 5. 数组引用读取（L42 arr）
	node = findIdentAtLine(t, s.src, 42, "arr")
	assertResolve(t, s.gc, s.fc, node, model.Use, "com.example.rel.UseRelationSuite.testUseCases(int).arr", model.Variable, false)

	// 6. 增强 for 循环集合（L62 list）与元素（L66 item）
	node = findIdentAtLine(t, s.src, 62, "list")
	assertResolve(t, s.gc, s.fc, node, model.Use, "com.example.rel.UseRelationSuite.testUseCases(int).list", model.Variable, false)
	node = findIdentAtLine(t, s.src, 66, "item")
	assertResolve(t, s.gc, s.fc, node, model.Use, "com.example.rel.UseRelationSuite.testUseCases(int).block$1.item", model.Variable, false)

	// 7. lambda 捕获外部字段（L74 fieldVar）
	node = findIdentAtLine(t, s.src, 74, "fieldVar")
	assertResolve(t, s.gc, s.fc, node, model.Use, "com.example.rel.UseRelationSuite.fieldVar", model.Field, false)

	// 8. 隐式 this 成员变量（L29 fieldVar，与参数 param 同行）
	node = findIdentAtLine(t, s.src, 29, "fieldVar")
	assertResolve(t, s.gc, s.fc, node, model.Use, "com.example.rel.UseRelationSuite.fieldVar", model.Field, false)

	// 9. 方法参数传递（L49 s / x，argument_list 上下文）
	node = findIdentAtLine(t, s.src, 49, "s")
	assertResolve(t, s.gc, s.fc, node, model.Use, "com.example.rel.UseRelationSuite.testUseCases(int).s", model.Variable, false)
	node = findIdentAtLine(t, s.src, 49, "x")
	assertResolve(t, s.gc, s.fc, node, model.Use, "com.example.rel.UseRelationSuite.testUseCases(int).x", model.Variable, false)

	// 10. 括号表达式读取（L55 x，parenthesized_expression 上下文）
	node = findIdentAtLine(t, s.src, 55, "x")
	assertResolve(t, s.gc, s.fc, node, model.Use, "com.example.rel.UseRelationSuite.testUseCases(int).x", model.Variable, false)

	// 11. 强转操作数读取（L82 obj，cast_expression 上下文）
	node = findIdentAtLine(t, s.src, 82, "obj")
	assertResolve(t, s.gc, s.fc, node, model.Use, "com.example.rel.UseRelationSuite.testUseCases(int).obj", model.Variable, false)
}

// TestResolverR8CreateCastThrow Create/Cast/Throw 类型解析。
func TestResolverR8CreateCastThrow(t *testing.T) {
	// Create 套件。
	cs := resolverSuite(t, "extractor/create")
	// L14 字段初始化 -> ArrayList（外部，import 补全 QN）
	cs.resolveLineKind(t, 14, "object_creation_expression", model.Create, "java.util.ArrayList", model.Class, true)
	// L20 静态字段 -> HashMap（外部）
	cs.resolveLineKind(t, 20, "object_creation_expression", model.Create, "java.util.HashMap", model.Class, true)
	// L27 局部变量 -> StringBuilder（外部，未 import）
	cs.resolveLineKind(t, 27, "object_creation_expression", model.Create, "StringBuilder", model.Class, true)
	// L33 匿名类 -> Runnable（外部）
	cs.resolveLineKind(t, 33, "object_creation_expression", model.Create, "Runnable", model.Class, true)
	// L50 本类实例化 -> 内部定义
	cs.resolveLineKind(t, 50, "object_creation_expression", model.Create, "com.example.rel.CreateRelationSuite", model.Class, false)

	// Cast 套件。
	xs := resolverSuite(t, "extractor/cast")
	// L13 向下转型 -> String（外部）
	xs.resolveLineKind(t, 13, "cast_expression", model.Cast, "String", model.Class, true)
	// L20 基本类型转型 -> int
	xs.resolveLineKind(t, 20, "cast_expression", model.Cast, "int", model.Class, true)
	// L26 泛型集合转型 -> java.util.List（外部，import 补全）
	xs.resolveLineKind(t, 26, "cast_expression", model.Cast, "java.util.List", model.Class, true)
	// L32 嵌套类转型 -> 内部定义
	xs.resolveLineKind(t, 32, "cast_expression", model.Cast, "com.example.rel.CastRelationSuite.SubClass", model.Class, false)
	// L47 多重转型外层 -> Runnable（外部）
	xs.resolveLineKind(t, 47, "cast_expression", model.Cast, "Runnable", model.Class, true)
	// L47 多重转型内层 (Object)input -> Object（外部）
	for _, cn := range nodesByKind(t, xs.src, "cast_expression") {
		if cn.StartPoint().Row+1 == 47 && cn.Utf8Text(xs.src.Content()) == "(Object)input" {
			assertResolve(t, xs.gc, xs.fc, cn, model.Cast, "Object", model.Class, true)
		}
	}
	// L38 模式匹配 instanceof -> String（外部）
	xs.resolveLineKind(t, 38, "instanceof_expression", model.Cast, "String", model.Class, true)

	// Throw 套件。
	ts := resolverSuite(t, "extractor/throw")
	// L22 主动抛异常 -> RuntimeException（外部，未 import）
	ts.resolveLineKind(t, 22, "throw_statement", model.Throw, "RuntimeException", model.Class, true)
	// L22 new RuntimeException(...) 的 object_creation_expression 按 Create 解析 -> RuntimeException（外部）
	ts.resolveLineKind(t, 22, "object_creation_expression", model.Create, "RuntimeException", model.Class, true)
	// L38 重抛捕获异常 e -> 声明类型 Exception（catch_formal_parameter 类型提取）
	ts.resolveLineKind(t, 38, "throw_statement", model.Throw, "Exception", model.Class, true)
	// L50 重抛多异常联合类型捕获的 e -> 取第一个类型 IOException
	ts.resolveLineKind(t, 50, "throw_statement", model.Throw, "IOException", model.Class, true)
	// L16 throws IOException / SQLException 签名（type_identifier 按 Throw 解析）
	ioNode := findTypeIdentAtLine(t, ts.src, 16, "IOException")
	assertResolve(t, ts.gc, ts.fc, ioNode, model.Throw, "IOException", model.Class, true)
	sqlNode := findTypeIdentAtLine(t, ts.src, 16, "SQLException")
	assertResolve(t, ts.gc, ts.fc, sqlNode, model.Throw, "java.sql.SQLException", model.Class, true)
	// L29 构造签名 throws Exception
	excNode := findTypeIdentAtLine(t, ts.src, 29, "Exception")
	assertResolve(t, ts.gc, ts.fc, excNode, model.Throw, "Exception", model.Class, true)
}

// TestResolverR9Alignment 合拍性校正：同一节点按不同关系类型解析得到不同目标。
func TestResolverR9Alignment(t *testing.T) {
	s := resolverSuite(t, "extractor/create")

	// L14 new ArrayList<>()：Create 期望类型 -> Class；
	// 同一节点按 Call 解析 -> 合拍性校正为外部 Method。
	node := resolveAtLine(t, s.src, "object_creation_expression", 14)
	assertResolve(t, s.gc, s.fc, node, model.Create, "java.util.ArrayList", model.Class, true)
	assertResolve(t, s.gc, s.fc, node, model.Call, "java.util.ArrayList", model.Method, true)

	// 同节点按 Use/Cast/Throw 解析：alignElementWithRelType 不误改（非期望分支保持原 Class）。
	assertResolve(t, s.gc, s.fc, node, model.Use, "java.util.ArrayList", model.Class, true)
	assertResolve(t, s.gc, s.fc, node, model.Cast, "java.util.ArrayList", model.Class, true)
	assertResolve(t, s.gc, s.fc, node, model.Throw, "java.util.ArrayList", model.Class, true)

	// L27 new StringBuilder()（未 import）按 Call -> 校正为外部 Method。
	node27 := resolveAtLine(t, s.src, "object_creation_expression", 27)
	assertResolve(t, s.gc, s.fc, node27, model.Call, "StringBuilder", model.Method, true)

	// L50 本类实例化按 Call -> 校正为内部 Method。
	node50 := resolveAtLine(t, s.src, "object_creation_expression", 50)
	assertResolve(t, s.gc, s.fc, node50, model.Call, "com.example.rel.CreateRelationSuite", model.Method, false)
}

// TestResolverR10Visibility 可见性：protected 跨包子类可见，default 跨包不可见。
func TestResolverR10Visibility(t *testing.T) {
	paths := ResolvePaths(t, "extractor/use/case3")
	gc, _ := RunExtraction(t, "x/java/testdata", paths)

	subPath := findPathByBase(t, paths, "Sub.java")
	src := parseSource(t, subPath)
	fc := mustFileContext(t, gc, subPath)

	// 1. L9 protectedVar：protected 跨包子类可见 -> 解析为父类字段（内部定义）
	node := findIdentAtLine(t, src, 9, "protectedVar")
	assertResolve(t, gc, fc, node, model.Use, "com.example.rel.use.case3.pk1.Base.protectedVar", model.Field, false)

	// 2. L10 packageVar：default 跨包不可见 -> 外部兜底
	node = findIdentAtLine(t, src, 10, "packageVar")
	assertResolve(t, gc, fc, node, model.Use, "packageVar", model.Class, true)
}

// resolveAction 调用 SymbolResolver.ResolveAction 并返回目标元素。
func resolveAction(t *testing.T, gc *core.GlobalContext, fc *core.FileContext, node coreast.Node, relTyp model.DependencyType) *model.CodeElement {
	t.Helper()
	resolver := gc.Resolver
	return resolver.ResolveAction(gc, fc, node, relTyp)
}

// parseSource 解析源文件为 SourceFile。
func parseSource(t *testing.T, path string) coreast.SourceFile {
	t.Helper()
	impl := JavaImpl(t)
	p := impl.Parser.NewParser()
	defer p.Close()
	src, err := p.ParseFile(path)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return src
}

// relOf 计算 path 相对 root 的路径。
func relOf(t *testing.T, root, path string) string {
	t.Helper()
	rel, err := filepathRel(root, path)
	if err != nil {
		t.Fatalf("rel %s to %s: %v", path, root, err)
	}
	return rel
}

// filepathRel 包装 filepath.Rel（避免测试文件直接引 filepath 造成混淆）。
func filepathRel(root, path string) (string, error) {
	absRoot := absRoot(root)
	rel, err := filepath.Rel(absRoot, path)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}
