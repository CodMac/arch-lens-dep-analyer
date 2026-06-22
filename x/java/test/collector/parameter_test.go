package collector

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

func TestJavaCollector_ParameterScope(t *testing.T) {
	// 1. 初始化解析环境
	filePath := test.GetTestFilePath(filepath.Join("collector", "parameter", "ParameterScopeTest.java"))
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

	// 1. 验证构造函数参数
	t.Run("Verify Constructor Parameter", func(t *testing.T) {
		// 注意 QN 路径：类 -> 构造函数 -> 参数
		qn := "com.example.base.test.ParameterScopeTest.ParameterScopeTester(String).initialConfig"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Constructor parameter 'initialConfig' not found at %s", qn)
		}

		elem := defs[0].Element
		if elem.Kind != model.Variable {
			t.Errorf("Expected Kind Variable, got %s", elem.Kind)
		}
		if isParam := elem.Extra.Mores[constants.VariableIsParam].(bool); !isParam {
			t.Error("Expected VariableIsParam to be true")
		}
	})

	// 2. 验证变长参数 (Varargs)
	t.Run("Verify Varargs Parameter", func(t *testing.T) {
		qn := "com.example.base.test.ParameterScopeTest.execute(int,String...).labels"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Varargs parameter 'labels' not found")
		}

		vType := defs[0].Element.Extra.Mores[constants.VariableRawType].(string)
		// 验证你的 extractTypeString 是否正确处理了 "..."
		if !strings.Contains(vType, "...") {
			t.Errorf("Expected type with '...', got %s", vType)
		}
	})

	// 3. 验证内部类方法参数的作用域层级
	t.Run("Verify Inner Class Method Parameter", func(t *testing.T) {
		qn := "com.example.base.test.ParameterScopeTest.InnerWorker.doWork(long).duration"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Inner class method parameter 'duration' not found")
		}

		// 验证路径层级是否包含 InnerWorker
		if !strings.Contains(defs[0].Element.QualifiedName, "InnerWorker") {
			t.Errorf("Parameter QN missing inner class scope: %s", defs[0].Element.QualifiedName)
		}

		// 验证 Signature 格式: long duration
		if !strings.Contains(defs[0].Element.Signature, "long duration") {
			t.Errorf("Invalid signature: %s", defs[0].Element.Signature)
		}
	})
}
