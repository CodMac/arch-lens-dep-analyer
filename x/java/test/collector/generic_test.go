package collector

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

func TestJavaCollector_GenericComplex(t *testing.T) {
	// 1. 初始化解析环境
	filePath := test.GetTestFilePath(filepath.Join("collector", "generic", "GenericTest.java"))
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

	// 1. 验证接口泛型边界 (Intersection Type: Serializable & Cloneable)
	t.Run("Verify Interface Generic Bounds", func(t *testing.T) {
		qn := "com.example.base.test.GenericTest"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Interface GenericTest not found")
		}
		elem := defs[0].Element

		// 验证 Signature 是否保留了泛型边界
		// 预期: interface GenericTest<T extends Serializable & Cloneable>
		if !strings.Contains(elem.Signature, "<T extends Serializable & Cloneable>") {
			t.Errorf("Signature missing intersection type bounds: %s", elem.Signature)
		}
	})

	// 2. 验证复杂的方法参数与返回类型 (List<? extends T>)
	t.Run("Verify Complex Wildcard Method", func(t *testing.T) {
		// 注意：QN 构建时会提取参数类型，并移除泛型部分以保证稳定性
		qn := "com.example.base.test.GenericTest.findAllByCriteria(List)"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method findAllByCriteria not found")
		}
		elem := defs[0].Element

		// 验证返回值 (MethodReturnType)
		retType, _ := elem.Extra.Mores[java.MethodReturnType].(string)
		if retType != "List<? extends T>" {
			t.Errorf("Expected return type List<? extends T>, got %s", retType)
		}

		// 验证原始参数列表 (MethodParameters)
		params, _ := elem.Extra.Mores[java.MethodParameters].([]string)
		if len(params) == 0 || params[0] != "List<? super T> criteria" {
			t.Errorf("Expected param List<? super T> criteria, got %v", params)
		}
	})

	// 3. 验证方法级泛型与异常声明 (throws E)
	t.Run("Verify Method-level Generics and Throws", func(t *testing.T) {
		qn := "com.example.base.test.GenericTest.executeOrThrow(E)"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method executeOrThrow not found")
		}
		elem := defs[0].Element

		// 验证 Signature 包含泛型定义 <E extends Exception>
		if !strings.Contains(elem.Signature, "<E extends Exception>") {
			t.Errorf("Signature missing method-level generic: %s", elem.Signature)
		}

		// 验证 Throws 元数据
		throws, _ := elem.Extra.Mores[java.MethodThrowsTypes].([]string)
		if len(throws) == 0 || throws[0] != "E" {
			t.Errorf("Expected throws E, got %v", throws)
		}
	})
}
