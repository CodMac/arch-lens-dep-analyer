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

func TestJavaExtractor_TypeArg(t *testing.T) {
	// 1. 准备与提取
	testFile := test.GetTestFilePath(filepath.Join("extractor", "type_arg", "TypeArgRelationSuite.java"))

	files := []string{testFile}
	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelations(allRelations)

	// 2. 定义断言数据集
	expectedRels := []struct {
		sourceQN   string
		targetQN   string
		index      int
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		// --- 1. 基础多泛型 (Map<String, Integer>) ---
		{
			sourceQN: "com.example.rel.TypeArgRelationSuite.map",
			targetQN: "String",
			index:    0,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, 0, m[java.RelTypeArgIndex])
				assert.Equal(t, "type_arguments", m[java.RelAstKind])
			},
		},
		{
			sourceQN: "com.example.rel.TypeArgRelationSuite.map",
			targetQN: "Integer",
			index:    1,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, 1, m[java.RelTypeArgIndex])
			},
		},

		// --- 2. 嵌套泛型 (List<Map<String, Object>>) ---
		{
			sourceQN: "com.example.rel.TypeArgRelationSuite.complexList",
			targetQN: "Map",
			index:    0,
		},
		{
			sourceQN: "com.example.rel.TypeArgRelationSuite.complexList",
			targetQN: "Object",
			index:    1, // 对应 Map<String, Object> 的第二个参数
		},

		// --- 3. 上界通配符 (? extends Serializable) ---
		{
			// 使用方法名和参数名片段，兼容 "process(List).input"
			sourceQN: "TypeArgRelationSuite.process",
			targetQN: "Serializable",
			index:    0,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Contains(t, m[java.RelRawText], "? extends Serializable")
			},
		},

		// --- 4. 构造函数泛型实参 (new ArrayList<String>) ---
		{
			sourceQN: "TypeArgRelationSuite.process",
			targetQN: "String",
			index:    0,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "type_arguments", m[java.RelAstKind])
			},
		},

		// --- 5. 下界通配符 (? super Integer) ---
		{
			sourceQN: "TypeArgRelationSuite.addNumbers",
			targetQN: "Integer",
			index:    0,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Contains(t, m[java.RelRawText], "? super Integer")
			},
		},
	}

	// 3. 执行匹配断言
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 获取实际的 Index
			relIndex, _ := rel.Mores[java.RelTypeArgIndex].(int)

			// 匹配原则：类型为 TYPE_ARG + 目标类名一致 + SourceQN 包含关键词 + Index 一致
			if rel.Type == model.TypeArg &&
				rel.Target.Name == exp.targetQN &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) &&
				relIndex == exp.index {

				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing TypeArg: %s -> %s (index %d)", exp.sourceQN, exp.targetQN, exp.index)
	}
}
