package capture_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

// AssignPointExpectation 包含定位 Assign 捕获点所需的核心元数据
type AssignPointExpectation struct {
	SceneName  string
	LineNum    int
	TargetText string
}

func TestCapture_AssignCase(t *testing.T) {
	// 1. 获取打样文件的 AST 树上下文 (非侵入式加载)
	testFile := test.GetTestFilePath(filepath.Join("capture", "AssignCase.java"))
	gCtx := test.RunPhase1Collection(t, []string{testFile})
	fCtx := gCtx.FileContexts[testFile]

	// 2. 初始化核心规则与待测 Extractor
	captures, err := java.NewJavaExtractor().GetCaptures(fCtx)
	if err != nil {
		t.Fatal(err)
	}

	// 3. 构造 AssignCase 下预期的 Assign 捕获点点位
	expectations := []AssignPointExpectation{
		// ==================== 简单赋值场景 ====================
		{SceneName: "场景1: 简单变量赋值 a", LineNum: 59, TargetText: "a"},
		{SceneName: "场景1: 简单变量赋值 s", LineNum: 60, TargetText: "s"},
		{SceneName: "场景1: 简单变量赋值 flag", LineNum: 61, TargetText: "flag"},

		{SceneName: "场景2: 简单字段赋值 stringValue", LineNum: 69, TargetText: "stringValue"},
		{SceneName: "场景2: 简单字段赋值 intValue", LineNum: 70, TargetText: "intValue"},
		{SceneName: "场景2: 简单字段赋值 booleanValue", LineNum: 71, TargetText: "booleanValue"},

		{SceneName: "场景3: 静态字段赋值 staticCount", LineNum: 79, TargetText: "staticCount"},

		{SceneName: "场景4: 方法调用结果赋值 result", LineNum: 87, TargetText: "result"},
		{SceneName: "场景4: 方法调用结果赋值 count", LineNum: 88, TargetText: "count"},
		{SceneName: "场景4: 方法调用结果赋值 upper", LineNum: 89, TargetText: "upper"},

		// ==================== 复合赋值场景 ====================
		{SceneName: "场景5: 复合加法赋值 a", LineNum: 102, TargetText: "a"},
		{SceneName: "场景5: 复合减法赋值 a", LineNum: 103, TargetText: "a"},
		{SceneName: "场景5: 复合乘法赋值 a", LineNum: 104, TargetText: "a"},
		{SceneName: "场景5: 复合除法赋值 a", LineNum: 105, TargetText: "a"},
		{SceneName: "场景5: 复合取模赋值 a", LineNum: 106, TargetText: "a"},

		{SceneName: "场景6: 位运算与赋值 a", LineNum: 117, TargetText: "a"},
		{SceneName: "场景6: 位运算或赋值 a", LineNum: 118, TargetText: "a"},
		{SceneName: "场景6: 位运算异或赋值 a", LineNum: 119, TargetText: "a"},
		{SceneName: "场景6: 位运算左移赋值 a", LineNum: 120, TargetText: "a"},
		{SceneName: "场景6: 位运算右移赋值 a", LineNum: 121, TargetText: "a"},
		{SceneName: "场景6: 位运算无符号右移赋值 a", LineNum: 122, TargetText: "a"},

		{SceneName: "场景7: 字符串拼接赋值 message", LineNum: 132, TargetText: "message"},
		{SceneName: "场景7: 字符串拼接追加 message", LineNum: 133, TargetText: "message"},

		// ==================== 自增自减场景 ====================
		{SceneName: "场景8: 后置自增 counter", LineNum: 145, TargetText: "counter"},
		{SceneName: "场景8: 前置自增 counter", LineNum: 148, TargetText: "counter"},

		{SceneName: "场景9: 后置自减 counter", LineNum: 158, TargetText: "counter"},
		{SceneName: "场景9: 前置自减 counter", LineNum: 161, TargetText: "counter"},

		{SceneName: "场景10: 字段后置自增 intValue", LineNum: 171, TargetText: "intValue"},
		{SceneName: "场景10: 字段前置自增 intValue", LineNum: 172, TargetText: "intValue"},
		{SceneName: "场景10: 字段后置自减 intValue", LineNum: 173, TargetText: "intValue"},
		{SceneName: "场景10: 字段前置自减 intValue", LineNum: 174, TargetText: "intValue"},

		// ==================== 声明时初始化场景 ====================
		{SceneName: "场景11: 声明初始化 age", LineNum: 184, TargetText: "age"},
		{SceneName: "场景11: 声明初始化 name", LineNum: 185, TargetText: "name"},
		{SceneName: "场景11: 声明初始化 active", LineNum: 186, TargetText: "active"},

		{SceneName: "场景12: 局部字段初始化 localValue", LineNum: 194, TargetText: "localValue"},
		{SceneName: "场景12: 局部字段初始化 localStr", LineNum: 195, TargetText: "localStr"},

		{SceneName: "场景13: 集合初始化 list", LineNum: 203, TargetText: "list"},
		{SceneName: "场景13: 集合初始化 map", LineNum: 204, TargetText: "map"},

		// ==================== 数组和集合元素赋值场景 ====================
		{SceneName: "场景14: 数组索引0赋值 numbers", LineNum: 217, TargetText: "numbers"},
		{SceneName: "场景14: 数组索引1赋值 strings", LineNum: 219, TargetText: "strings"},

		{SceneName: "场景15: 集合元素赋值 list", LineNum: 227, TargetText: "list"},
		{SceneName: "场景15: 集合元素赋值 map", LineNum: 228, TargetText: "map"},

		{SceneName: "场景16: 数组元素复合加赋值 numbers", LineNum: 246, TargetText: "numbers"},
		{SceneName: "场景16: 数组元素复合乘赋值 numbers", LineNum: 247, TargetText: "numbers"},

		// ==================== 链式字段赋值场景 ====================
		{SceneName: "场景17: 深层字段赋值 innerString", LineNum: 261, TargetText: "innerString"},
		{SceneName: "场景17: 嵌套字段赋值 nestedField", LineNum: 262, TargetText: "nestedField"},

		{SceneName: "场景18: 链式方法修改状态 innerClass", LineNum: 270, TargetText: "innerClass"},
		{SceneName: "场景18: 链式方法修改状态 nestedClass", LineNum: 271, TargetText: "nestedClass"},

		{SceneName: "场景19: 复杂链式字段赋值 innerString", LineNum: 285, TargetText: "innerString"},

		// ==================== 条件和循环中的赋值场景 ====================
		{SceneName: "场景20: 条件表达式赋值 max", LineNum: 299, TargetText: "max"},
		{SceneName: "场景21: if分支赋值 grade", LineNum: 312, TargetText: "grade"},
		{SceneName: "场景21: else if分支赋值 grade", LineNum: 314, TargetText: "grade"},

		{SceneName: "场景22: for循环定义索引 i", LineNum: 328, TargetText: "i"},
		{SceneName: "场景22: for循环内部加累加 sum", LineNum: 329, TargetText: "sum"},

		{SceneName: "场景23: foreach循环容器弹射 item", LineNum: 343, TargetText: "item"},

		// ==================== 方法参数中的赋值场景 ====================
		{SceneName: "场景24: 参数构造返回赋值 username", LineNum: 358, TargetText: "username"},
		{SceneName: "场景25: 复杂算术源赋值 result1", LineNum: 371, TargetText: "result1"},

		// ==================== try-catch中的赋值场景 ====================
		{SceneName: "场景26: try块内赋值 message", LineNum: 386, TargetText: "message"},
		{SceneName: "场景26: catch块内赋值 errorCode", LineNum: 390, TargetText: "errorCode"},
		{SceneName: "场景27: finally块内状态更替 status", LineNum: 406, TargetText: "status"},

		// ==================== Lambda和闭包赋值场景 ====================
		{SceneName: "场景28: Lambda外层常量定义 multiplier", LineNum: 420, TargetText: "multiplier"},

		{SceneName: "场景29: 闭包容器数组初始化 counter", LineNum: 433, TargetText: "counter"},
		{SceneName: "场景29: 闭包容器数组初始化 counter", LineNum: 440, TargetText: "counter"},

		// ==================== 嵌套和递归赋值场景 ====================
		{SceneName: "场景30: 串联嵌套赋值 a", LineNum: 454, TargetText: "a"},
		{SceneName: "场景30: 串联嵌套赋值 b", LineNum: 454, TargetText: "b"},
		{SceneName: "场景30: 串联嵌套赋值 c", LineNum: 454, TargetText: "c"},
		{SceneName: "场景31: 递归归纳赋值 result", LineNum: 466, TargetText: "result"},

		// ==================== 多重赋值场景 ====================
		{SceneName: "场景32: 多次重定向赋值 counter", LineNum: 479, TargetText: "counter"},
		{SceneName: "场景33: 复合条件分发结果 result1", LineNum: 496, TargetText: "result1"},
		{SceneName: "场景33: 复合条件分发结果 result1", LineNum: 500, TargetText: "result1"},

		// ==================== Switch语句中的赋值场景 ====================
		{SceneName: "场景35: switch传统分支赋值 dayName", LineNum: 534, TargetText: "dayName"},
		{SceneName: "场景36: switch新特性表达式赋值 grade", LineNum: 552, TargetText: "grade"},
	}

	// 4. 提取实际捕获到的所有原始点位标识
	actualCapturedPoints := make(map[string]bool)
	for _, ct := range captures {
		// 过滤非 assign 相关的捕获目标，收窄断言噪声
		if ct.CapName != "assign_target" && ct.CapName != "id_atom" {
			continue
		}
		lineNum := int(ct.Node.StartPosition().Row) + 1
		rawText := ct.Node.Utf8Text(*fCtx.SourceBytes)

		// 仅记录核心点位特征 格式 -> "行号:变量名/字段名"
		pointKey := fmt.Sprintf("%d:%s", lineNum, rawText)
		actualCapturedPoints[pointKey] = true
	}

	// 5. 单纯对是否存在捕获点进行遍历断言
	for _, exp := range expectations {
		t.Run(exp.SceneName, func(t *testing.T) {
			expectedKey := fmt.Sprintf("%d:%s", exp.LineNum, exp.TargetText)
			if !actualCapturedPoints[expectedKey] {
				t.Errorf("【赋值捕获遗漏】JavaActionQuery 规则未能在第 %d 行捕获到赋值目标: %s", exp.LineNum, exp.TargetText)
			}
		})
	}
}
