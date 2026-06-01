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

func TestJavaExtractor_Assign(t *testing.T) {
	// 1. 准备与提取
	testFile := test.GetTestFilePath(filepath.Join("extractor", "assign", "AssignRelationSuite.java"))
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
		sourceMatch string // 匹配 Source.QualifiedName
		targetMatch string // 匹配 Target.Name
		matchMores  func(m map[string]interface{}) bool
		checkMores  func(t *testing.T, mores map[string]interface{})
	}{
		// 1. 字段声明初始化
		{
			sourceMatch: "AssignRelationSuite.count",
			targetMatch: "count",
			matchMores: func(m map[string]interface{}) bool {
				return m[java.RelAssignIsInitializer] == true
			},
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "0", m[java.RelAssignValueExpression])
			},
		},
		// 2. 静态块赋值
		{
			sourceMatch: "$static$1",
			targetMatch: "status",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "\"INIT\"", m[java.RelAssignValueExpression])
			},
		},
		// 3. 局部变量基础赋值
		{
			sourceMatch: "testAssignments(int)",
			targetMatch: "local",
			matchMores: func(m map[string]interface{}) bool {
				return m[java.RelAssignIsInitializer] == true
			},
		},
		// 6. 链式赋值 (b)
		{
			sourceMatch: "testAssignments(int)",
			targetMatch: "b",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "c = 50", m[java.RelAssignValueExpression])
			},
		},
		// 8. 数组元素赋值 (Target 应该是数组变量名)
		{
			sourceMatch: "testAssignments(int)",
			targetMatch: "arr",
			matchMores: func(m map[string]interface{}) bool {
				return strings.Contains(fmt.Sprintf("%v", m[java.RelRawText]), "arr[0]")
			},
		},
		// 9. Lambda 内部赋值
		{
			sourceMatch: "lambda$1",
			targetMatch: "count",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "300", m[java.RelAssignValueExpression])
			},
		},
	}

	// 执行匹配循环
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 1. 必须是 Assign 关系
			if rel.Type != model.Assign {
				continue
			}

			// 2. 匹配 Source (支持 QN 后缀匹配)
			sourceOk := strings.Contains(rel.Source.QualifiedName, exp.sourceMatch)

			// 3. 匹配 Target (支持短名或 QN 匹配)
			// 关键修复：同时检查 Name 和 QualifiedName
			targetOk := rel.Target.Name == exp.targetMatch ||
				strings.HasSuffix(rel.Target.QualifiedName, "."+exp.targetMatch)

			if sourceOk && targetOk {
				if exp.matchMores != nil && !exp.matchMores(rel.Mores) {
					continue
				}
				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing Assign: %s -> %s", exp.sourceMatch, exp.targetMatch)
	}
}

