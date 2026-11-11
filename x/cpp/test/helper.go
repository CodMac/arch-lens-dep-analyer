// Package test 提供 C/C++ 层三层测试（Collector/Resolver/Extractor）与 e2e 共用的测试辅助：
// 统一负责"解析 -> Collect -> Bind -> Extract"的流水线搭建，以及元素/关系的打印与检索。
package test

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/engine"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	_ "github.com/CodMac/arch-lens-dep-analyer/x/cpp" // 注册 C/C++ 实现
)

// TestDataDir 指向测试样例目录。
var TestDataDir = ""

func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	// helper.go -> test -> cpp -> x -> root，共 4 级上溯。
	return filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))
}

func init() {
	TestDataDir = filepath.Join(repoRoot(), "x", "cpp", "testdata")
}

// RepoRoot 返回仓库根目录的绝对路径。
func RepoRoot() string { return repoRoot() }

// absRoot 将测试传入的 root（相对仓库根或绝对）规范为绝对路径。
func absRoot(root string) string {
	if filepath.IsAbs(root) {
		return root
	}
	return filepath.Join(repoRoot(), root)
}

// ParserFactory 返回可创建 parser 的工厂闭包（供 engine 调度器使用）。
func ParserFactory(impl *core.LanguageImpl) func() (core.Parser, error) {
	return func() (core.Parser, error) {
		return impl.Parser.NewParser(), nil
	}
}

// CppImpl 获取已注册的 C++ 语言实现。
func CppImpl(t *testing.T) *core.LanguageImpl {
	t.Helper()
	impl, err := core.Get(core.LangCPP)
	if err != nil {
		t.Fatalf("get cpp impl: %v", err)
	}
	return impl
}

// ResolvePaths 将 testdata 子目录下的指定扩展名文件拼为绝对路径列表。
func ResolvePaths(t *testing.T, dir string, exts ...string) []string {
	t.Helper()
	if len(exts) == 0 {
		exts = []string{"*.c", "*.h", "*.cpp", "*.hpp", "*.cc", "*.cxx"}
	}
	var files []string
	for _, ext := range exts {
		matches, err := filepath.Glob(filepath.Join(TestDataDir, dir, ext))
		if err != nil {
			t.Fatalf("glob %s: %v", ext, err)
		}
		files = append(files, matches...)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatalf("no testdata files matched in %s", dir)
	}
	return files
}

