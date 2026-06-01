package collector

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

func TestJavaCollector_ExtendAndImplement(t *testing.T) {
	// 1. 初始化解析环境
	filePath := test.GetTestFilePath(filepath.Join("collector", "extend_and_implement", "ExtendAndImplementTest.java"))
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

	// 1. 验证抽象类 BaseClass
	t.Run("Verify Abstract BaseClass", func(t *testing.T) {
		qn := "com.example.base.test.BaseClass"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("BaseClass not found")
		}
		elem := defs[0].Element

		// 验证修饰符
		if isAbs := elem.Extra.Mores[java.ClassIsAbstract].(bool); !isAbs {
			t.Errorf("Expected ClassIsAbstract to be true")
		}

		// 验证接口实现
		ifaces, _ := elem.Extra.Mores[java.ClassImplementedInterfaces].([]string)
		if len(ifaces) != 1 || ifaces[0] != "Serializable" {
			t.Errorf("Expected interface Serializable, got %v", ifaces)
		}

		// 验证注解 (存在两个注解)
		annos := elem.Extra.Annotations
		if len(annos) != 2 {
			t.Errorf("Expected 2 annotations, got %d", len(annos))
		}
	})

	// 2. 验证最终类 FinalClass
	t.Run("Verify Final FinalClass", func(t *testing.T) {
		qn := "com.example.base.test.FinalClass"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("FinalClass not found")
		}
		elem := defs[0].Element

		// 验证 Final 状态
		if isFinal := elem.Extra.Mores[java.ClassIsFinal].(bool); !isFinal {
			t.Errorf("Expected ClassIsFinal to be true")
		}

		// 验证父类继承
		super, _ := elem.Extra.Mores[java.ClassSuperClass].(string)
		if super != "BaseClass" {
			t.Errorf("Expected SuperClass BaseClass, got %s", super)
		}

		// 验证多接口实现
		ifaces, _ := elem.Extra.Mores[java.ClassImplementedInterfaces].([]string)
		expectedIfaces := []string{"Cloneable", "Runnable"}
		if len(ifaces) != 2 {
			t.Errorf("Expected 2 interfaces, got %v", ifaces)
		}
		for _, expected := range expectedIfaces {
			found := false
			for _, got := range ifaces {
				if got == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected interface %s not found in %v", expected, ifaces)
			}
		}
	})

	// 3. 验证方法重写与 Signature
	t.Run("Verify Override Method Signature", func(t *testing.T) {
		qn := "com.example.base.test.FinalClass.run()"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method run() not found")
		}
		elem := defs[0].Element

		// 验证包含 @Override 注解
		hasOverride := false
		for _, anno := range elem.Extra.Annotations {
			if strings.Contains(anno, "Override") {
				hasOverride = true
				break
			}
		}
		if !hasOverride {
			t.Errorf("Method run() should have @Override annotation")
		}
	})
}