func TestJavaExtractor_AssignClass(t *testing.T) {
	testFile := test.GetTestFilePath(filepath.Join("extractor", "assign", "AssignRelationForClassSuite.java"))
	files := []string{testFile}

	// 假设 test.RunPhase1Collection 已经处理了符号定义
	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelations(allRelations)

	expectedRels := []struct {
		sourceQN   string
		targetName string // 补全为全路径 QN
		value      string
		checkMores func(t *testing.T, mores map[string]interface{})
	}{
		// 0. 字段声明处的初始化 (count = 0)
		// Source 和 Target 均为 Field 本身 QN
		{
			sourceQN:   "com.example.rel.AssignRelationForClassSuite.count",
			targetName: "com.example.rel.AssignRelationForClassSuite.count",
			value:      "0",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelAssignIsInitializer])
			},
		},
		// 1. 局部变量初始化 (local = 10)
		// Target QN 包含方法名(带参数类型)和变量名
		{
			sourceQN:   "com.example.rel.AssignRelationForClassSuite.testAssignments(int)",
			targetName: "com.example.rel.AssignRelationForClassSuite.testAssignments(int).local",
			value:      "10",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelAssignIsInitializer])
			},
		},
		// 2. 隐式 this 字段赋值 (count += 5)
		{
			sourceQN:   "com.example.rel.AssignRelationForClassSuite.testAssignments(int)",
			targetName: "com.example.rel.AssignRelationForClassSuite.count",
			value:      "5",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "+=", m[java.RelAssignOperator])
				assert.Equal(t, "this", m[java.RelAssignReceiver])
			},
		},
		// 3. 显式 this 字段赋值 (this.count = 100)
		{
			sourceQN:   "com.example.rel.AssignRelationForClassSuite.testAssignments(int)",
			targetName: "com.example.rel.AssignRelationForClassSuite.count",
			value:      "100",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "this", m[java.RelAssignReceiver])
			},
		},
		// 4. 静态字段赋值 (AssignRelationForClassSuite.TAG = "UPDATED")
		{
			sourceQN:   "com.example.rel.AssignRelationForClassSuite.testAssignments(int)",
			targetName: "com.example.rel.AssignRelationForClassSuite.TAG",
			value:      "\"UPDATED\"",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "AssignRelationForClassSuite", m[java.RelAssignReceiver])
			},
		},
		// 5. 跨对象字段赋值 (node.name = "NewName")
		// Target QN 指向 DataNode 内部类中的字段定义
		{
			sourceQN:   "com.example.rel.AssignRelationForClassSuite.testAssignments(int)",
			targetName: "com.example.rel.AssignRelationForClassSuite.DataNode.name",
			value:      "\"NewName\"",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "node", m[java.RelAssignReceiver])
			},
		},
		// 6. 参数二次赋值 (param = 200)
		{
			sourceQN:   "com.example.rel.AssignRelationForClassSuite.testAssignments(int)",
			targetName: "com.example.rel.AssignRelationForClassSuite.testAssignments(int).param",
			value:      "200",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, false, m[java.RelAssignIsInitializer])
			},
		},
	}

	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			relValue, _ := rel.Mores[java.RelAssignValueExpression].(string)

			// 匹配逻辑：
			// Source QN 使用 strings.Contains (防止由于空格等引起的微小不一致)
			// Target QN 使用完全匹配
			if rel.Type == model.Assign &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) &&
				rel.Target.QualifiedName == exp.targetName &&
				relValue == exp.value {

				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing Assign: %s -> %s (value: %s)", exp.sourceQN, exp.targetName, exp.value)
	}
}

func TestJavaExtractor_AssignDataFlow(t *testing.T) {
	// 1. 准备测试文件路径（注意文件名需与 testdata 目录一致）
	testFile := test.GetTestFilePath(filepath.Join("extractor", "assign", "AssignRelationForDataFlow.java"))
	files := []string{testFile}

	// 2. 运行符号收集与提取逻辑
	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	// 打印结果便于调试
	test.PrintRelations(allRelations)

	// 3. 定义预期关系
	expectedRels := []struct {
		sourceQN   string
		targetName string
		value      string // 用于精确定位具体的赋值语句
		checkMores func(t *testing.T, mores map[string]interface{})
	}{
		// --- 1. 常量赋值 (this.data = "CONST") ---
		{
			sourceQN:   "com.example.rel.AssignRelationForDataFlow.testDataFlow",
			targetName: "data",
			value:      "\"CONST\"",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[java.RelAstKind])
				assert.Equal(t, "=", m[java.RelAssignOperator])
			},
		},
		// --- 2. 返回值流向 (Object localObj = fetch()) ---
		{
			sourceQN:   "com.example.rel.AssignRelationForDataFlow.testDataFlow",
			targetName: "localObj",
			value:      "fetch()",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[java.RelAstKind])
				assert.Equal(t, true, m[java.RelAssignIsInitializer])
			},
		},
		// --- 3. 转换流向 (String msg = (String) localObj) ---
		{
			sourceQN:   "com.example.rel.AssignRelationForDataFlow.testDataFlow",
			targetName: "msg",
			value:      "(String) localObj",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[java.RelAstKind])
				assert.Equal(t, true, m[java.RelAssignIsInitializer])
				assert.Equal(t, "msg", m[java.RelAssignTargetName])
			},
		},
	}

	// 4. 执行匹配与验证逻辑
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 获取当前关系的 ValueExpression 以便精确定位
			relValue, _ := rel.Mores[java.RelAssignValueExpression].(string)

			// 匹配 ASSIGN 类型，且 Source QN、Target Name 和 Value 对齐
			if rel.Type == model.Assign &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) &&
				rel.Target.Name == exp.targetName &&
				relValue == exp.value {

				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing Data Flow relation: %s -> %s (value: %s)",
			exp.sourceQN, exp.targetName, exp.value)
	}
}
