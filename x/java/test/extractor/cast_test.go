package extractor

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
	"github.com/stretchr/testify/assert"
)

func TestJavaExtractor_Cast(t *testing.T) {
	testFile := test.GetTestFilePath(filepath.Join("extractor", "cast", "CastRelationSuite.java"))
	files := []string{testFile}

	// 1. 运行提取
	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Cast})

	// 定义公共的 Source 方法名
	sourceMethodQN := "com.example.rel.CastRelationSuite.testCastCases(Object)"

	expectedRels := []struct {
		caseDesc   string // 用例描述
		targetName string // 期望的 Target Qualified Name 或 Name
		targetKind model.ElementKind
		astKind    string // 期望的 Mores["java.rel.ast_kind"]，如果为空则不检查或接受 cast_expression
	}{
		// 1. 基础对象向下转型: (String) input
		{
			caseDesc:   "Case 1: Downcasting to String",
			targetName: "String",
			targetKind: model.Class, // 通常 JDK 类被视为 Class
			astKind:    "cast_expression",
		},
		// 2. 基础数据类型转换: (int) pi
		{
			caseDesc:   "Case 2: Primitive cast to int",
			targetName: "int",
			targetKind: model.Class, // 或者 model.Type，取决于你的模型如何处理基本类型
			astKind:    "cast_expression",
		},
		// 3. 泛型集合转型: (List<String>) input
		{
			caseDesc:   "Case 3: Generic Collection cast to List",
			targetName: "java.util.List",
			targetKind: model.Class, // List 是接口
			astKind:    "cast_expression",
		},
		// 4. 链式调用中的转型: ((SubClass) input)
		{
			caseDesc:   "Case 4: Inline cast to SubClass",
			targetName: "com.example.rel.CastRelationSuite.SubClass",
			targetKind: model.Class,
			astKind:    "cast_expression",
		},
		// 5. 模式匹配转型: instanceof String str
		{
			caseDesc:   "Case 5: Pattern Matching instanceof String",
			targetName: "String",
			targetKind: model.Class,
			astKind:    "instanceof_expression", // 必须明确区分这是 instanceof
		},
		// 6. 多重转型: (Object) input
		{
			caseDesc:   "Case 6a: Double cast to Object",
			targetName: "Object",
			targetKind: model.Class,
			astKind:    "cast_expression",
		},
		// 6. 多重转型: (Runnable) ...
		{
			caseDesc:   "Case 6b: Double cast to Runnable",
			targetName: "Runnable",
			targetKind: model.Class,
			astKind:    "cast_expression",
		},
		// 6. 多重转型: (Runnable) ...
		{
			caseDesc:   "Case 7: QN",
			targetName: "java.util.List",
			targetKind: model.Class,
			astKind:    "cast_expression",
		},
	}

	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 1. 检查关系类型
			if rel.Type != model.Cast { // 确保你已经在 model 中定义了 Cast 常量
				continue
			}

			// 2. 检查 Source (必须是 testCastCases 方法)
			if rel.Source.QualifiedName != sourceMethodQN {
				continue
			}

			// 3. 检查 Target Name (后缀匹配或全匹配)
			// 如果提取器能解析完整包名，优先用 QualifiedName；否则可以用 Name
			if !strings.HasSuffix(rel.Target.QualifiedName, exp.targetName) && rel.Target.Name != exp.targetName {
				continue
			}

			// 4. 检查 AST Kind (用于区分 (String) 和 instanceof String)
			if exp.astKind != "" {
				if val, ok := rel.Mores["java.rel.ast_kind"]; ok {
					if valStr, ok := val.(string); ok {
						if valStr != exp.astKind {
							continue // AST 类型不匹配（例如我们要 instanceof 但找到了 cast）
						}
					}
				}
			}

			// 找到匹配项
			found = true

			// 验证 Target Kind
			assert.Equal(t, exp.targetKind, rel.Target.Kind, "[%s] Target Kind mismatch", exp.caseDesc)

			// 验证 raw_text (可选，稍微检查一下是否存在)
			// assert.NotEmpty(t, rel.Mores["java.rel.raw_text"], "[%s] Should have raw text", exp.caseDesc)

			break
		}
		assert.True(t, found, "Missing expected Cast relation: [%s] -> %s (AST: %s)",
			exp.caseDesc, exp.targetName, exp.astKind)
	}
}
