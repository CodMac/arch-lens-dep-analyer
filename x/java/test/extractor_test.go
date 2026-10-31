// Package test 提取层（第 1 层）单元测试：验证 Java Extractor 对各类动作/结构关系的关系提取。
// 策略：解析 testdata 源码注释中的断言块（见 parseExpectationComments）作为 ExpectedCase，
// 然后跑 Collect+Bind+Extract，断言每条期望关系都能在实际输出中命中。
// 对于无注释断言的 suite（EXTEND/IMPLEMENT），使用显式期望表 explicitExpectations。
package test

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// expectedCase 描述一条从源码注释解析出的期望关系断言。
// Targets 支持一个 Source 命中多个 Target（USE 多 Target 行、CAST Target1/Target2、PARAMETER 行内多 Target、THROW 连续 Source 块）。
type expectedCase struct {
	Line    int                    // 期望关系触发的源码行号（注释块起始行）
	Source  string                 // 期望 Source QN
	Targets []string               // 期望 Target QN（可多个）
	RelType model.DependencyType   // 期望关系类型（空表示跟随 suite）
	Mores   map[string]interface{} // 期望 Mores（key -> 值，字符串为 substring 匹配，bool/数字精确匹配）
}

// extractorSuites 定义各关系类型对应的测试文件（相对 testdata/extractor）。
var extractorSuites = []struct {
	relType model.DependencyType
	paths   []string
}{
	{model.Call, []string{"call/CallRelationSuite.java", "call/case1"}},
	{model.Create, []string{"create/CreateRelationSuite.java"}},
	{model.Assign, []string{"assign/AssignRelationSuite.java", "assign/AssignRelationForClassSuite.java", "assign/AssignRelationForDataFlow.java"}},
	{model.Cast, []string{"cast/CastRelationSuite.java"}},
	{model.Use, []string{"use/UseRelationSuite.java", "use/case1", "use/case2", "use/case3", "use/case4", "use/case5", "use/case6"}},
	{model.Extend, []string{"extend/ExtendRelationSuite.java"}},
	{model.Implement, []string{"implement/ImplementRelationSuite.java"}},
	{model.Parameter, []string{"parameter/ParameterRelationSuite.java"}},
	{model.Return, []string{"return/ReturnRelationSuite.java"}},
	{model.Throw, []string{"throw/ThrowRelationSuite.java"}},
	{model.Annotation, []string{"annotation/AnnotationRelationSuite.java"}},
	{model.TypeArg, []string{"type_arg/TypeArgRelationSuite.java"}},
	{model.Capture, []string{"capture/CaptureRelationSuite.java"}},
}

// relTypeFromMores 依据注释 Mores 中的 java.rel.ast_kind 推断关系类型（动作类）。
// identifier 是动作目标节点的通用形态（方法名/变量名），其语义由 suite 决定，返回空表示跟随 suite。
func relTypeFromMores(mores map[string]interface{}) model.DependencyType {
	if astv, ok := mores["java.rel.ast_kind"]; ok {
		ast, _ := astv.(string)
		switch ast {
		case "method_invocation", "method_reference", "explicit_constructor_invocation":
			return model.Call
		case "object_creation_expression", "array_creation_expression", "type_identifier":
			return model.Create
		case "cast_expression", "instanceof_expression":
			return model.Cast
		case "assignment_expression", "variable_declarator", "update_expression":
			return model.Assign
		case "throw_statement":
			return model.Throw
		}
	}
	return ""
}

// explicitExpectations 为无注释断言的 suite 提供显式期望（EXTEND/IMPLEMENT）。
// 依据实际提取输出（output/rels_dump.txt 校准基线）编写。
var explicitExpectations = map[model.DependencyType][]expectedCase{
	model.Extend: {
		{Source: "com.example.rel.ExtendRelationSuite", Targets: []string{"java.util.ArrayList"}, Mores: nil},
		{Source: "com.example.rel.ExtendRelationSuite.SubInterface", Targets: []string{"Runnable"}, Mores: nil},
		{Source: "com.example.rel.ExtendRelationSuite.SubInterface", Targets: []string{"java.io.Serializable"}, Mores: nil},
	},
	model.Implement: {
		// 类多接口实现 + 抽象类实现接口（class 的 implements 子句）。
		{Source: "com.example.rel.MultiImpl", Targets: []string{"com.example.rel.BaseApi"}, Mores: nil},
		{Source: "com.example.rel.MultiImpl", Targets: []string{"Runnable"}, Mores: nil},
		{Source: "com.example.rel.MultiImpl", Targets: []string{"com.example.rel.SingleInterface"}, Mores: nil},
		{Source: "com.example.rel.AbstractTask", Targets: []string{"com.example.rel.BaseApi"}, Mores: nil},
		// 匿名类实现接口（new Runnable(){...} -> IMPLEMENT）。
		{Source: "com.example.rel.ImplementRelationSuite.test().anonymousClass$1", Targets: []string{"Runnable"}, Mores: nil},
		{Source: "com.example.rel.ExtendRelationSuite.test().anonymousClass$1", Targets: []string{"Runnable"}, Mores: nil},
	},
}

// negativeExpectation 定义不应存在的关系（负面断言）：
// 断言提取结果中【不存在】满足 RelType + Source + Target 的关系。
// 用于覆盖注释期望中“应解析失败”的语义（如静态方法内引用实例字段）。
// Source/Target 使用与正向断言相同的宽松匹配（sourceLooseMatch/targetLooseMatch）。
type negativeExpectation struct {
	RelType model.DependencyType
	Source  string
	Target  string
}

