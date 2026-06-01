package collector

import (
	"path/filepath"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

func TestJavaCollector_AnonymousAndNested(t *testing.T) {
	// 1. 初始化解析环境
	filePath := test.GetTestFilePath(filepath.Join("collector", "anonymous", "AnonymousClassTest.java"))
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

	// 1. 验证匿名内部类 (通常 QN 带有 $1 等后缀)
	t.Run("Verify Anonymous Class", func(t *testing.T) {
		// 匿名类位于 run 方法内部
		qn := "com.example.base.test.AnonymousClassTest.run().anonymousClass$1"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Anonymous class not found at %s", qn)
		}
		if defs[0].Element.Kind != model.AnonymousClass {
			t.Errorf("Expected AnonymousClass, got %s", defs[0].Element.Kind)
		}

		// 验证匿名内部类里的方法
		methodQN := qn + ".compareTo(String)"
		if len(test.FindDefinitionsByQN(fCtx, methodQN)) == 0 {
			t.Errorf("Method compareTo not found inside anonymous class")
		}
	})

	// 2. 验证静态内部类
	t.Run("Verify Static Inner Class", func(t *testing.T) {
		qn := "com.example.base.test.AnonymousClassTest.Inner"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Inner class not found")
		}

		isStatic := defs[0].Element.Extra.Mores[java.ClassIsStatic].(bool)
		if !isStatic {
			t.Errorf("Expected Inner class to be static")
		}
	})

	// 3. 验证嵌套在内部类里的枚举
	t.Run("Verify Nested Enum in Inner Class", func(t *testing.T) {
		qn := "com.example.base.test.AnonymousClassTest.Inner.Color"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Nested enum Color not found")
		}
		if defs[0].Element.Kind != model.Enum {
			t.Errorf("Expected Enum, got %s", defs[0].Element.Kind)
		}

		// 验证枚举项
		constantQN := qn + ".RED"
		if len(test.FindDefinitionsByQN(fCtx, constantQN)) == 0 {
			t.Errorf("Enum constant RED not found")
		}
	})
}
