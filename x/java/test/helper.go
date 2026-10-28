// Package test 提供 Java 层三层测试（Collector/Resolver/Extractor）与 e2e 共用的测试辅助：
// 统一负责"解析 -> Collect -> Bind -> Extract"的流水线搭建，以及元素/关系的打印与检索。
package test

import (
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/engine"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	_ "github.com/CodMac/arch-lens-dep-analyer/x/java" // 注册 Java 实现
)

// TestDataDir 指向黄金回归样例目录（仓库根下的相对路径）。
// 测试运行时 CWD 为各包目录，因此通过 helper.go 自身位置上溯仓库根，再拼相对路径。
var TestDataDir = ""

// repoRoot 定位仓库根（helper.go 位于 <root>/x/java/test/helper.go）。
func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	// helper.go -> test -> java -> x -> root，共 4 级上溯。
	return filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))
}

func init() {
	TestDataDir = filepath.Join(repoRoot(), "x", "java", "testdata")
}

// RepoRoot 返回仓库根目录的绝对路径。
func RepoRoot() string {
	return repoRoot()
}

// absRoot 将测试传入的 root（相对仓库根或绝对）规范为绝对路径。
func absRoot(root string) string {
	if filepath.IsAbs(root) {
		return root
	}
	return filepath.Join(repoRoot(), root)
}

// ParserFactory 返回可创建 Java parser 的工厂闭包（供 engine 调度器使用）。
func ParserFactory(impl *core.LanguageImpl) func() (core.Parser, error) {
	return func() (core.Parser, error) {
		return impl.Parser.NewParser(), nil
	}
}

// JavaImpl 获取已注册的 Java 语言实现。
func JavaImpl(t *testing.T) *core.LanguageImpl {
	t.Helper()
	impl, err := core.Get(core.LangJava)
	if err != nil {
		t.Fatalf("get java impl: %v", err)
	}
	return impl
}

// ResolvePaths 将 testdata 子路径拼为仓库根下的文件路径列表。
// 例：ResolvePaths(t, "collector/base") 返回该目录内全部 .java 文件。
func ResolvePaths(t *testing.T, dirs ...string) []string {
	t.Helper()
	var files []string
	for _, d := range dirs {
		matches, err := filepath.Glob(filepath.Join(TestDataDir, d, "*.java"))
		if err != nil {
			t.Fatalf("glob %s: %v", d, err)
		}
		files = append(files, matches...)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatalf("no testdata files matched for %v", dirs)
	}
	return files
}