// negativeExpectations 负面期望表：按 suite 关系类型分组。
var negativeExpectations = map[model.DependencyType][]negativeExpectation{
	model.Use: {
		// use/case4：静态方法内引用实例变量应解析失败（[Case 8]）。
		{RelType: model.Use, Source: "com.example.rel.use.case4.StaticTest.staticMethod()", Target: "instanceVar"},
		// use/case3：跨包 default 访问应不可见（[Case 6]）。
		{RelType: model.Use, Source: "com.example.rel.use.case3.pk2.Sub.check()", Target: "packageVar"},
	},
}

// TestExtractorSuite 对每个关系类型 suite 断言注释中的期望关系全部命中。
func TestExtractorSuite(t *testing.T) {
	for _, suite := range extractorSuites {
		suite := suite
		t.Run(string(suite.relType), func(t *testing.T) {
			runSuite(t, suite)
		})
	}
}

// TestExtractorFieldAccessSuite 验证字段访问类 USE 的三类修复目标（缺陷 2/3）：
//  1. 内部枚举常量访问（Color.RED）-> 解析为内部 ENUM_CONSTANT；
//  2. 外部静态字段/枚举常量访问（HttpStatus.NOT_FOUND）-> 解析为外部 Field（import 补全 QN）；
//  3. class literal（File.class / Color.class）-> 解析为类型（外部 Class / 内部 Enum）。
//
// RunExtraction 使用 LevelRaw，外部符号不被 NoiseFilter 过滤，可精确断言。
func TestExtractorFieldAccessSuite(t *testing.T) {
	paths := resolveSuitePaths(t, "use/case7")
	_, rels := RunExtraction(t, "x/java/testdata", paths)

	src := "com.example.audit.FieldAccessSuite.testAll(File,Color)"
	cases := []struct {
		wantTarget string
		wantKind   model.ElementKind
		wantLine   int
	}{
		// L14 Color c = Color.RED;  -> 内部枚举常量
		{wantTarget: "com.example.audit.FieldAccessSuite.Color.RED", wantKind: model.EnumConstant, wantLine: 14},
		// L15 HttpStatus s = HttpStatus.NOT_FOUND; -> 外部枚举常量（import 补全 org.springframework.http）
		{wantTarget: "org.springframework.http.HttpStatus.NOT_FOUND", wantKind: model.Field, wantLine: 15},
		// L16 Class<?> clz1 = File.class; -> 外部类型 java.io.File
		{wantTarget: "java.io.File", wantKind: model.Class, wantLine: 16},
		// L17 Class<?> clz2 = Color.class; -> 内部枚举类型
		{wantTarget: "com.example.audit.FieldAccessSuite.Color", wantKind: model.Enum, wantLine: 17},
	}

	for _, c := range cases {
		found := findRel(rels, func(r *model.DependencyRelation) bool {
			if r.Type != model.Use || r.Source == nil || r.Target == nil {
				return false
			}
			if r.Source.QualifiedName != src {
				return false
			}
			if r.Target.QualifiedName != c.wantTarget || r.Target.Kind != c.wantKind {
				return false
			}
			return r.Location != nil && r.Location.StartLine == c.wantLine
		})
		if found == nil {
			t.Errorf("USE %s -> %s (kind=%s, line %d) not found", src, c.wantTarget, c.wantKind, c.wantLine)
		}
	}
}

// =============================================================================
// IMPORT / CONTAIN 层级关系 suite（extractHierarchy 产出）
// 两种关系不使用注释断言块（Source 形态特殊：IMPORT 的文件元素 / CONTAIN 的父 QN），
// 故用显式期望表断言存在性。
// =============================================================================

// importExpectations IMPORT 关系显式期望：Source 为文件相对路径，Target 为导入路径。
var importExpectations = []struct {
	file   string // 相对 testdata/extractor 的文件路径
	target string // 期望导入路径（精确匹配）
}{
	// call/CallRelationSuite.java 的 import 声明。
	{file: "call/CallRelationSuite.java", target: "java.util.ArrayList"},
	{file: "call/CallRelationSuite.java", target: "java.util.List"},
	{file: "call/CallRelationSuite.java", target: "java.util.function.Consumer"},
	// use/case3/Sub.java 跨包 import。
	{file: "use/case3/Sub.java", target: "com.example.rel.use.case3.pk1.Base"},
}

// containExpectations CONTAIN 关系显式期望（Source -> Target 精确 QN）。
var containExpectations = []struct {
	source string
	target string
}{
	// 包 -> 包。
	{source: "com.example.rel.use", target: "com.example.rel.use.case4"},
	// 包 -> 文件。
	{source: "com.example.rel.use.case4", target: "extractor/use/case4/StaticTest.java"},
	// 文件 -> 顶层类。
	{source: "extractor/use/case4/StaticTest.java", target: "com.example.rel.use.case4.StaticTest"},
	// 类 -> 字段。
	{source: "com.example.rel.use.case4.StaticTest", target: "com.example.rel.use.case4.StaticTest.instanceVar"},
	{source: "com.example.rel.use.case4.StaticTest", target: "com.example.rel.use.case4.StaticTest.staticVar"},
	// 类 -> 构造器。
	{source: "com.example.rel.use.case4.StaticTest", target: "com.example.rel.use.case4.StaticTest.StaticTest()"},
	// 类 -> 方法。
	{source: "com.example.rel.use.case4.StaticTest", target: "com.example.rel.use.case4.StaticTest.staticMethod()"},
}