// ResolveTree 递归收集 testdata 子目录下指定扩展名的源文件。
// 用于包含嵌套子目录的样例组（如 extractor/structural/nested/）。
func ResolveTree(t *testing.T, dir string, exts ...string) []string {
	t.Helper()
	if len(exts) == 0 {
		exts = []string{".c", ".h", ".cpp", ".hpp", ".cc", ".cxx"}
	}
	want := make(map[string]bool, len(exts))
	for _, e := range exts {
		want[e] = true
	}
	var files []string
	err := filepath.Walk(filepath.Join(TestDataDir, dir), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !want[filepath.Ext(path)] {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatalf("no testdata files matched in %s", dir)
	}
	return files
}

// RunPipeline 跑完整五阶段流水线。
func RunPipeline(t *testing.T, root string, paths []string, level core.FilterLevel) *engine.EngineContext {
	t.Helper()
	impl := CppImpl(t)
	ec, err := engine.RunDefault(engine.Config{
		Language:    core.LangCPP,
		RootPath:    absRoot(root),
		FilePaths:   paths,
		Concurrency: 4,
		FilterLevel: level,
	}, ParserFactory(impl))
	if err != nil {
		t.Fatalf("run pipeline: %v", err)
	}
	return ec
}

// runCollect 并行执行 Collect 阶段，将文件注册进 GlobalContext。
func runCollect(t *testing.T, impl *core.LanguageImpl, gc core.GlobalContext, root string, paths []string) {
	t.Helper()
	sched := engine.NewScheduler(4, ParserFactory(impl))
	absRootPath := absRoot(root)
	err := sched.Run(paths, func(path string, p core.Parser) error {
		src, err := p.ParseFile(path)
		if err != nil {
			return err
		}
		fc, err := impl.Collector.CollectDefinitions(src)
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(absRootPath, path)
		if err != nil {
			return err
		}
		// 与 engine.normalizeFilePath 保持一致：文件上下文与 File 元素统一用相对路径。
		slashRel := filepath.ToSlash(relPath)
		fc.SetFilePath(slashRel)
		for _, def := range fc.Definitions() {
			el := def.Element
			if el == nil {
				continue
			}
			if el.Kind == model.File {
				el.QualifiedName = slashRel
			}
			el.Path = slashRel
			if el.Location != nil {
				el.Location.FilePath = slashRel
			}
		}
		gc.RegisterFileContext(fc)
		return nil
	})
	if err != nil {
		t.Fatalf("collect stage: %v", err)
	}
}

// RunExtraction 跑"解析 -> Collect -> Bind -> Extract"四步，
// 返回可断言的 GlobalContext 与关系（不含 Linker 的 Contain 与 NoiseFilter 过滤）。
func RunExtraction(t *testing.T, root string, paths []string) (core.GlobalContext, []*model.DependencyRelation) {
	t.Helper()
	impl := CppImpl(t)

	ec, err := engine.NewContext(engine.Config{
		Language:    core.LangCPP,
		RootPath:    root,
		FilePaths:   paths,
		Concurrency: 4,
		FilterLevel: core.LevelRaw,
	})
	if err != nil {
		t.Fatalf("create context: %v", err)
	}

	runCollect(t, impl, ec.GC, root, paths)

	if err := impl.Binder.BindSymbols(ec.GC); err != nil {
		t.Fatalf("bind stage: %v", err)
	}

	sched := engine.NewScheduler(4, ParserFactory(impl))
	absRootPath := absRoot(root)
	var mu sync.Mutex
	err = sched.Run(paths, func(path string, _ core.Parser) error {
		relPath, err := filepath.Rel(absRootPath, path)
		if err != nil {
			return err
		}
		rels, err := impl.Extractor.Extract(filepath.ToSlash(relPath), ec.GC)
		if err != nil {
			return err
		}
		mu.Lock()
		ec.Relations = append(ec.Relations, rels...)
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("extract stage: %v", err)
	}

	return ec.GC, ec.Relations
}

// RunCollection 只跑"解析 -> Collect"两步，返回含全部定义（未 Bind）的 GlobalContext。
func RunCollection(t *testing.T, root string, paths []string) core.GlobalContext {
	t.Helper()
	impl := CppImpl(t)
	gc := impl.NewGlobalContext(impl.SymbolResolver, impl.SymbolBuilder)
	runCollect(t, impl, gc, root, paths)
	return gc
}

// FindDef 按 QN 在全局上下文中查找定义，找不到则 fatal。
func FindDef(t *testing.T, gc core.GlobalContext, qn string) *model.CodeElement {
	t.Helper()
	entry, ok := gc.FindByQualifiedName(qn)
	if !ok {
		t.Fatalf("definition not found: %s", qn)
	}
	return entry.Element
}

// DefExists 判断某 QN 的定义是否存在。
func DefExists(gc core.GlobalContext, qn string) bool {
	_, ok := gc.FindByQualifiedName(qn)
	return ok
}

// RelFilter 按类型过滤关系列表。
func RelFilter(rels []*model.DependencyRelation, typ model.DependencyType) []*model.DependencyRelation {
	var out []*model.DependencyRelation
	for _, r := range rels {
		if r.Type == typ {
			out = append(out, r)
		}
	}
	return out
}

// FindRel 在关系中查找满足谓词的第一条，找不到则 fatal。
func FindRel(t *testing.T, rels []*model.DependencyRelation, pred func(*model.DependencyRelation) bool) *model.DependencyRelation {
	t.Helper()
	for _, r := range rels {
		if pred(r) {
			return r
		}
	}
	t.Fatalf("no relation matched predicate")
	return nil
}

// HasRel 判断是否存在满足谓词的关系。
func HasRel(rels []*model.DependencyRelation, pred func(*model.DependencyRelation) bool) bool {
	for _, r := range rels {
		if pred(r) {
			return true
		}
	}
	return false
}

// PrintCodeElements 打印全部定义（调试用）。
func PrintCodeElements(t *testing.T, gc core.GlobalContext) {
	t.Helper()
	for _, def := range gc.Definitions() {
		el := def.Element
		t.Logf("  [%s] %s (parent=%s)", el.Kind, el.QualifiedName, def.ParentQN)
	}
}

// PrintRelations 打印全部关系（调试用）。
func PrintRelations(t *testing.T, rels []*model.DependencyRelation) {
	t.Helper()
	for _, r := range rels {
		if r == nil {
			continue
		}
		src, tgt := "?", "?"
		if r.Source != nil {
			src = r.Source.QualifiedName
		}
		if r.Target != nil {
			tgt = r.Target.QualifiedName
		}
		t.Logf("  [%s] %s -> %s", r.Type, src, tgt)
	}
}

// AssertDef 断言某 QN 的定义存在且 kind 匹配（wantKind 传空串则只校验存在）。
func AssertDef(t *testing.T, gc core.GlobalContext, qn string, wantKind model.ElementKind) *model.CodeElement {
	t.Helper()
	entry, ok := gc.FindByQualifiedName(qn)
	if !ok {
		t.Errorf("definition not found: %s", qn)
		return nil
	}
	if wantKind != "" && entry.Element.Kind != wantKind {
		t.Errorf("%s kind = %s, want %s", qn, entry.Element.Kind, wantKind)
	}
	return entry.Element
}

// AssertNoDef 断言某 QN 的定义不存在。
func AssertNoDef(t *testing.T, gc core.GlobalContext, qn string) {
	t.Helper()
	if _, ok := gc.FindByQualifiedName(qn); ok {
		t.Errorf("definition should NOT exist: %s", qn)
	}
}

// AssertMores 断言某 QN 元素的 Mores 键值（want 传 nil 表示仅断言键存在）。
func AssertMores(t *testing.T, gc core.GlobalContext, qn, key string, want interface{}) {
	t.Helper()
	elem := AssertDef(t, gc, qn, "")
	if elem == nil {
		return
	}
	got, ok := elem.GetMores()[key]
	if !ok {
		t.Errorf("%s Mores[%s] missing", qn, key)
		return
	}
	if want != nil && !moresEqual(got, want) {
		t.Errorf("%s Mores[%s] = %v (%T), want %v (%T)", qn, key, got, got, want, want)
	}
}

func moresEqual(got, want interface{}) bool {
	switch w := want.(type) {
	case []string:
		g, ok := got.([]string)
		if !ok || len(g) != len(w) {
			return false
		}
		for i := range g {
			if g[i] != w[i] {
				return false
			}
		}
		return true
	case string:
		g, ok := got.(string)
		return ok && g == w
	case bool:
		g, ok := got.(bool)
		return ok && g == w
	case int:
		g, ok := got.(int)
		return ok && g == w
	}
	return got == want
}

// ChildrenOf 返回某父 QN 下的全部子定义。
func ChildrenOf(t *testing.T, gc core.GlobalContext, parentQN string) []*model.CodeElement {
	t.Helper()
	var out []*model.CodeElement
	for _, def := range gc.Definitions() {
		if def.ParentQN == parentQN {
			out = append(out, def.Element)
		}
	}
	return out
}

// AssertDisplayFields 断言元素展示层三字段。
func AssertDisplayFields(t *testing.T, gc core.GlobalContext, qn, wantNS, wantEnc, wantType string) {
	t.Helper()
	elem := FindDef(t, gc, qn)
	if elem.NamespaceQN != wantNS {
		t.Errorf("%s NamespaceQN = %q, want %q", qn, elem.NamespaceQN, wantNS)
	}
	if elem.EnclosingQN != wantEnc {
		t.Errorf("%s EnclosingQN = %q, want %q", qn, elem.EnclosingQN, wantEnc)
	}
	if elem.TypeQN != wantType {
		t.Errorf("%s TypeQN = %q, want %q", qn, elem.TypeQN, wantType)
	}
}
