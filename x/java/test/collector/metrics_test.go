package collector

import (
	"path/filepath"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

func TestCollector_Metrics_LOC(t *testing.T) {
	// 1. 定义全覆盖断言矩阵
	testCases := []struct {
		name     string
		fileName string
		expected int
	}{
		{
			name:     "纯代码无注释",
			fileName: "PureCode.java",
			expected: 6,
		},
		{
			name:     "包含空行和单行注释",
			fileName: "SingleLine.java",
			expected: 6,
		},
		{
			name:     "包含单行和多行块注释(JavaDoc)",
			fileName: "BlockComment.java",
			expected: 7,
		},
		{
			name:     "复杂的注释穿插",
			fileName: "Complex.java",
			expected: 6,
		},
	}

	collector := java.NewJavaCollector()

	// 2. 执行循环断言
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 1) 构建目标文件路径并解析 AST
			filePath := test.GetTestFilePath(filepath.Join("collector", "metrics", "loc", tc.fileName))
			rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
			if err != nil {
				t.Fatalf("Parse error for %s: %v", tc.fileName, err)
			}

			// 2) 执行核心采集生命周期
			fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
			if err != nil {
				t.Fatalf("CollectDefinitions failed for %s: %v", tc.fileName, err)
			}

			// 3) 从 Context 中查找 File 类型的 Definition
			fileDefs, ok := fCtx.FindByElementKind(model.File)
			if !ok || len(fileDefs) == 0 {
				t.Fatalf("File definition not found in context for %s", tc.fileName)
			}

			fileElem := fileDefs[0].Element

			// 验证 File 基础元数据结构
			if fileElem.Extra == nil || fileElem.Extra.Mores == nil {
				t.Fatalf("Extra.Mores is nil for File element in %s", tc.fileName)
			}

			// 4) 提取 LOC 指标并验证
			actualLOCRaw, exists := fileElem.Extra.Mores[constants.FileLOC] // 使用你定义的常量 FileLOC
			if !exists {
				t.Fatalf("Metric '%s' not found in File Extra.Mores for %s", constants.FileLOC, tc.fileName)
			}

			// 类型断言为 int
			actualLOC, ok := actualLOCRaw.(int)
			if !ok {
				t.Fatalf("Expected LOC to be of type int, got %T", actualLOCRaw)
			}

			// 断言代码行数是否符合预期
			if actualLOC != tc.expected {
				t.Errorf("LOC metric mismatch for %q: got %d, want %d", tc.fileName, actualLOC, tc.expected)
			}
		})
	}
}

func TestJavaCollector_Metrics_Complexity(t *testing.T) {
	filePath := test.GetTestFilePath(filepath.Join("collector", "metrics", "complexity", "ComplexityTest.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	testCases := []struct {
		methodQN string
		expected int
	}{
		{"com.example.metrics.ComplexityTest.simple()", 1},
		{"com.example.metrics.ComplexityTest.medium(int)", 4},
		{"com.example.metrics.ComplexityTest.complexSwitch(int)", 4},
		{"com.example.metrics.ComplexityTest.withTryCatch()", 3},
		{"com.example.metrics.ComplexityTest.ternary(int)", 2},
		{"com.example.metrics.ComplexityTest.arrowSwitch(int)", 4},
	}

	for _, tc := range testCases {
		t.Run(tc.methodQN, func(t *testing.T) {
			defs := test.FindDefinitionsByQN(fCtx, tc.methodQN)
			if len(defs) == 0 {
				t.Fatalf("Method %s not found", tc.methodQN)
			}

			complexity := 0
			if val, ok := defs[0].Element.Extra.Mores[constants.MethodComplexity].(int); ok {
				complexity = val
			} else if val, ok := defs[0].Element.Extra.Mores[constants.MethodComplexity].(float64); ok {
				complexity = int(val)
			} else {
				t.Errorf("Complexity metric missing or invalid type for %s", tc.methodQN)
			}

			if complexity != tc.expected {
				t.Errorf("Expected complexity %d, got %d", tc.expected, complexity)
			}
		})
	}
}