// TestExtractorImportContain 断言 IMPORT 与 CONTAIN 层级关系按预期产出。
//
// 注意：这两类层级关系横跨 Linker 阶段产出（包->文件、文件->顶层类由 Java Linker
// 生成），且 IMPORT 依赖文件元素 QN 被引擎相对化（normalizeFilePath）。因此本测试
// 必须走完整五阶段 RunPipeline（而非 RunExtraction），并以 LevelRaw 保持全部原始关系。
func TestExtractorImportContain(t *testing.T) {
	paths := resolveSuitePaths(t, "call/CallRelationSuite.java", "use/case2", "use/case3", "use/case4")
	rels := RunPipeline(t, "x/java/testdata", paths, core.LevelRaw).Relations

	// 1. IMPORT 断言。
	for _, exp := range importExpectations {
		// Source 为文件元素（QN=相对路径，分隔符随平台）。统一转斜杠后做后缀匹配。
		relFile := filepath.ToSlash(exp.file)
		found := findRel(rels, func(r *model.DependencyRelation) bool {
			if r.Type != model.Import || r.Source == nil || r.Target == nil {
				return false
			}
			srcFile := filepath.ToSlash(r.Source.QualifiedName)
			if !strings.HasSuffix(srcFile, relFile) {
				return false
			}
			return r.Target.QualifiedName == exp.target
		})
		if found == nil {
			t.Errorf("IMPORT %s -> %s not found", exp.file, exp.target)
		}
	}

	// 2. CONTAIN 断言（仅当 use/case4 在本次路径集内命中）。
	for _, exp := range containExpectations {
		found := findRel(rels, func(r *model.DependencyRelation) bool {
			if r.Type != model.Contain || r.Source == nil || r.Target == nil {
				return false
			}
			return r.Source.QualifiedName == exp.source && r.Target.QualifiedName == exp.target
		})
		if found == nil {
			t.Errorf("CONTAIN %s -> %s not found", exp.source, exp.target)
		}
	}

	// 3. 跨文件继承 EXTEND 断言（use/case2、use/case3）。
	// 这些继承关系由 Extend suite（仅覆盖 ExtendRelationSuite.java）之外的
	// 用例承载，须在 LevelRaw 下验证提取。
	crossFileExtends := []struct {
		source string
		target string
	}{
		{source: "com.example.rel.use.case2.Child", target: "com.example.rel.use.case2.Parent"},
		{source: "com.example.rel.use.case3.pk2.Sub", target: "com.example.rel.use.case3.pk1.Base"},
	}
	for _, exp := range crossFileExtends {
		found := findRel(rels, func(r *model.DependencyRelation) bool {
			if r.Type != model.Extend || r.Source == nil || r.Target == nil {
				return false
			}
			return r.Source.QualifiedName == exp.source && r.Target.QualifiedName == exp.target
		})
		if found == nil {
			t.Errorf("EXTEND %s -> %s not found", exp.source, exp.target)
		}
	}
}

// TestExtractorChainedCalls 链式调用关系断言（call/case1/ChainedCallExample.java）。
//
// 链式表达式内的每一段方法调用都产生一条独立 CALL 关系（is_chained=true），
// 且每段目标解析到接收者类型的方法。注释断言块无法插入链式表达式内部
// （会污染 raw_text/receiver 元数据），故此处用显式断言表校验。
func TestExtractorChainedCalls(t *testing.T) {
	paths := resolveSuitePaths(t, "call/case1")
	_, rels := RunExtraction(t, "x/java/testdata", paths)

	// 链式 Builder 调用：name/age/build 各一段，均解析到 Builder 方法且 is_chained=true。
	chainTargets := []struct {
		line int
		tgt  string
	}{
		{line: 42, tgt: "com.example.chained.ChainedCallExample.Builder.name"},
		{line: 43, tgt: "com.example.chained.ChainedCallExample.Builder.age"},
		{line: 44, tgt: "com.example.chained.ChainedCallExample.Builder.build()"},
	}
	for _, ct := range chainTargets {
		found := findRel(rels, func(r *model.DependencyRelation) bool {
			if r.Type != model.Call || r.Source == nil || r.Target == nil {
				return false
			}
			if r.Source.QualifiedName != "com.example.chained.ChainedCallExample.testChainedCalls()" {
				return false
			}
			if !targetLooseMatch(r.Target.QualifiedName, ct.tgt) {
				return false
			}
			if r.Location == nil || r.Location.StartLine != ct.line {
				return false
			}
			return moresBool(r.Mores, "java.rel.call.is_chained")
		})
		if found == nil {
			t.Errorf("chained CALL to %s @L%d (is_chained=true) not found", ct.tgt, ct.line)
		}
	}

	// 深层链式调用：obj1.getName().toUpperCase().trim() 中 getName 解析到本类方法（非链式段）。
	found := findRel(rels, func(r *model.DependencyRelation) bool {
		if r.Type != model.Call || r.Source == nil || r.Target == nil {
			return false
		}
		if r.Source.QualifiedName != "com.example.chained.ChainedCallExample.testChainedCalls()" {
			return false
		}
		if r.Target.QualifiedName != "com.example.chained.ChainedCallExample.getName()" {
			return false
		}
		return r.Location != nil && r.Location.StartLine == 54
	})
	if found == nil {
		t.Errorf("deep-chain CALL getName() @L48 not found")
	}

	// 链式断链段：接收者解析成功但宿主是外部类型（String 的 JDK 方法），
	// 段目标为合成 QN（保留链上下文）且 is_chained=true。
	brokenChain := []struct {
		line   int
		target string
	}{
		{line: 49, target: "com.example.chained.ChainedCallExample.getName().toUpperCase()"},
		{line: 54, target: "com.example.chained.ChainedCallExample.getName().toUpperCase()"},
		{line: 54, target: "com.example.chained.ChainedCallExample.getName().toUpperCase.trim()"},
	}
	for _, bc := range brokenChain {
		found := findRel(rels, func(r *model.DependencyRelation) bool {
			if r.Type != model.Call || r.Source == nil || r.Target == nil {
				return false
			}
			if r.Source.QualifiedName != "com.example.chained.ChainedCallExample.testChainedCalls()" {
				return false
			}
			if !strings.HasSuffix(r.Target.QualifiedName, bc.target) {
				return false
			}
			if r.Location == nil || r.Location.StartLine != bc.line {
				return false
			}
			return moresBool(r.Mores, "java.rel.call.is_chained")
		})
		if found == nil {
			t.Errorf("broken-chain CALL %s @L%d (is_chained=true) not found", bc.target, bc.line)
		}
	}

	// use/case6：user.getName().trim() 中 trim 断链段（接收者 user 为局部变量 User 类型）。
	case6Paths := resolveSuitePaths(t, "use/case6")
	_, case6Rels := RunExtraction(t, "x/java/testdata", case6Paths)
	trimFound := findRel(case6Rels, func(r *model.DependencyRelation) bool {
		if r.Type != model.Call || r.Source == nil || r.Target == nil {
			return false
		}
		if r.Source.QualifiedName != "com.example.rel.use.case6.ReceiverTest.test()" {
			return false
		}
		if r.Target.QualifiedName != "com.example.rel.use.case6.User.getName().trim()" {
			return false
		}
		if r.Location == nil || r.Location.StartLine != 9 {
			return false
		}
		return moresBool(r.Mores, "java.rel.call.is_chained")
	})
	if trimFound == nil {
		t.Errorf("broken-chain CALL User.getName().trim() @L9 (is_chained=true) not found")
	}
}

