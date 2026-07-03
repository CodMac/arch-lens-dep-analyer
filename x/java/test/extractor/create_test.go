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

func TestJavaExtractor_Create(t *testing.T) {
	// 1. 准备与提取
	testFile := test.GetTestFilePath(filepath.Join("extractor", "create", "CreateRelationSuite.java"))
	files := []string{testFile}
	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Create})

	// 预定义基础路径
	baseQN := "com.example.rel.CreateRelationSuite"
	methodQN := baseQN + ".testCreateCases()"

	// 2. 定义断言数据集 (使用 Collector 生成的完整 QN)
	expectedRels := []struct {
		sourceQN   string
		targetQN   string // 实例化的类全限定名
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		// --- 1. 成员变量声明时实例化 (有 Import，保持全称) ---
		{
			sourceQN: baseQN + ".fieldInstance",
			targetQN: "java.util.ArrayList",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "fieldInstance", m[constants.RelCreateVariableName])
			},
		},
		// --- 2. 静态成员变量实例化 ---
		{
			sourceQN: baseQN + ".staticMap",
			targetQN: "java.util.HashMap",
			checkMores: func(t *testing.T, m map[string]interface{}) {
			},
		},
		// --- 3. 局部变量实例化 (无 Import，保持简写) ---
		{
			sourceQN: methodQN,
			targetQN: "StringBuilder", // 调整：不带 java.lang 前缀
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "sb", m[constants.RelCreateVariableName])
				assert.Equal(t, "object_creation_expression", m[constants.RelNodeAstKind])
			},
		},
		// --- 4. 匿名内部类 (无 Import，保持简写) ---
		{
			sourceQN: methodQN,
			targetQN: "Runnable", // 调整
			checkMores: func(t *testing.T, m map[string]interface{}) {
			},
		},
		// --- 5. 数组实例化 (Fix: 真实的 AST 类型 + 简写 QN) ---
		{
			sourceQN: methodQN,
			targetQN: "String", // 调整
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelCreateIsArray])
				assert.Equal(t, "array_creation_expression", m[constants.RelNodeAstKind])
			},
		},
		// --- 6. 链式调用中的实例化 ---
		{
			sourceQN: methodQN,
			targetQN: baseQN,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "object_creation_expression", m[constants.RelNodeAstKind])
			},
		},
		// --- 7. super 调用 (super 关键字保持原样) ---
		{
			sourceQN: baseQN + ".CreateRelationSuite()",
			targetQN: "Object", // 调整：super() 对应的类符号通常在 Java 中解析为 Object
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "explicit_constructor_invocation", m[constants.RelNodeAstKind])
			},
		},
	}

	// 3. 执行匹配断言
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 匹配原则：类型为 CREATE + 目标 QN 一致 + SourceQN 包含关系
			if rel.Type == model.Create &&
				rel.Target.QualifiedName == exp.targetQN &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) {

				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing Create relation: %s -> %s", exp.sourceQN, exp.targetQN)
	}
}
