package capture_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

// UsePointExpectation 仅包含定位捕获点所需的核心元数据
type UsePointExpectation struct {
	SceneName  string
	LineNum    int
	TargetText string
}

func TestCapture_UseCase(t *testing.T) {
	// 1. 获取打样文件的 AST 树上下文 (非侵入式加载)
	testFile := test.GetTestFilePath(filepath.Join("capture", "UseCase.java"))
	gCtx := test.RunPhase1Collection(t, []string{testFile})
	fCtx := gCtx.FileContexts[testFile]

	// 2. 初始化核心规则与待测 Resolver
	captures, err := java.NewJavaExtractor().GetCaptures(fCtx)
	if err != nil {
		t.Fatal(err)
	}

	// 3. 构造 29 个场景下预期的 Use 捕获点点位
	expectations := []UsePointExpectation{
		// ==================== 简单变量使用场景 ====================
		{SceneName: "场景1: 算术表达式变量 a", LineNum: 53, TargetText: "a"},
		{SceneName: "场景1: 算术表达式变量 b", LineNum: 53, TargetText: "b"},
		{SceneName: "场景1: 除法表达式变量 b", LineNum: 56, TargetText: "b"},
		{SceneName: "场景2: 比较表达式变量 x", LineNum: 68, TargetText: "x"},
		{SceneName: "场景2: 比较表达式变量 y", LineNum: 68, TargetText: "y"},
		{SceneName: "场景3: 方法参数变量 message", LineNum: 81, TargetText: "message"},

		// ==================== 字段访问使用场景 ====================
		{SceneName: "场景4: Case 2修复 - this显式访问成员变量", LineNum: 95, TargetText: "stringValue"},
		{SceneName: "场景4: 显式访问成员变量 intValue", LineNum: 96, TargetText: "intValue"},
		{SceneName: "场景5: 静态字段前缀类 Math", LineNum: 104, TargetText: "Math"},
		{SceneName: "场景6: 深层字段访问起点 innerClass", LineNum: 118, TargetText: "innerClass"},
		{SceneName: "场景6: 深层字段访问起点 innerClass", LineNum: 118, TargetText: "innerField"},

		// ==================== 链式访问使用场景 ====================
		{SceneName: "场景7: 纯字段访问链起点 innerClass", LineNum: 134, TargetText: "innerClass"},
		{SceneName: "场景8: 混合链式调用起点 innerClass", LineNum: 147, TargetText: "innerClass"},

		// ==================== 集合和数组使用场景 ====================
		{SceneName: "场景9: Case 5修复 - 数组访问中的主体 numbers", LineNum: 161, TargetText: "numbers"},
		{SceneName: "场景9: 数组访问同行多次读取 numbers", LineNum: 163, TargetText: "numbers"},
		{SceneName: "场景10: List 集合对象方法调用 items", LineNum: 175, TargetText: "items"},
		{SceneName: "场景11: Map 集合对象包含 Key 检查 data", LineNum: 189, TargetText: "data"},

		// ==================== 复杂表达式使用场景 ====================
		{SceneName: "场景12: 嵌套算术表达式变量 c", LineNum: 205, TargetText: "c"},
		{SceneName: "场景13: 三元表达式条件中方法调用主体 str", LineNum: 219, TargetText: "str"},
		{SceneName: "场景14: 一元取反布尔变量 flag1", LineNum: 230, TargetText: "flag1"},

		// ==================== 方法调用中的使用场景 ====================
		{SceneName: "场景15: 方法参数位置运算变量 name", LineNum: 246, TargetText: "name"},
		{SceneName: "场景16: 复杂链式方法调用基础变量 list", LineNum: 259, TargetText: "list"},

		// ==================== 字符串操作使用场景 ====================
		{SceneName: "场景17: 字符串拼接整型变量 age", LineNum: 274, TargetText: "age"},
		{SceneName: "场景18: 字符串常规对象方法调用 text", LineNum: 284, TargetText: "text"},

		// ==================== 条件和循环使用场景 ====================
		{SceneName: "场景19: if 条件小括号内变量 score", LineNum: 299, TargetText: "score"},
		{SceneName: "场景20: For 循环更新段自增索引 i", LineNum: 313, TargetText: "i"},
		{SceneName: "场景20(续): 增强 For 循环被迭代容器 items", LineNum: 317, TargetText: "items"},
		{SceneName: "场景21: Switch 语句判断枢纽变量 day", LineNum: 329, TargetText: "day"},

		// ==================== Lambda和流式处理使用场景 ====================
		{SceneName: "场景22: Lambda 内部变量消费 name", LineNum: 359, TargetText: "name"},
		{SceneName: "场景23: Lambda 闭包捕获外层变量 prefix", LineNum: 373, TargetText: "prefix"},

		// ==================== 异常处理使用场景 ====================
		{SceneName: "场景24: Catch 块内异常变量 e", LineNum: 388, TargetText: "e"},
		{SceneName: "场景25: Throw 参数中字符串变量 errorMessage", LineNum: 401, TargetText: "errorMessage"},

		// ==================== 类型转换使用场景 ====================
		{SceneName: "场景26: 强转表达式被转换变量 obj", LineNum: 415, TargetText: "obj"},
		{SceneName: "场景27: Instanceof 右侧被核对实例 obj", LineNum: 426, TargetText: "obj"},

		// ==================== 泛型使用场景 ====================
		{SceneName: "场景28: 带泛型限定的 List 容器 stringList", LineNum: 445, TargetText: "stringList"},
		{SceneName: "场景29: 泛型方法体内入参 item", LineNum: 454, TargetText: "item"},
	}

	// 4. 提取实际捕获到的所有原始点位标识
	actualCapturedPoints := make(map[string]bool)
	for _, ct := range captures {
		lineNum := int(ct.Node.StartPosition().Row) + 1
		rawText := ct.Node.Utf8Text(*fCtx.SourceBytes)

		// 仅记录核心点位特征 格式 -> "行号:变量名"
		pointKey := fmt.Sprintf("%d:%s", lineNum, rawText)
		actualCapturedPoints[pointKey] = true
	}

	// 5. 单纯对是否存在捕获点进行遍历断言
	for _, exp := range expectations {
		t.Run(exp.SceneName, func(t *testing.T) {
			expectedKey := fmt.Sprintf("%d:%s", exp.LineNum, exp.TargetText)
			if !actualCapturedPoints[expectedKey] {
				t.Errorf("【捕获遗漏】JavaActionQuery 规则未能在第 %d 行捕获到标识符: %s", exp.LineNum, exp.TargetText)
			}
		})
	}
}