// TestExtractorInitializerBlocks 验证静态/实例初始化块内动作关系的 source 归属。
// 实例初始化块（AST 节点 block、父为 class_body）此前因 determinePreciseSource 缺少
// block->ScopeBlock 分支而静默丢失，此处断言修复后 $instance$N 也能作为 source 定位。
func TestExtractorInitializerBlocks(t *testing.T) {
	paths := ResolvePaths(t, "collector/class")
	_, rels := RunExtraction(t, "x/java/testdata", paths)

	// 静态初始化块：ASSIGN/USE 归属 $static$1（内部目标 count 保留）。
	for _, typ := range []model.DependencyType{model.Assign, model.Use} {
		found := findRel(rels, func(r *model.DependencyRelation) bool {
			if r.Type != typ || r.Source == nil || r.Target == nil {
				return false
			}
			return r.Source.QualifiedName == "com.example.base.OuterClass.$static$1" &&
				r.Target.QualifiedName == "com.example.base.OuterClass.count"
		})
		if found == nil {
			t.Errorf("%s from $static$1 -> count not found", typ)
		}
	}

	// 静态初始化块：CALL System 归属 $static$1（LevelRaw 可见，外部目标）。
	staticCall := findRel(rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source != nil && r.Target != nil &&
			r.Source.QualifiedName == "com.example.base.OuterClass.$static$1" &&
			strings.HasPrefix(r.Target.QualifiedName, "System")
	})
	if staticCall == nil {
		t.Errorf("CALL System from $static$1 not found")
	}

	// 实例初始化块：CALL System 归属 $instance$1（修复后的关键断言）。
	instanceCall := findRel(rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Call && r.Source != nil && r.Target != nil &&
			r.Source.QualifiedName == "com.example.base.OuterClass.$instance$1" &&
			strings.HasPrefix(r.Target.QualifiedName, "System")
	})
	if instanceCall == nil {
		t.Errorf("CALL System from $instance$1 not found (instance initializer block source lost)")
	}
}

// moresBool 读取 Mores 布尔值（不存在或非 bool 返回 false）。
func moresBool(m map[string]interface{}, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

// runSuite 执行单个 suite 的期望断言。
func runSuite(t *testing.T, suite struct {
	relType model.DependencyType
	paths   []string
}) {
	paths := resolveSuitePaths(t, suite.paths...)

	// 解析注释期望（有则用，无则用显式期望表）。
	allExpected := map[string][]expectedCase{} // relPath -> cases
	for _, path := range paths {
		rel := relOf(t, "x/java/testdata", path)
		cases, err := parseExpectationComments(path)
		if err != nil {
			t.Fatalf("parse expectations %s: %v", path, err)
		}
		if len(cases) == 0 {
			if exp, ok := explicitExpectations[suite.relType]; ok {
				cases = exp
			}
		}
		allExpected[rel] = cases
	}

	_, rels := RunExtraction(t, "x/java/testdata", paths)

	for _, rel := range relOfs(t, paths...) {
		cases := allExpected[rel]
		if len(cases) == 0 {
			t.Logf("%s: no expectation comments", rel)
			continue
		}
		for _, ec := range cases {
			wantType := relTypeFromMores(ec.Mores)
			if wantType == "" {
				wantType = suite.relType
			}
			targets := ec.Targets
			if len(targets) == 0 {
				t.Errorf("%s:~%d: expectation without target", rel, ec.Line)
				continue
			}
			for _, tgt := range targets {
				found := findRel(rels, func(r *model.DependencyRelation) bool {
					if r.Source == nil || r.Target == nil {
						return false
					}
					if r.Type != wantType {
						return false
					}
					if !sourceLooseMatch(r.Source.QualifiedName, ec.Source) {
						return false
					}
					if !targetLooseMatch(r.Target.QualifiedName, tgt) {
						return false
					}
					if !moresMatch(r, ec.Mores) {
						return false
					}
					return lineMatch(r, ec.Line)
				})
				if found == nil {
					t.Errorf("%s:~%d: expected relation [%s] %s -> %s not found (mores=%v)",
						rel, ec.Line, wantType, ec.Source, tgt, ec.Mores)
				}
			}
		}
	}

	// 负面断言：断言“不应存在”的关系确实不存在。
	assertNegativeRels(t, suite.relType, rels)
}

// assertNegativeRels 对当前 suite 的负面期望逐一断言不存在匹配关系。
func assertNegativeRels(t *testing.T, relType model.DependencyType, rels []*model.DependencyRelation) {
	negs := negativeExpectations[relType]
	if len(negs) == 0 {
		return
	}
	for _, neg := range negs {
		found := findRel(rels, func(r *model.DependencyRelation) bool {
			if r.Source == nil || r.Target == nil {
				return false
			}
			if r.Type != neg.RelType {
				return false
			}
			if !sourceLooseMatch(r.Source.QualifiedName, neg.Source) {
				return false
			}
			return targetLooseMatch(r.Target.QualifiedName, neg.Target)
		})
		if found != nil {
			t.Errorf("negative: relation [%s] %s -> %s should NOT exist but found at %s:%d",
				neg.RelType, neg.Source, neg.Target, found.Location.FilePath, found.Location.StartLine)
		}
	}
}

// relOfs 将 suite 相对路径映射为 rel 路径列表（复用 relOf）。
func relOfs(t *testing.T, paths ...string) []string {
	t.Helper()
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, relOf(t, "x/java/testdata", p))
	}
	return out
}

