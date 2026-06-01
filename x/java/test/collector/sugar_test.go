package collector

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

func TestJavaCollector_DefaultConstructorTest(t *testing.T) {
	// 1. 初始化解析环境
	filePath := test.GetTestFilePath(filepath.Join("collector", "sugar", "DefaultConstructorTest.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 2. 执行 Collector (包含第4步：语法糖增强)
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	test.PrintCodeElements(fCtx)

	// --- 断言开始 ---

	// 1. 验证默认构造函数补全
	// 在测试代码中，直接检查 SN 映射
	t.Run("Verify_Implicit_Default_Constructor", func(t *testing.T) {
		shortName := "DefaultConstructor"
		qn := "com.example.sugar.DefaultConstructor.DefaultConstructor()"

		found := false
		// 显式遍历 fCtx 的定义，不要信任外部封装的 find 函数
		if entries, ok := fCtx.FindByShortName(shortName); ok {
			for _, e := range entries {
				if e.Element.QualifiedName == qn {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("Default constructor NOT found in DefinitionsBySN. QN: %s", qn)
		}
	})

	// 2. 验证 Enum 的自动方法 values()
	t.Run("Verify Enum values() Method", func(t *testing.T) {
		qn := "com.example.sugar.Color.values()"
		defs := test.FindDefinitionsByQN(fCtx, qn)

		if len(defs) == 0 {
			t.Fatalf("Implicit method values() not found for Enum Color")
		}

		elem := defs[0].Element
		expectedSig := "public static Color[] values()"
		if elem.Signature != expectedSig {
			t.Errorf("Expected signature %s, got %s", expectedSig, elem.Signature)
		}
	})

	// 3. 验证 Enum 的自动方法 valueOf(String)
	t.Run("Verify Enum valueOf() Method", func(t *testing.T) {
		qn := "com.example.sugar.Color.valueOf(String)"
		defs := test.FindDefinitionsByQN(fCtx, qn)

		if len(defs) == 0 {
			t.Fatalf("Implicit method valueOf(String) not found for Enum Color")
		}

		elem := defs[0].Element
		expectedSig := "public static Color valueOf(String name)"
		if elem.Signature != expectedSig {
			t.Errorf("Expected signature %s, got %s", expectedSig, elem.Signature)
		}
	})

	// 4. 验证原有的显式定义不受影响
	t.Run("Verify Explicit Enum Constant Still Exists", func(t *testing.T) {
		qn := "com.example.sugar.Color.RED"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Explicit enum constant RED should still exist")
		}
		if isImplicit := defs[0].Element.Extra.Mores[java.MethodIsImplicit]; isImplicit != nil {
			t.Errorf("Explicit constant RED should NOT be marked as implicit")
		}
	})
}

func TestJavaCollector_RecordSugar(t *testing.T) {
	filePath := test.GetTestFilePath(filepath.Join("collector", "sugar", "RecordTest.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, false)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	test.PrintCodeElements(fCtx)

	// 1. 验证隐式字段
	t.Run("Verify Implicit Fields", func(t *testing.T) {
		qn := "com.example.sugar.User.id"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 || defs[0].Element.Kind != model.Field {
			t.Errorf("Implicit field 'id' not found or wrong kind")
		}
	})

	// 2. 验证隐式 Accessor (id())
	t.Run("Verify Implicit Accessor id()", func(t *testing.T) {
		qn := "com.example.sugar.User.id()"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Implicit accessor id() not found")
		}
		if sig := defs[0].Element.Signature; sig != "public Long id()" {
			t.Errorf("Wrong signature for id(): %s", sig)
		}
	})

	// 3. 验证显式覆盖的方法 (name())
	t.Run("Verify Explicit Accessor name()", func(t *testing.T) {
		qn := "com.example.sugar.User.name()"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method name() not found")
		}

		// 修正：在 Record 中，"name" 既是字段也是方法，所以 SN 列表长度应该是 2
		// 我们应该验证：在该 SN 下，Method 类型的定义是否只有一个
		methodCount := 0
		var methodDef *model.CodeElement
		defs, _ = fCtx.FindByShortName("name")
		for _, d := range defs {
			if d.Element.Kind == model.Method {
				methodCount++
				methodDef = d.Element
			}
		}

		if methodCount != 1 {
			t.Errorf("Expected 1 method definition for name(), found %d", methodCount)
		}

		// 验证显式定义没有被标记为隐式
		isImp, _ := methodDef.Extra.Mores[java.MethodIsImplicit].(bool)
		if isImp {
			t.Errorf("Explicitly defined method name() should NOT be marked as implicit")
		}
	})
}

func TestJavaCollector_TryWithResources(t *testing.T) {
	// 1. 初始化解析环境
	// 假设文件路径为 com/example/sugar/TryWithResourcesTest.java
	filePath := test.GetTestFilePath(filepath.Join("collector", "sugar", "TryWithResourcesTest.java"))
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

	// 打印结果便于调试
	test.PrintCodeElements(fCtx)

	// --- 断言开始 ---
	baseQN := "com.example.sugar.TryWithResourcesTest.test()"

	// 场景 1: 验证标准定义 (input 应该在第一个 block 中)
	t.Run("Scenario_SingleResource", func(t *testing.T) {
		varName := "input"
		expectedQN := baseQN + ".block$1.input"

		entries, ok := fCtx.FindByShortName(varName)
		if !ok || len(entries) == 0 {
			t.Fatalf("Variable %s not found", varName)
		}

		actualQN := entries[0].Element.QualifiedName
		if actualQN != expectedQN {
			t.Errorf("Variable %s QN mismatch:\n  Expected: %s\n  Actual:   %s", varName, expectedQN, actualQN)
		}
	})

	// 场景 2: 验证多个资源及其唯一性 (out 和 in 应该都在第二个 block 中)
	t.Run("Scenario_MultipleResources", func(t *testing.T) {
		resources := []struct {
			name     string
			expected string
		}{
			{"out", baseQN + ".block$2.out"},
			{"in", baseQN + ".block$2.in"},
		}

		for _, res := range resources {
			entries, ok := fCtx.FindByShortName(res.name)
			if !ok || len(entries) == 0 {
				t.Errorf("Variable %s not found", res.name)
				continue
			}

			actualQN := entries[0].Element.QualifiedName
			if actualQN != res.expected {
				t.Errorf("Variable %s QN mismatch:\n  Expected: %s\n  Actual:   %s", res.name, res.expected, actualQN)
			}

			// 额外验证：父级应该是同一个 block$2
			if entries[0].ParentQN != baseQN+".block$2" {
				t.Errorf("Variable %s has wrong ParentQN: %s", res.name, entries[0].ParentQN)
			}
		}
	})

	// 验证：方法下应该只有 2 个 block
	t.Run("Verify_BlockCount", func(t *testing.T) {
		blocks, _ := fCtx.FindByShortName("block")
		// 注意：这里需要过滤出属于 test() 方法下的 block
		count := 0
		for _, b := range blocks {
			if strings.HasPrefix(b.Element.QualifiedName, baseQN) {
				count++
			}
		}
		if count != 2 {
			t.Errorf("Expected 2 blocks in test(), but found %d", count)
		}
	})
}

func TestJavaCollector_Lambda(t *testing.T) {
	filePath := test.GetTestFilePath(filepath.Join("collector", "sugar", "LambdaTest.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	test.PrintCodeElements(fCtx)

	const qnPrefix = "com.example.sugar.LambdaTest.testLambda()"

	// 1. 验证 Lambda 自身的元数据与 Signature
	t.Run("Verify Lambda Metadata", func(t *testing.T) {
		testCases := []struct {
			name            string
			qn              string
			expectedSig     string
			expectedParams  string
			expectedIsBlock bool
		}{
			{
				name:            "Inferred Multi-params",
				qn:              qnPrefix + ".lambda$1",
				expectedSig:     "(a, b) -> expr",
				expectedParams:  "(a, b)",
				expectedIsBlock: false,
			},
			{
				name:            "Single Param No Paren",
				qn:              qnPrefix + ".lambda$2",
				expectedSig:     "s -> {...}",
				expectedParams:  "s",
				expectedIsBlock: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				defs := test.FindDefinitionsByQN(fCtx, tc.qn)
				if len(defs) == 0 {
					t.Fatalf("Lambda %s not found", tc.qn)
				}
				elem := defs[0].Element

				// 验证 Signature
				if elem.Signature != tc.expectedSig {
					t.Errorf("Signature mismatch: got %v, want %v", elem.Signature, tc.expectedSig)
				}

				// 验证深度解析元数据
				mores := elem.Extra.Mores
				if mores[java.LambdaParameters] != tc.expectedParams {
					t.Errorf("Params mismatch: got %v, want %v", mores[java.LambdaParameters], tc.expectedParams)
				}
				if mores[java.LambdaBodyIsBlock] != tc.expectedIsBlock {
					t.Errorf("IsBlock mismatch: got %v, want %v", mores[java.LambdaBodyIsBlock], tc.expectedIsBlock)
				}
			})
		}
	})

	// 2. 验证 Lambda 参数变量 (归属于 lambda$n 作用域)
	t.Run("Verify Lambda Parameter Variables", func(t *testing.T) {
		paramVariables := []string{
			qnPrefix + ".lambda$1.a",
			qnPrefix + ".lambda$1.b",
			qnPrefix + ".lambda$2.s",
		}

		for _, qn := range paramVariables {
			defs := test.FindDefinitionsByQN(fCtx, qn)
			if len(defs) == 0 {
				t.Errorf("Lambda parameter variable not found: %s", qn)
			} else {
				// 验证 Kind 必须是 Variable
				if defs[0].Element.Kind != model.Variable {
					t.Errorf("Kind mismatch for %s: got %v", qn, defs[0].Element.Kind)
				}
			}
		}
	})

	// 3. 验证 Lambda 内部的局部变量 (prefix)
	t.Run("Verify Variable Inside Lambda Body", func(t *testing.T) {
		qnVar := qnPrefix + ".lambda$2.prefix"
		defs := test.FindDefinitionsByQN(fCtx, qnVar)
		if len(defs) == 0 {
			t.Errorf("Variable 'prefix' inside lambda body not found: %s", qnVar)
			return
		}

		// 验证它确实被标记为 Lambda 作用域内的变量
		elem := defs[0].Element
		if elem.Name != "prefix" {
			t.Errorf("Name mismatch: got %v", elem.Name)
		}
	})
}

func TestJavaCollector_MethodReference(t *testing.T) {
	// 1. 加载测试文件
	filePath := test.GetTestFilePath(filepath.Join("collector", "sugar", "MethodRefTest.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// 2. 执行采集
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	// 打印结果便于调试
	test.PrintCodeElements(fCtx)

	// 3. 定义全覆盖断言矩阵
	const qnPrefix = "com.example.sugar.MethodRefTest.testAllMethodReferences()"

	testCases := []struct {
		name             string
		expectedQN       string
		expectedSig      string
		expectedReceiver string // 新增：预期的 Receiver
		expectedTarget   string // 新增：预期的 Target (方法名或 new)
	}{
		{
			name:             "Static Method Reference",
			expectedQN:       qnPrefix + ".method_ref$1",
			expectedSig:      "Integer::parseInt",
			expectedReceiver: "Integer",
			expectedTarget:   "parseInt",
		},
		{
			name:             "Bound Instance Method Reference",
			expectedQN:       qnPrefix + ".method_ref$2",
			expectedSig:      "System.out::println",
			expectedReceiver: "System.out",
			expectedTarget:   "println",
		},
		{
			name:             "Arbitrary Instance Method Reference",
			expectedQN:       qnPrefix + ".method_ref$3",
			expectedSig:      "String::toLowerCase",
			expectedReceiver: "String",
			expectedTarget:   "toLowerCase",
		},
		{
			name:             "Constructor Reference",
			expectedQN:       qnPrefix + ".method_ref$4",
			expectedSig:      "ArrayList::new",
			expectedReceiver: "ArrayList",
			expectedTarget:   "new",
		},
		{
			name:             "Array Constructor Reference",
			expectedQN:       qnPrefix + ".method_ref$5",
			expectedSig:      "int[]::new",
			expectedReceiver: "int[]",
			expectedTarget:   "new",
		},
		{
			name:             "Generic Method Reference",
			expectedQN:       qnPrefix + ".method_ref$6",
			expectedSig:      "this::<String>genericMethod",
			expectedReceiver: "this",
			expectedTarget:   "genericMethod",
		},
	}

	// 4. 执行循环断言
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defs := test.FindDefinitionsByQN(fCtx, tc.expectedQN)

			if len(defs) == 0 {
				t.Errorf("Method reference definition not found: %s", tc.expectedQN)
				return
			}

			entry := defs[0]
			elem := entry.Element

			// 验证 Kind
			if elem.Kind != model.MethodRef {
				t.Errorf("Kind mismatch: got %v, want %v", elem.Kind, model.MethodRef)
			}

			// 验证 Signature
			if elem.Signature != tc.expectedSig {
				t.Errorf("Signature mismatch: got %s, want %s", elem.Signature, tc.expectedSig)
			}

			// 5. 验证深度解析的元数据 (Mores)
			if elem.Extra == nil || elem.Extra.Mores == nil {
				t.Errorf("Extra.Mores is nil for %s", tc.name)
				return
			}

			actualReceiver := elem.Extra.Mores[java.MethodRefReceiver]
			actualTarget := elem.Extra.Mores[java.MethodRefTarget]

			if actualReceiver != tc.expectedReceiver {
				t.Errorf("Receiver mismatch: got %v, want %v", actualReceiver, tc.expectedReceiver)
			}
			if actualTarget != tc.expectedTarget {
				t.Errorf("Target mismatch: got %v, want %v", actualTarget, tc.expectedTarget)
			}
		})
	}
}
