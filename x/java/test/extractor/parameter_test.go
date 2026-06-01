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

func TestJavaExtractor_Parameter(t *testing.T) {
	testFile := test.GetTestFilePath(filepath.Join("extractor", "parameter", "ParameterRelationSuite.java"))
	files := []string{testFile}

	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelations(allRelations)

	expectedRels := []struct {
		sourceQN   string
		targetQN   string
		index      int // 显式提取 Index 以便在多参数场景下精准匹配
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		// --- 1. 多参数顺序与类型 (String name) ---
		{
			sourceQN: "com.example.rel.ParameterRelationSuite.update",
			targetQN: "String",
			index:    0,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "name", m[java.RelParameterName])
				assert.Equal(t, 0, m[java.RelParameterIndex])
			},
		},
		// --- 1.1 多参数顺序与类型 (long id) ---
		{
			sourceQN: "com.example.rel.ParameterRelationSuite.update",
			targetQN: "long",
			index:    1,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "id", m[java.RelParameterName])
				assert.Equal(t, 1, m[java.RelParameterIndex])
			},
		},
		// --- 2. 可变参数 (Object... args) ---
		{
			sourceQN: "com.example.rel.ParameterRelationSuite.log",
			targetQN: "Object",
			index:    1,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelParameterIsVarargs])
				assert.Equal(t, "args", m[java.RelParameterName])
			},
		},
		// --- 3. Final 参数与注解修饰 ---
		{
			sourceQN: "com.example.rel.ParameterRelationSuite.setPath",
			targetQN: "String",
			index:    0,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "path", m[java.RelParameterName])
			},
		},
		// --- 4. 构造函数参数 ---
		{
			sourceQN: "ParameterRelationSuite", // 兼容 <init> 或 类名
			targetQN: "int",
			index:    0,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "val", m[java.RelParameterName])
			},
		},
	}

	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 匹配原则：类型为 PARAMETER + 目标类型名一致 + SourceQN 匹配 + Index 一致
			relIndex, _ := rel.Mores[java.RelParameterIndex].(int)

			if rel.Type == model.Parameter &&
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
		assert.True(t, found, "Missing Parameter relation: %s -> %s (index %d)", exp.sourceQN, exp.targetQN, exp.index)
	}
}