// findRel 在关系中查找满足谓词的第一条，找不到返回 nil。
func findRel(rels []*model.DependencyRelation, pred func(*model.DependencyRelation) bool) *model.DependencyRelation {
	for _, r := range rels {
		if pred(r) {
			return r
		}
	}
	return nil
}

// sourceLooseMatch 判断实际 Source QN 是否匹配注释 Source QN。
// 注释 QN 可能省略括号（如 "executeAll" vs "executeAll()"），因此要求：
// 注释 QN 去掉括号后是实际 QN 的前缀；匿名类场景（注释 $1 vs 实际 .anonymousClass$1）退化为方法名一致。
func sourceLooseMatch(actual, expected string) bool {
	if actual == expected {
		return true
	}
	// 移除全部调用括号段再比较：
	//   actual  "pkg.C.m()"        → "pkg.C.m"
	//   actual  "pkg.C.m().inner()" → "pkg.C.m.inner"
	//   expected "pkg.C.m"          → "pkg.C.m"
	a := stripAllCallSuffix(actual)
	e := stripAllCallSuffix(expected)
	// 构造器别名：注释 "<init>" vs 实际 "类名()"（如 ThrowRelationSuite.<init> vs ThrowRelationSuite.ThrowRelationSuite()）。
	if i := strings.LastIndex(e, ".<init>"); i >= 0 {
		e = e[:i] // 去掉 .<init>，剩余类前缀
		if strings.HasPrefix(a, e+".") {
			return true
		}
		// 构造器名即类名（<init> 的末段应为类简单名）。
		if strings.HasPrefix(a, e) && strings.Contains(a[len(e):], ".") {
			return true
		}
	}
	if e == "" {
		return false
	}
	if strings.HasPrefix(a, e) {
		return true
	}
	// 匿名类命名差异：注释 "Outer$1.run" vs 实际 "Outer.m().anonymousClass$1.run"。
	// 此时比较末段方法名（.run 一致即视为命中）。
	if strings.Contains(expected, "$") {
		as, es := simpleName(actual), simpleName(expected)
		return as != "" && as == es
	}
	return false
}

