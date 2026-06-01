package collector

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

func TestJavaCollector_ScopeAndShadowing(t *testing.T) {
	// 1. 初始化解析环境
	filePath := test.GetTestFilePath(filepath.Join("collector", "var_scope", "ScopeTest.java"))
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

	// 1. 验证方法直接作用域下的变量 x (第一次出现，无后缀)
	t.Run("Verify Root Method Variable", func(t *testing.T) {
		qn := "com.example.base.ScopeTest.test().x" // 修正：移除 $1
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Root variable x not found")
		}
	})

	// 2. 验证第一个独立代码块 { int x = 2; }
	t.Run("Verify First Block Shadowing", func(t *testing.T) {
		// block 始终带 $n，但 block 内部的第一个 x 不带 $n
		qn := "com.example.base.ScopeTest.test().block$1.x"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Variable in block$1 not found")
		}
	})

	// 3. 验证 if 分支代码块 { int x = 3; }
	t.Run("Verify If-Statement Block", func(t *testing.T) {
		qn := "com.example.base.ScopeTest.test().block$2.x"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Variable in block$2 not found")
		}
	})

	// 4. 验证 Lambda 表达式及其内部变量
	t.Run("Verify Lambda Scope", func(t *testing.T) {
		// QN: com.example.base.ScopeTest.test().lambda$1.x
		lambdaVarQN := "com.example.base.ScopeTest.test().lambda$1.x"
		if len(test.FindDefinitionsByQN(fCtx, lambdaVarQN)) == 0 {
			t.Errorf("Variable x not found inside Lambda block scope")
		}

		// QN: com.example.base.ScopeTest.test().lambda$1.a
		lambdaVarA := "com.example.base.ScopeTest.test().lambda$1.a"
		if len(test.FindDefinitionsByQN(fCtx, lambdaVarA)) == 0 {
			t.Errorf("Variable a not found inside Lambda block scope")
		}

		// QN: com.example.base.ScopeTest.test().lambda$1.b
		lambdaVarB := "com.example.base.ScopeTest.test().lambda$1.b"
		if len(test.FindDefinitionsByQN(fCtx, lambdaVarB)) == 0 {
			t.Errorf("Variable b not found inside Lambda block scope")
		}

		// QN: com.example.base.ScopeTest.test().lambda$1.c
		lambdaVarC := "com.example.base.ScopeTest.test().lambda$1.c"
		if len(test.FindDefinitionsByQN(fCtx, lambdaVarC)) == 0 {
			t.Errorf("Variable c not found inside Lambda block scope")
		}
	})

	// 5. 验证 Lambda 多参数识别
	t.Run("Verify Lambda Multi-Parameters", func(t *testing.T) {
		// 情况 A: (p1, p2) -> ...
		// 注意：在你的 identifyLambdaParameter 逻辑中，这些会被识别为 Variable
		params := []string{"p1", "p2", "v1", "v2"}
		for _, p := range params {
			// 根据你的 QN 生成逻辑，这些参数属于对应的 lambda$n
			// 需要通过 printCodeElements 确认具体的 lambda 序号，这里假设是 lambda$2 和 lambda$3
			found := false
			defs, _ := fCtx.FindByShortName(p)
			for _, entry := range defs {
				if strings.Contains(entry.Element.QualifiedName, "lambda") {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Lambda parameter %s not found in any lambda scope", p)
			}
		}
	})
}

func TestJavaCollector_ScopeVariable(t *testing.T) {
	// 1. 初始化解析环境
	filePath := test.GetTestFilePath(filepath.Join("collector", "var_scope", "ScopeVariableTest.java"))
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
	// 定义期望的变量路径 (相对于包名 com.example.base.test)
	baseQN := "com.example.base.test.ScopeVariableTest.test()"

	expectedScopes := map[string]string{
		"bis":  baseQN + ".block$1.bis",
		"e":    baseQN + ".block$2.e",
		"i":    baseQN + ".block$3.i",
		"item": baseQN + ".block$4.item",
		"s":    baseQN + ".block$5.s",
		"list": baseQN + ".list", // list 是方法一级变量，不进入 block
		"obj":  baseQN + ".obj",  // obj 同上
	}

	for varName, expectedQN := range expectedScopes {
		t.Run("Verify_"+varName, func(t *testing.T) {
			entries, ok := fCtx.FindByShortName(varName)
			if !ok || len(entries) == 0 {
				t.Fatalf("Variable %s not found in definitions", varName)
			}

			// 验证 QN 是否匹配
			actualQN := entries[0].Element.QualifiedName
			if actualQN != expectedQN {
				t.Errorf("Variable %s QN mismatch:\n  Expected: %s\n  Actual:   %s",
					varName, expectedQN, actualQN)
			}

			// 验证 Kind 是否为 Variable
			if entries[0].Element.Kind != model.Variable {
				t.Errorf("Variable %s has wrong kind: %v", varName, entries[0].Element.Kind)
			}
		})
	}
}
