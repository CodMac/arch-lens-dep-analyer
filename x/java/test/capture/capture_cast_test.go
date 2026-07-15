package capture_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

// CastPointExpectation 包含定位 Cast/Instanceof 捕获点所需的核心断言元数据
type CastPointExpectation struct {
	SceneName  string
	LineNum    int
	TargetText string // 预期捕获到的完整强转或类型检查表达式
}

func TestCapture_CastCase(t *testing.T) {
	// 1. 获取打样文件的 AST 树上下文 (非侵入式加载)
	testFile := test.GetTestFilePath(filepath.Join("capture", "CastCase.java"))
	gCtx := test.RunPhase1Collection(t, []string{testFile})
	fCtx := gCtx.FileContexts[testFile]

	// 2. 初始化核心规则与待测 Extractor
	captures, err := java.NewJavaExtractor().GetCaptures(fCtx)
	if err != nil {
		t.Fatal(err)
	}

	// 3. 构造 8 个场景下预期的 Cast 捕获点点位 (基于完整表达式匹配)
	expectations := []CastPointExpectation{
		// ==================== 基础与经典强转场景 ====================
		{SceneName: "场景1: 基础向下转型", LineNum: 14, TargetText: "(String) obj"},
		{SceneName: "场景2: 基础数据类型转换", LineNum: 17, TargetText: "(double) intVal"},
		{SceneName: "场景3: 带泛型的集合转型", LineNum: 20, TargetText: "(List<String>) obj"},
		{SceneName: "场景4: 全限定类名转型", LineNum: 23, TargetText: "(java.util.Map) input"},

		// ==================== 复杂强转场景 (链式/多重) ====================
		{SceneName: "场景5: 多重强转 (Double Cast) 并伴随链式调用", LineNum: 27, TargetText: "(Runnable)(Object)input"},
		{SceneName: "场景6: 链式调用中的转型", LineNum: 30, TargetText: "(SubClass) obj"},

		// ==================== 类型检查与模式匹配 ====================
		{SceneName: "场景7: 传统 instanceof 检查", LineNum: 34, TargetText: "obj instanceof String"},
		{SceneName: "场景8: Java 14+ 模式匹配 instanceof", LineNum: 39, TargetText: "obj instanceof String str"},
	}

	// 4. 提取实际捕获到的所有原始点位标识
	type CapturedPoint struct {
		LineNum int
		RawText string
	}
	actualCapturedPoints := make([]CapturedPoint, 0)

	for _, ct := range captures {
		// 只关注最新一元化重构后的 cast_target
		if ct.CapName != "cast_target" {
			continue
		}

		lineNum := int(ct.Node.StartPosition().Row) + 1
		rawText := ct.Node.Utf8Text(*fCtx.SourceBytes)

		// 格式化文本：合并多行并剔除多余空格，确保证书断言匹配稳定性
		cleanText := strings.Join(strings.Fields(rawText), " ")
		actualCapturedPoints = append(actualCapturedPoints, CapturedPoint{
			LineNum: lineNum,
			RawText: cleanText,
		})
	}

	// 5. 遍历并精确断言每一个 Cast 捕获点的表现
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
				// 输出友好的提示，便于精准定位到具体场景行号
				t.Errorf("【捕获遗漏】JavaActionQuery 规则未能在第 %d 行捕获到包含 [%s] 的转型表达式", exp.LineNum, exp.TargetText)
			}
		})
	}
}
