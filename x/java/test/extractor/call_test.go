package extractor

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
	"github.com/stretchr/testify/assert"
)

func TestJavaExtractor_Call(t *testing.T) {
	// 1. 准备与提取
	testFile := test.GetTestFilePath(filepath.Join("extractor", "call", "CallRelationSuite.java"))
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
		sourceQN   string               // Source 节点的 QN 片段
		targetName string               // Target 节点的名称 (Short Name)
		relType    model.DependencyType // 关系类型
		value      string               // 对应 RelRawText 的精确定位
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		{
			sourceQN:   "CallRelationSuite.executeAll()",
			targetName: "simpleMethod",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "this", m[java.RelCallReceiver])
				assert.Equal(t, false, m[java.RelCallIsStatic])
			},
		},
		{
			sourceQN:   "CallRelationSuite.executeAll()",
			targetName: "staticMethod",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelCallIsStatic])
				assert.Equal(t, "CallRelationSuite", m[java.RelCallReceiverType])
			},
		},
		{
			sourceQN:   "CallRelationSuite.executeAll()",
			targetName: "currentTimeMillis",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "System", m[java.RelCallReceiver])
				assert.Equal(t, true, m[java.RelCallIsStatic])
			},
		},
		{
			sourceQN:   "CallRelationSuite.executeAll()",
			targetName: "add",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelCallIsChained])
			},
		},
		{
			sourceQN:   "CallRelationSuite.executeAll()",
			targetName: "ArrayList",
			relType:    model.Create, // 确认 Create 逻辑存在
		},
		{
			sourceQN:   "CallRelationSuite.executeAll()",
			targetName: "ArrayList",
			relType:    model.Call, // 采纳建议：同时也存在 CALL 构造函数
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelCallIsConstructor])
			},
		},
		{
			sourceQN:   "lambda$1",
			targetName: "simpleMethod",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "com.example.rel.CallRelationSuite.executeAll()", m[java.RelCallEnclosingMethod])
			},
		},
		{
			sourceQN:   "anonymousClass$1.run()",
			targetName: "simpleMethod",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "com.example.rel.CallRelationSuite.executeAll()", m[java.RelCallEnclosingMethod])
			},
		},
		{
			sourceQN:   "SubClass.SubClass()",
			targetName: "super",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "explicit_constructor_invocation", m[java.RelAstKind])
				assert.Equal(t, true, m[java.RelCallIsConstructor])
			},
		},
	}

	// 3. 执行断言
	for _, exp := range expectedRels {
		t.Run(fmt.Sprintf("%s_to_%s", exp.relType, exp.targetName), func(t *testing.T) {
			found := false
			for _, rel := range allRelations {
				if rel.Type == exp.relType &&
					strings.Contains(rel.Target.Name, exp.targetName) &&
					strings.Contains(rel.Source.QualifiedName, exp.sourceQN) {

					found = true
					if exp.checkMores != nil {
						exp.checkMores(t, rel.Mores)
					}
					break
				}
			}
			assert.True(t, found, "Missing: [%s] Source:%s -> Target:%s",
				exp.relType, exp.sourceQN, exp.targetName)
		})
	}
}
