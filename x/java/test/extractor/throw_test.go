package extractor

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
	"github.com/stretchr/testify/assert"
)

func TestJavaExtractor_Throw(t *testing.T) {
	// 1. 准备与提取
	testFile := test.GetTestFilePath(filepath.Join("extractor", "throw", "ThrowRelationSuite.java"))
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
		{
			sourceQN: "com.example.rel.ThrowRelationSuite.readFile",
			targetQN: "IOException",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelThrowIsSignature])
				assert.Equal(t, 0, m[constants.RelThrowIndex])
			},
		},
		{
			sourceQN: "com.example.rel.ThrowRelationSuite.readFile",
			targetQN: "SQLException",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelThrowIsSignature])
				assert.Equal(t, 1, m[constants.RelThrowIndex])
			},
		},
		{
			sourceQN: "com.example.rel.ThrowRelationSuite.readFile",
			targetQN: "RuntimeException",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				isSig, _ := m[constants.RelThrowIsSignature].(bool)
				assert.False(t, isSig)
				assert.Contains(t, m[constants.RelRawText], "throw new RuntimeException")
			},
		},
		{
			sourceQN: "com.example.rel.ThrowRelationSuite.ThrowRelationSuite", // 改掉 <init>
			targetQN: "Exception",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelThrowIsSignature])
			},
		},
		{
			sourceQN: "com.example.rel.ThrowRelationSuite.rethrow",
			targetQN: "Exception",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				// 重新抛出暂无特殊标记
			},
		},
	}

	// 3. 校验逻辑 (修正 Unused Variable 问题)
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 基础条件匹配
			if rel.Type == model.Throw &&
				rel.Target.Name == exp.targetQN &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) {

				// 如果有 checkMores，需要确保当前这个 rel 满足 mores 里的特定条件
				// (防止在 readFile 里把 IOException 错认成 SQLException)
				if exp.checkMores != nil {
					// 使用匿名测试函数进行判定
					isCurrentMatch := t.Run("SubCheck", func(st *testing.T) {
						exp.checkMores(st, rel.Mores)
					})

					if !isCurrentMatch {
						continue // 当前 rel 属性不匹配，去找下一个
					}
				}

				found = true
				break
			}
		}
		assert.True(t, found, "Missing Throw relation: [%s] %s -> %s", model.Throw, exp.sourceQN, exp.targetQN)
	}
}
