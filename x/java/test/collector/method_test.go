package collector

import (
	"path/filepath"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

func TestJavaCollector_MethodOverloading(t *testing.T) {
	// 1. 初始化解析环境
	filePath := test.GetTestFilePath(filepath.Join("collector", "method", "MethodTest.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 2. 执行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	test.PrintCodeElements(fCtx)

	// --- 断言开始 ---

	// 1. 验证构造函数识别
	t.Run("Verify Constructor", func(t *testing.T) {
		qn := "com.example.base.MethodTest.MethodTest()"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Constructor not found by QN: %s", qn)
		}
		elem := defs[0].Element

		if isCons, _ := elem.Extra.Mores[java.MethodIsConstructor].(bool); !isCons {
			t.Errorf("Expected MethodIsConstructor to be true")
		}
		if elem.Kind != model.Method {
			t.Errorf("Expected Kind Method, got %s", elem.Kind)
		}
	})

	// 2. 验证方法重载 (Overloading) 的 QN 唯一性
	t.Run("Verify Method Overloading QNs", func(t *testing.T) {
		overloads := []struct {
			qn       string
			expected string
		}{
			{"com.example.base.MethodTest.exec(int)", "void"},
			{"com.example.base.MethodTest.exec(String)", "String"},
			{"com.example.base.MethodTest.exec(int,String)", "void"},
		}

		for _, tc := range overloads {
			defs := test.FindDefinitionsByQN(fCtx, tc.qn)
			if len(defs) == 0 {
				t.Errorf("Method overload not found: %s", tc.qn)
				continue
			}

			// 验证返回值提取
			retType, _ := defs[0].Element.Extra.Mores[java.MethodReturnType].(string)
			if retType != tc.expected {
				t.Errorf("For %s, expected return type %s, got %s", tc.qn, tc.expected, retType)
			}
		}
	})

	// 3. 验证 FileContext 里的 SN (Short Name) 聚合
	t.Run("Verify SN Aggregation", func(t *testing.T) {
		// exec 这个名字应该对应 3 个定义
		entries, exists := fCtx.FindByShortName("exec")
		if !exists {
			t.Fatalf("SN 'exec' not found in DefinitionsBySN")
		}
		if len(entries) != 3 {
			t.Errorf("Expected 3 overloads for 'exec', found %d", len(entries))
		}

		// 验证 QN 是否各不相同
		qnSet := make(map[string]bool)
		for _, entry := range entries {
			qnSet[entry.Element.QualifiedName] = true
		}
		if len(qnSet) != 3 {
			t.Errorf("Duplicate QNs detected in SN aggregation: %v", qnSet)
		}
	})
}