// RunPipeline 跑完整五阶段流水线（Collect+Bind+Extract+Linker+NoiseFilter），
// 返回带最终 Relations 的 EngineContext（等价 CLI 行为）。
func RunPipeline(t *testing.T, root string, paths []string, level core.FilterLevel) *engine.EngineContext {
	t.Helper()
	impl := JavaImpl(t)
	ec, err := engine.RunDefault(engine.Config{
		Language:    core.LangJava,
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
func runCollect(t *testing.T, impl *core.LanguageImpl, gc *core.GlobalContext, root string, paths []string) {
	t.Helper()
	sched := engine.NewScheduler(4, ParserFactory(impl))
	absRoot := absRoot(root)
	err := sched.Run(paths, func(path string, p core.Parser) error {
		src, err := p.ParseFile(path)
		if err != nil {
			return err
		}
		fc, err := impl.Collector.CollectDefinitions(src)
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}
		fc.FilePath = filepath.ToSlash(relPath)
		gc.RegisterFileContext(fc)
		return nil
	})
	if err != nil {
		t.Fatalf("collect stage: %v", err)
	}
}

// RunExtraction 跑"解析 -> Collect -> Bind -> Extract"四步，
// 返回可断言的 GlobalContext 与动作/结构关系（不含 Linker 的 Contain 与 NoiseFilter 过滤），
// 供三层测试精确断言关系提取结果。
func RunExtraction(t *testing.T, root string, paths []string) (*core.GlobalContext, []*model.DependencyRelation) {
	t.Helper()
	impl := JavaImpl(t)

	ec, err := engine.NewContext(engine.Config{
		Language:    core.LangJava,
		RootPath:    root,
		FilePaths:   paths,
		Concurrency: 4,
		FilterLevel: core.LevelRaw,
	})
	if err != nil {
		t.Fatalf("create context: %v", err)
	}

	// 阶段 1: Collect（并行）
	runCollect(t, impl, ec.GC, root, paths)

	// 阶段 2: Bind（串行）
	if err := impl.Binder.BindSymbols(ec.GC); err != nil {
		t.Fatalf("bind stage: %v", err)
	}

	// 阶段 3: Extract（并行）
	sched := engine.NewScheduler(4, ParserFactory(impl))
	absRoot := absRoot(root)
	var mu sync.Mutex
	err = sched.Run(paths, func(path string, _ core.Parser) error {
		relPath, err := filepath.Rel(absRoot, path)
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

// RunCollection 只跑"解析 -> Collect"两步，返回含全部定义（未 Bind）的 GlobalContext，
// 供 Collector 层测试直接断言元素定义。
func RunCollection(t *testing.T, root string, paths []string) *core.GlobalContext {
	t.Helper()
	impl := JavaImpl(t)
	gc := core.NewGlobalContext(impl.SymbolResolver, impl.SymbolBuilder)
	runCollect(t, impl, gc, root, paths)
	return gc
}

// FindDef 按 QN 在全局上下文中查找定义，找不到则 fatal。
func FindDef(t *testing.T, gc *core.GlobalContext, qn string) *model.CodeElement {
	t.Helper()
	entry, ok := gc.FindByQualifiedName(qn)
	if !ok {
		t.Fatalf("definition not found: %s", qn)
	}
	return entry.Element
}

// DefExists 判断某 QN 的定义是否存在。
func DefExists(gc *core.GlobalContext, qn string) bool {
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

// PrintCodeElements 打印全部定义（调试用，-v 时可见）。
func PrintCodeElements(t *testing.T, gc *core.GlobalContext) {
	t.Helper()
	for _, def := range gc.Definitions() {
		el := def.Element
		t.Logf("  [%s] %s (parent=%s)", el.Kind, el.QualifiedName, def.ParentQN)
	}
}

// PrintRelations 打印全部关系（调试用，-v 时可见）。
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

// =============================================================================
// 断言辅助（Collector 层场景化测试）
// =============================================================================

// AssertDef 断言某 QN 的定义存在且 kind 匹配（wantKind 传空串则只校验存在）。
func AssertDef(t *testing.T, gc *core.GlobalContext, qn string, wantKind model.ElementKind) *model.CodeElement {
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

// AssertNoDef 断言某 QN 的定义不存在（守护"明确不生成"的项）。
func AssertNoDef(t *testing.T, gc *core.GlobalContext, qn string) {
	t.Helper()
	if _, ok := gc.FindByQualifiedName(qn); ok {
		t.Errorf("definition should NOT exist: %s", qn)
	}
}

// AssertMores 断言某 QN 元素的 Mores 键值（want 传 nil 表示仅断言键存在）。
func AssertMores(t *testing.T, gc *core.GlobalContext, qn, key string, want interface{}) {
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
	if want != nil {
		if !moresEqual(got, want) {
			t.Errorf("%s Mores[%s] = %v (%T), want %v (%T)", qn, key, got, got, want, want)
		}
	}
}

// moresEqual 比较 Mores 值（支持 []string / bool / int / string 等可比较类型）。
func moresEqual(got, want interface{}) bool {
	switch w := want.(type) {
	case []string:
		g, ok := got.([]string)
		if !ok {
			return false
		}
		if len(g) != len(w) {
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

// AssertSig 断言某 QN 元素的 Signature 包含子串。
func AssertSig(t *testing.T, gc *core.GlobalContext, qn, wantSubstr string) {
	t.Helper()
	elem := AssertDef(t, gc, qn, "")
	if elem == nil {
		return
	}
	if !strings.Contains(elem.Signature, wantSubstr) {
		t.Errorf("%s Signature = %q, want contains %q", qn, elem.Signature, wantSubstr)
	}
}

// AssertMods 断言某 QN 元素的 Extra.Modifiers 包含全部指定修饰符（顺序无关）。
// 用于直接校验修饰符列表（如包私有空列表 / protected / public static 等）。
func AssertMods(t *testing.T, gc *core.GlobalContext, qn string, want ...string) {
	t.Helper()
	elem := AssertDef(t, gc, qn, "")
	if elem == nil {
		return
	}
	if elem.Extra == nil {
		t.Errorf("%s Extra is nil", qn)
		return
	}
	got := elem.Extra.Modifiers
	if len(want) == 0 {
		// 空期望：断言无修饰符。
		if len(got) != 0 {
			t.Errorf("%s Modifiers = %v, want none", qn, got)
		}
		return
	}
	missing := make([]string, 0)
	for _, w := range want {
		if !slices.Contains(got, w) {
			missing = append(missing, w)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%s Modifiers = %v, missing %v", qn, got, missing)
	}
}

// AssertAnnos 断言某 QN 元素的 Extra.Annotations 包含全部指定注解文本（顺序无关）。
// 注解文本为原始形态（如 "@Override" / "@Loggable"）。
func AssertAnnos(t *testing.T, gc *core.GlobalContext, qn string, want ...string) {
	t.Helper()
	elem := AssertDef(t, gc, qn, "")
	if elem == nil {
		return
	}
	if elem.Extra == nil {
		t.Errorf("%s Extra is nil", qn)
		return
	}
	got := elem.Extra.Annotations
	if len(want) == 0 {
		// 空期望：断言无注解。
		if len(got) != 0 {
			t.Errorf("%s Annotations = %v, want none", qn, got)
		}
		return
	}
	missing := make([]string, 0)
	for _, w := range want {
		if !slices.Contains(got, w) {
			missing = append(missing, w)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%s Annotations = %v, missing %v", qn, got, missing)
	}
}

// ChildrenOf 返回某父 QN 下的全部子定义（用于枚举容器成员）。
func ChildrenOf(t *testing.T, gc *core.GlobalContext, parentQN string) []*model.CodeElement {
	t.Helper()
	var out []*model.CodeElement
	for _, def := range gc.Definitions() {
		if def.ParentQN == parentQN {
			out = append(out, def.Element)
		}
	}
	return out
}

// ChildQNs 返回某父 QN 下的全部子定义 QN 集合（断言用）。
func ChildQNs(t *testing.T, gc *core.GlobalContext, parentQN string) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, def := range gc.Definitions() {
		if def.ParentQN == parentQN {
			out[def.Element.QualifiedName] = true
		}
	}
	return out
}