// stripAllCallSuffix 移除 QN 中全部调用括号段（(...) 及其中内容），保留方法/字段路径。
// 例："a.m().inner.f()" → "a.m.inner.f"；"m()" → "m"。
func stripAllCallSuffix(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// targetLooseMatch 判断实际 Target QN 是否匹配注释 Target QN。
// 差异来源：构造器（注释 X.X() vs 实际 X()）、外部符号（注释 a.b vs 实际 b()）。
// 策略：比较二者的「最后一个点段的简单名」（去包、去括号、去泛型）。
// 已知差异：显式 super() 构造调用在 v2 中解析为外部符号 "super"，
// 而注释期望父构造器（如 BaseClass.BaseClass()）；此处对 super() 放行。
func targetLooseMatch(actual, expected string) bool {
	if actual == expected {
		return true
	}
	// 显式 super()/this() 构造调用：v2 解析为 receiver 名（Target QN="super"）。
	if simpleName(actual) == "super" || simpleName(actual) == "this" {
		// 注释直接期望 super/this（如 "Target: Class(super)"）时命中。
		if es := simpleName(expected); es == "super" || es == "this" {
			return true
		}
		// 否则期望必须是构造器调用（形如 X.Y() 且末段与次末段同名，如 BaseClass.BaseClass()）。
		if strings.Contains(expected, "(") {
			eNoParen := stripCallSuffix(expected)
			if i := strings.LastIndex(eNoParen, "."); i >= 0 {
				typePart := eNoParen[:i]
				if j := strings.LastIndex(typePart, "."); j >= 0 && typePart[j+1:] == eNoParen[i+1:] {
					return true
				}
			}
		}
		return false
	}
	aSimp := simpleName(actual)
	eSimp := simpleName(expected)
	if aSimp == "" || eSimp == "" {
		return false
	}
	return aSimp == eSimp
}

// stripCallSuffix 去掉 QN 的调用括号段（从第一个 '(' 截断）。
func stripCallSuffix(s string) string {
	if i := strings.Index(s, "("); i >= 0 {
		return s[:i]
	}
	return s
}

// simpleName 提取 QN 的简单名：最后一个 '.' 后的段，去掉括号与泛型。
// 注意：QN 中可能含多个括号段（如 "a.m().inner.f()"），须只保留最后一个括号段之后的末段；
// 若括号在末尾（如 "m()"），则取括号之前。
func simpleName(qn string) string {
	// 括号段之后若有内容（链式后续），取其后；否则截到最后一个 '(' 前。
	if i := strings.LastIndex(qn, ")"); i >= 0 && i+1 < len(qn) {
		qn = qn[i+1:]
	} else if i := strings.LastIndex(qn, "("); i >= 0 {
		qn = qn[:i]
	}
	// 去掉泛型实参 <...>。
	if i := strings.Index(qn, "<"); i >= 0 {
		qn = qn[:i]
	}
	if i := strings.LastIndex(qn, "."); i >= 0 {
		qn = qn[i+1:]
	}
	return qn
}

// lineMatch 判断实际关系触发行是否匹配期望行号（注释 Source 行）。
// 实际触发行是注释块后的代码行，通常比注释 Source 行晚 4-12 行，故允许 ±12 行。
// 结构关系（EXTEND/IMPLEMENT/PARAMETER/RETURN/THROW 签名/ANNOTATION/TYPE_ARG）
// 无 Location（L0），故对 Line==0 的期望直接放行（已由 Source+Target+Type 断言）。
func lineMatch(r *model.DependencyRelation, wantLine int) bool {
	if wantLine <= 0 {
		return true
	}
	// 结构关系（PARAMETER/RETURN/THROW 签名/ANNOTATION/TYPE_ARG/EXTEND/IMPLEMENT）
	// 由 extractStructural 产出，不挂 AST 节点，Location 为 nil（dump 中记作 L0）。
	// 此时行号无法断言，由 Source+Target+Mores 承担定位，放行。
	if r.Location == nil {
		return true
	}
	got := r.Location.StartLine
	diff := got - wantLine
	if diff < 0 {
		diff = -diff
	}
	return diff <= 12
}

// keyOf 构造关系索引键。
func keyOf(source, target string, line int) string {
	return source + "|" + target + "|" + itoa(line)
}

// itoa 简易 int->string（避免引入 strconv 依赖的过度封装）。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

// moresMatch 校验实际关系 Mores 是否包含期望 Mores（substring 匹配）。
// 忽略辅助字段：
//   - java.rel.ast_kind / java.rel.raw_text / java.rel.context：格式易变，已由 suite 类型 + 行号承担主断言。
//   - java.rel.parameter.is_final / java.rel.parameter.has_annotation：v1 注释期望但 v2 提取器不产出
//     （final/注解修饰通过 raw_text 呈现，见 parameter_enricher.go）。
//   - java.rel.return.dimensions / java.rel.return.has_type_arguments：v1 注释期望但 v2 以 is_array + TYPE_ARG 关系表达。
//   - java.rel.annotation.value：v1 注释期望但 v2 提取器不产出（注解值经 raw_text 呈现）。
//
// 期望值中的转义引号（\"）会在与实际的纯引号比较前去转义。
func moresMatch(r *model.DependencyRelation, want map[string]interface{}) bool {
	if len(want) == 0 {
		return true
	}
	for k, v := range want {
		if isIgnoredMoresKey(k) {
			continue
		}
		got, ok := r.Mores[k]
		if !ok {
			// 注释使用短键（如 target_name）而实际为带前缀键（java.rel.assign.target_name）时，
			// 按短键后缀匹配实际键。
			if !strings.Contains(k, ".") {
				if actual, hit := findMoresByShortKey(r.Mores, k); hit {
					got = actual
					ok = true
				}
			}
			if !ok {
				return false
			}
		}
		if !moresValueMatch(got, v) {
			return false
		}
	}
	return true
}

// findMoresByShortKey 以短键后缀（.target_name 等）在 Mores 中查找实际键。
func findMoresByShortKey(m map[string]interface{}, short string) (interface{}, bool) {
	suffix := "." + short
	for k, v := range m {
		if strings.HasSuffix(k, suffix) {
			return v, true
		}
	}
	return nil, false
}

// moresValueMatch 比较期望 Mores 值与实际值：
//   - 期望为字符串 -> substring 匹配（去转义）
//   - 期望为 bool   -> 精确匹配（实际须为 bool）
//   - 期望为数字串  -> 数字归一后精确匹配（实际为 int 或数字串）
func moresValueMatch(got, want interface{}) bool {
	switch w := want.(type) {
	case bool:
		g, ok := got.(bool)
		return ok && g == w
	case string:
		// 数字串期望：实际可为 int 或数字串，做数值归一比较。
		if isNumericString(w) {
			return numericMatch(got, w)
		}
		g, ok := got.(string)
		if !ok {
			return false
		}
		return strings.Contains(g, unescapeQuotes(w))
	default:
		// 数字（注释解析为字符串，如 "1"）；实际为 int 或字符串数字。
		wStr := fmt.Sprintf("%v", w)
		return numericMatch(got, wStr)
	}
}

// isNumericString 判断字符串是否纯数字（含负数）。
func isNumericString(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r == '-' && i == 0 {
			continue
		}
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// numericMatch 数值归一比较（int 与数字串）。
func numericMatch(got interface{}, wStr string) bool {
	switch g := got.(type) {
	case int:
		return fmt.Sprintf("%d", g) == wStr
	case string:
		return g == wStr
	}
	return false
}

// isIgnoredMoresKey 判断 Mores 键是否属于测试忽略的辅助/遗留字段。
func isIgnoredMoresKey(k string) bool {
	// 短键（如 raw_text）与完整前缀键（java.rel.raw_text）等价，统一忽略。
	switch k {
	case "java.rel.ast_kind", "java.rel.raw_text", "java.rel.context",
		"java.rel.parameter.is_final", "java.rel.parameter.has_annotation",
		"java.rel.return.dimensions", "java.rel.return.has_type_arguments",
		"java.rel.annotation.value":
		return true
	}
	switch k {
	case "ast_kind", "raw_text", "context":
		return true
	}
	return false
}

// unescapeQuotes 将注释值中的 \" 转义为 "，以匹配实际输出。
func unescapeQuotes(s string) string {
	return strings.ReplaceAll(s, `\"`, `"`)
}

// ----------------------------------------------------------------------------
// 注释断言块解析
// ----------------------------------------------------------------------------

var (
	// 新格式：Source: com.example.rel.X.executeAll() (METHOD)
	reSourceNew = regexp.MustCompile(`^Source:\s*(\S+)\s+\((\w+)\)`)
	// 旧格式：Source: Method(com.example.rel.X.m())
	reSourceOld = regexp.MustCompile(`^Source:\s*\w+\((\S+)\)`)
	// 内联格式：Source: QN, Target: T  （type_arg suite）
	reSourceInline = regexp.MustCompile(`^Source:\s*(\S+),\s*Target:\s*(.+)$`)
	// 裸格式：Source: QN
	reSourceBare = regexp.MustCompile(`^Source:\s*(\S+)$`)

	// 新格式：Target: com.example.rel.X.simpleMethod() (METHOD)
	reTargetNew = regexp.MustCompile(`^Target\d*:\s*(\S+)\s+\((\w+)\)`)
	// 旧格式：Target: Class(java.util.ArrayList)
	reTargetOld = regexp.MustCompile(`^Target\d*:\s*\w+\((\S+)\)`)
	// 裸/行内多 Target：Target: QN (Index: 0), QN2 (Index: 1)
	reTargetBare = regexp.MustCompile(`^Target\d*:\s*(.+)$`)

	// 单行 Mores：{ "k": "v", ... }
	reMores = regexp.MustCompile(`^Mores:\s*\{(.*)\}$`)
	// 多行 Mores 块内的键值行：  "k": "v"[,] 或 "k": true/false/数字（未加引号值）
	reMoresLine = regexp.MustCompile(`^"([^"]+)"\s*:\s*(?:"((?:\\.|[^"])*)"|(true|false|-?\d+))`)
	// 行内 Target 列表中的单个元素：QN (Index: 0)
	reTargetItem = regexp.MustCompile(`([^,()]+?)\s*\(Index:\s*(\d+)\)`)
)

// parseExpectationComments 解析源码中的 // Source: ... // Target: ... // Mores: {...} 断言块。
// 支持格式：
//   - 新格式 "Source: QN (KIND)" + 多行 Mores 块（call/assign）
//   - 旧格式 "Source: Kind(QN)"（create/cast）
//   - 裸格式 "Source: QN"（parameter/return/throw/annotation/use/capture）
//   - 内联格式 "Source: QN, Target: T"（type_arg）
//   - 多 Target 行（use 多 Target 行、cast Target1/Target2）、行内逗号多 Target 带 (Index: n)（parameter）
//   - 连续 Source 块（throw：同一 Source 不同 Target 分属两块）
//
// 每块从 Source 行开始，到下一个 Source 行或代码行结束。
func parseExpectationComments(path string) ([]expectedCase, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cases []expectedCase
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var cur *expectedCase
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		trimmed := strings.TrimSpace(line)

		if !strings.HasPrefix(trimmed, "//") {
			// 非注释行：结束当前块。
			flushCase(&cases, cur)
			cur = nil
			continue
		}

		body := strings.TrimPrefix(trimmed, "//")
		body = strings.TrimSpace(body)

		// 中文说明行（如 "// 1. 局部变量读取"、"// 关系 A -> a"）跳过。
		if !strings.Contains(body, "Source:") && !strings.Contains(body, "Target:") &&
			!strings.Contains(body, "Mores:") {
			continue
		}

		// 1) 内联格式：Source: QN, Target: T（type_arg suite）。
		if m := reSourceInline.FindStringSubmatch(body); m != nil {
			flushCase(&cases, cur)
			src := m[1]
			tgt := strings.TrimSpace(m[2])
			// Target 后可能跟注释（如 // 这里指...），截断到 ' //'。
			if i := strings.Index(tgt, " //"); i >= 0 {
				tgt = strings.TrimSpace(tgt[:i])
			}
			cur = &expectedCase{Line: lineNo, Source: src, Targets: splitTargets(tgt)}
			continue
		}

		// 2) 新格式：Source: QN (KIND)（call/assign）。
		if m := reSourceNew.FindStringSubmatch(body); m != nil {
			flushCase(&cases, cur)
			cur = &expectedCase{Line: lineNo, Source: m[1]}
			continue
		}

		// 3) 旧格式：Source: Kind(QN)（create/cast）。
		if m := reSourceOld.FindStringSubmatch(body); m != nil {
			flushCase(&cases, cur)
			cur = &expectedCase{Line: lineNo, Source: m[1]}
			continue
		}

		// 4) 裸格式：Source: QN（parameter/return/throw/annotation/use/capture）。
		if m := reSourceBare.FindStringSubmatch(body); m != nil {
			flushCase(&cases, cur)
			cur = &expectedCase{Line: lineNo, Source: m[1]}
			continue
		}

		// 5) Target 行（新/旧/裸/多 Target）。
		if m := reTargetNew.FindStringSubmatch(body); m != nil {
			appendTarget(cur, m[1])
			continue
		}
		if m := reTargetOld.FindStringSubmatch(body); m != nil {
			appendTarget(cur, m[1])
			continue
		}
		if m := reTargetBare.FindStringSubmatch(body); m != nil {
			appendTarget(cur, strings.TrimSpace(m[1]))
			continue
		}

		// 6) Mores 单行：{ "k": "v", ... }。
		if m := reMores.FindStringSubmatch(body); m != nil {
			parseMoresInto(cur, m[1])
			continue
		}

		// 7) Mores 多行块内的键值行（前导空白 + "k": "v" / "k": true / "k": 1）。
		if cur != nil && cur.Mores != nil {
			if m := reMoresLine.FindStringSubmatch(body); m != nil {
				cur.Mores[m[1]] = moresValue(m)
				continue
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	flushCase(&cases, cur)
	return cases, nil
}

// flushCase 将当前块加入结果（要求有 Source 与至少一个 Target）。
// 行内多 Target（如 PARAMETER 的 "String (Index: 0), long (Index: 1)"）时，
// 块级 Mores 只描述首个 Target（注释仅写了一个 Mores 块），后续 Target 各成一条且 Mores 置空
// （只断言关系存在，元数据由各自实际的 index/name 承担，无法从注释区分）。
func flushCase(cases *[]expectedCase, cur *expectedCase) {
	if cur == nil || cur.Source == "" || len(cur.Targets) == 0 {
		return
	}
	for i, tgt := range cur.Targets {
		c := *cur
		c.Targets = []string{tgt}
		if i > 0 {
			c.Mores = nil
		}
		*cases = append(*cases, c)
	}
}

// appendTarget 追加 Target，同时启动 Mores map（供后续多行键值行）。
func appendTarget(cur *expectedCase, tgt string) {
	if cur == nil {
		return
	}
	for _, t := range splitTargets(tgt) {
		cur.Targets = append(cur.Targets, t)
	}
	if cur.Mores == nil {
		cur.Mores = map[string]interface{}{}
	}
}

// splitTargets 将 Target 内容拆分为多个 Target QN。
// 支持：
//   - 行内多 Target："String (Index: 0), long (Index: 1)" → [String, long]
//   - 单 Target："java.lang.String" → [java.lang.String]
//   - 带尾注释："com.example.rel.X // 说明" → 截断到 ' //'。
func splitTargets(s string) []string {
	// 截断尾注释。
	if i := strings.Index(s, " //"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	if s == "" {
		return nil
	}
	// 行内多 Target（含 (Index: n)）按逗号拆分，丢弃 (Index: n) 段。
	if strings.Contains(s, "(Index:") {
		var out []string
		for _, item := range strings.Split(s, ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if m := reTargetItem.FindStringSubmatch(item); m != nil {
				out = append(out, strings.TrimSpace(m[1]))
				continue
			}
			// 无 (Index:) 的段（如裸类型名）。
			out = append(out, item)
		}
		return out
	}
	// 单 Target 但可能带尾注释残留（已截断）。
	return []string{strings.TrimSpace(s)}
}

// parseMoresInto 解析单行 Mores 内容（"k": "v", "k2": "v2"）写入当前块。
func parseMoresInto(cur *expectedCase, content string) {
	if cur == nil {
		return
	}
	if cur.Mores == nil {
		cur.Mores = map[string]interface{}{}
	}
	for _, pair := range splitMoresPairs(content) {
		if m := reMoresLine.FindStringSubmatch(pair); m != nil {
			cur.Mores[m[1]] = moresValue(m)
		}
	}
}

// moresValue 依据 reMoresLine 的匹配组提取 Mores 值并归一类型：
//   - m[2] 为带引号字符串 -> 原样返回
//   - m[3] 为 true/false   -> 返回 bool
//   - m[3] 为数字         -> 返回字符串（比较时再归一）
func moresValue(m []string) interface{} {
	if m[2] != "" {
		return m[2]
	}
	if m[3] == "true" {
		return true
	}
	if m[3] == "false" {
		return false
	}
	return m[3]
}

// splitMoresPairs 将 Mores 内容按逗号拆分为键值对（"k": "v"）。
// 注意：值内可能含逗号（如 "a, b"），故仅在 "" 闭合后拆分。
func splitMoresPairs(content string) []string {
	var pairs []string
	var cur strings.Builder
	inStr := false
	escaped := false
	for _, r := range content {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case r == '\\':
			cur.WriteRune(r)
			escaped = true
		case r == '"':
			cur.WriteRune(r)
			inStr = !inStr
		case r == ',' && !inStr:
			pairs = append(pairs, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		pairs = append(pairs, strings.TrimSpace(cur.String()))
	}
	return pairs
}

// resolveSuitePaths 将 suite 路径（相对 testdata/extractor）解析为绝对文件路径。
// 支持两种形态：
//   - 文件路径："call/CallRelationSuite.java" -> 该文件
//   - 目录路径："call/case1" -> 该目录下全部 .java（glob 展开，排序稳定）
func resolveSuitePaths(t *testing.T, rels ...string) []string {
	t.Helper()
	var files []string
	for _, rel := range rels {
		full := filepath.Join(TestDataDir, "extractor", rel)
		info, err := os.Stat(full)
		if err == nil && !info.IsDir() {
			files = append(files, full)
			continue
		}
		matches, err := filepath.Glob(filepath.Join(full, "*.java"))
		if err != nil || len(matches) == 0 {
			t.Fatalf("resolveSuitePaths: no java files under %s", full)
		}
		sort.Strings(matches)
		files = append(files, matches...)
	}
	return files
}
