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

func TestJavaExtractor_Return(t *testing.T) {
	// 1. 准备与提取
	testFile := test.GetTestFilePath(filepath.Join("extractor", "return", "ReturnRelationSuite.java"))
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
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		// --- 1. 对象返回 ---
		{
			sourceQN: "com.example.rel.ReturnRelationSuite.getName",
			targetQN: "String",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				// 默认不标记 is_primitive 时，Extractor 应根据类型识别并填充
				assert.Equal(t, false, m[java.RelReturnIsPrimitive])
			},
		},
		// --- 2. 数组返回 ---
		{
			sourceQN: "com.example.rel.ReturnRelationSuite.getBuffer",
			targetQN: "byte",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelReturnIsArray])
				assert.Equal(t, true, m[java.RelReturnIsPrimitive])
			},
		},
		// --- 3. 泛型复合返回 ---
		{
			sourceQN:   "com.example.rel.ReturnRelationSuite.getValues",
			targetQN:   "List",
			checkMores: func(t *testing.T, m map[string]interface{}) {},
		},
		// --- 4. 基础类型返回 ---
		{
			sourceQN: "com.example.rel.ReturnRelationSuite.getAge",
			targetQN: "int",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelReturnIsPrimitive])
			},
		},
		// --- 5. 嵌套数组返回 ---
		{
			sourceQN: "com.example.rel.ReturnRelationSuite.getMatrix",
			targetQN: "double",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelReturnIsArray])
			},
		},
	}

	// 3. 执行匹配断言
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			if rel.Type == model.Return &&
				rel.Target.Name == exp.targetQN &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) {

				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing Return relation: %s -> %s", exp.sourceQN, exp.targetQN)
	}
}
