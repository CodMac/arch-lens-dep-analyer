package collector

import (
	"path/filepath"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

func TestJavaCollector_NestedAndStaticBlocks(t *testing.T) {
	filePath := test.GetTestFilePath(filepath.Join("collector", "class", "OuterClass.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	test.PrintCodeElements(fCtx)

	// 验证 1: 静态初始化块与实例块
	t.Run("Verify Initialization Blocks", func(t *testing.T) {
		// 静态块通常被识别为 static_initializer 节点, 我们将其命名为 $static
		staticBlockQN := "com.example.base.OuterClass.$static$1"
		if len(test.FindDefinitionsByQN(fCtx, staticBlockQN)) == 0 {
			t.Errorf("Static initializer block not found at expected QN: %s", staticBlockQN)
		}
	})

	// 验证 2: 内部类与静态嵌套类
	t.Run("Verify Nested Classes", func(t *testing.T) {
		// 内部类 QN
		innerQN := "com.example.base.OuterClass.InnerClass"
		if len(test.FindDefinitionsByQN(fCtx, innerQN)) == 0 {
			t.Errorf("InnerClass not found")
		}

		// 静态嵌套类方法 QN
		nestedMethodQN := "com.example.base.OuterClass.StaticNestedClass.run()"
		if len(test.FindDefinitionsByQN(fCtx, nestedMethodQN)) == 0 {
			t.Errorf("Method run() in StaticNestedClass not found")
		}
	})

	// 验证 3: 方法内部类 (Local Class)
	t.Run("Verify Local Class", func(t *testing.T) {
		// 注意层级：OuterClass -> scopeTest() -> LocalClass
		localClassQN := "com.example.base.OuterClass.scopeTest().LocalClass"
		defs := test.FindDefinitionsByQN(fCtx, localClassQN)
		if len(defs) == 0 {
			t.Errorf("Local class inside method not found at: %s", localClassQN)
		}
	})
}
