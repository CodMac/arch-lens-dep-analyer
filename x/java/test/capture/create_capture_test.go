package capture_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

// CreatePointExpectation 仅包含定位捕获点所需的核心元数据
type CreatePointExpectation struct {
	SceneName  string
	LineNum    int
	TargetText string // 预期捕获到的原始文本 (这里通常是整个 new 表达式的一部分特征文本即可，方便断言)
}

func TestCapture_CreateCase(t *testing.T) {
	// 1. 获取打样文件的 AST 树上下文 (非侵入式加载)
	testFile := test.GetTestFilePath(filepath.Join("capture", "CreateCase.java"))
	gCtx := test.RunPhase1Collection(t, []string{testFile})
	fCtx := gCtx.FileContexts[testFile]

	// 2. 初始化核心规则与待测 Resolver
	captures, err := java.NewJavaExtractor().GetCaptures(fCtx)
	if err != nil {
		t.Fatal(err)
	}

	// 3. 构造 16 个场景下预期的 Create 捕获点点位
	// 注意：现在由于我们捕获的是整个 object_creation_expression 或 array_creation_expression，
	// TargetText 应该包含 `new` 关键字及后续内容。
	expectations := []CreatePointExpectation{
		// ==================== 基础创建场景 ====================
		{SceneName: "场景1: 普通无参对象创建", LineNum: 11, TargetText: "new Object()"},
		{SceneName: "场景2: 带参对象创建", LineNum: 14, TargetText: "new String(\"Hello\")"},
		// 匿名内部类比较特殊，包含换行，所以用前缀匹配
		{SceneName: "场景3: 匿名内部类创建", LineNum: 17, TargetText: "new Runnable() {"},

		// ==================== 嵌套限定类型场景 ====================
		{SceneName: "场景4: 嵌套限定类型创建 (普通内部类)", LineNum: 25, TargetText: "new Outer.Inner()"},
		{SceneName: "场景5: 深层嵌套限定类型创建", LineNum: 28, TargetText: "new A.B.C()"},

		// ==================== 泛型场景 ====================
		{SceneName: "场景6: 菱形语法 (Diamond operator) 创建", LineNum: 33, TargetText: "new ArrayList<>()"},
		{SceneName: "场景7: 显式指定泛型类型的创建", LineNum: 36, TargetText: "new HashMap<String, Integer>()"},
		{SceneName: "场景8: 嵌套泛型类型的创建", LineNum: 39, TargetText: "new ArrayList<List<String>>()"},

		// ==================== 数组场景 ====================
		{SceneName: "场景9: 基本类型数组创建 (带长度)", LineNum: 44, TargetText: "new int[10]"},
		{SceneName: "场景10: 基本类型数组创建 (带初始值)", LineNum: 47, TargetText: "new int[]{1, 2, 3}"},
		{SceneName: "场景11: 对象类型数组创建", LineNum: 50, TargetText: "new String[5]"},
		{SceneName: "场景12: 多维数组创建", LineNum: 53, TargetText: "new int[3][4]"},

		// ==================== 链式调用场景 ====================
		{SceneName: "场景13: 创建后立即进行方法调用", LineNum: 58, TargetText: "new String(\"test\")"},
		{SceneName: "场景14: 创建后立即访问字段", LineNum: 61, TargetText: "new DummyClass()"},

		// ==================== 嵌套表达式场景 ====================
		{SceneName: "场景15: 在方法传参中创建对象", LineNum: 66, TargetText: "new DummyClass()"},
		{SceneName: "场景16: 在 Return 语句中创建对象", LineNum: 69, TargetText: "new DummyClass()"},
	}

	// 4. 提取实际捕获到的所有原始点位标识
	// 由于完整的 object_creation_expression 可能跨越多行（如匿名内部类），
	// 这里提取时，我们取它的第一行，并清理空格，方便做前缀或包含匹配。
	type CapturedPoint struct {
		LineNum int
		RawText string
	}
	actualCapturedPoints := make([]CapturedPoint, 0)

	for _, ct := range captures {
		// 只关注 Create 动作对应的目标节点
		if ct.CapName != "create_target" {
			continue
		}

		lineNum := int(ct.Node.StartPosition().Row) + 1
		rawText := ct.Node.Utf8Text(*fCtx.SourceBytes)

		// 为了比较方便，将多行合并，并替换多个空格为单个
		cleanText := strings.Join(strings.Fields(rawText), " ")
		actualCapturedPoints = append(actualCapturedPoints, CapturedPoint{
			LineNum: lineNum,
			RawText: cleanText,
		})
	}

	// 5. 对是否存在捕获点进行遍历断言
	for _, exp := range expectations {
		t.Run(exp.SceneName, func(t *testing.T) {
			found := false
			for _, act := range actualCapturedPoints {
				if act.LineNum == exp.LineNum && strings.Contains(act.RawText, exp.TargetText) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("【捕获遗漏】JavaActionQuery 规则未能在第 %d 行捕获到包含 [%s] 的创建表达式", exp.LineNum, exp.TargetText)
			}
		})
	}
}
