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

// ExpectedCase 定义了通用的赋值关系测试断言模型
type ExpectedCase struct {
	name        string
	sourceMatch string // 精确或包含匹配 Source.QualifiedName
	targetMatch string // 精确匹配 Target.QualifiedName (全路径)
	lineNum     int    // 可选：行号匹配（若传 0 则跳过行号校验）
	checkMores  func(t *testing.T, mores map[string]interface{})
}

func RunCases(t *testing.T, expectedRels []ExpectedCase, allRelations []*model.DependencyRelation, relType model.DependencyType) {
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 1. 严格过滤非 Assign 类型
			if rel.Type != relType {
				continue
			}

			// 2. Source.QualifiedName 柔性模糊/子串包含校验
			if !strings.Contains(rel.Source.QualifiedName, exp.sourceMatch) {
				continue
			}

			// 3. Target 属性多模态路由校验
			if exp.targetMatch != "" && rel.Target.QualifiedName != exp.targetMatch {
				continue
			}

			// 4. 行号精确校验（只有当用例中给定了非 0 值时才强制比对）
			if exp.lineNum > 0 && rel.Location.StartLine != exp.lineNum {
				continue
			}

			// 命中目标，锁定成功并执行细粒度自定义断言
			found = true
			if exp.checkMores != nil {
				exp.checkMores(t, rel.Mores)
			}
			break
		}

		assert.True(t, found, "Missing %s relation: %s -> %s", relType, exp.sourceMatch, exp.targetMatch)
	}
}

// ============================================================================
// === 1. 标准赋值场景测试 ===
// ============================================================================

func TestJavaExtractor_Assign(t *testing.T) {
	testFile := test.GetTestFilePath(filepath.Join("extractor", "assign", "AssignRelationSuite.java"))
	files := []string{testFile}

	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Assign})

	expectedRels := []ExpectedCase{
		// 1. 字段声明初始化
		{
			sourceMatch: "com.example.rel.AssignRelationSuite.count",
			targetMatch: "com.example.rel.AssignRelationSuite.count",
			lineNum:     17,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelAssignIsInitializer])
				assert.Equal(t, "0", m[constants.RelAssignRightExpression])
			},
		},
		// 2. 静态块赋值
		{
			sourceMatch: "com.example.rel.AssignRelationSuite.$static$1",
			targetMatch: "com.example.rel.AssignRelationSuite.status",
			lineNum:     33,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, false, m[constants.RelAssignIsInitializer])
				assert.Equal(t, "\"INIT\"", m[constants.RelAssignRightExpression])
			},
		},
		// 3. 局部变量基础赋值
		{
			sourceMatch: "com.example.rel.AssignRelationSuite.testAssignments(int)",
			targetMatch: "com.example.rel.AssignRelationSuite.testAssignments(int).local",
			lineNum:     49,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelAssignIsInitializer])
				assert.Equal(t, "10", m[constants.RelAssignRightExpression])
			},
		},
		{
			sourceMatch: "com.example.rel.AssignRelationSuite.testAssignments(int)",
			targetMatch: "com.example.rel.AssignRelationSuite.testAssignments(int).local",
			lineNum:     62,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, false, m[constants.RelAssignIsInitializer])
				assert.Equal(t, "20", m[constants.RelAssignRightExpression])
			},
		},
		// 4. 成员变量赋值
		{
			sourceMatch: "com.example.rel.AssignRelationSuite.testAssignments(int)",
			targetMatch: "com.example.rel.AssignRelationSuite.count",
			lineNum:     75,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, false, m[constants.RelAssignIsInitializer])
				assert.Equal(t, "100", m[constants.RelAssignRightExpression])
			},
		},
		// 5. 复合赋值
		{
			sourceMatch: "com.example.rel.AssignRelationSuite.testAssignments(int)",
			targetMatch: "com.example.rel.AssignRelationSuite.count",
			lineNum:     88,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "+=", m[constants.RelAssignOperator])
				assert.Equal(t, false, m[constants.RelAssignIsInitializer])
				assert.Equal(t, "5", m[constants.RelAssignRightExpression])
			},
		},
		// 6. 链式赋值
		{
			sourceMatch: "com.example.rel.AssignRelationSuite.testAssignments(int)",
			targetMatch: "com.example.rel.AssignRelationSuite.testAssignments(int).a",
			lineNum:     104,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "b = c = 50", m[constants.RelAssignRightExpression])
			},
		},
		{
			sourceMatch: "com.example.rel.AssignRelationSuite.testAssignments(int)",
			targetMatch: "com.example.rel.AssignRelationSuite.testAssignments(int).b",
			lineNum:     104,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "c = 50", m[constants.RelAssignRightExpression])
			},
		},
		{
			sourceMatch: "com.example.rel.AssignRelationSuite.testAssignments(int)",
			targetMatch: "com.example.rel.AssignRelationSuite.testAssignments(int).c",
			lineNum:     104,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "50", m[constants.RelAssignRightExpression])
			},
		},
		// 7. 更新表达式
		{
			sourceMatch: "com.example.rel.AssignRelationSuite.testAssignments(int)",
			targetMatch: "com.example.rel.AssignRelationSuite.count",
			lineNum:     116,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "++", m[constants.RelAssignOperator])
				assert.Equal(t, "count++", m[constants.RelRawText])
			},
		},
		{
			sourceMatch: "com.example.rel.AssignRelationSuite.testAssignments(int)",
			targetMatch: "com.example.rel.AssignRelationSuite.count",
			lineNum:     117,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "--", m[constants.RelAssignOperator])
				assert.Equal(t, "--count", m[constants.RelRawText])
			},
		},
		// 8. 数组元素赋值
		{
			sourceMatch: "com.example.rel.AssignRelationSuite.testAssignments(int)",
			targetMatch: "com.example.rel.AssignRelationSuite.testAssignments(int).arr",
			lineNum:     120,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "new int[5]", m[constants.RelAssignRightExpression])
			},
		},
		{
			sourceMatch: "com.example.rel.AssignRelationSuite.testAssignments(int)",
			targetMatch: "com.example.rel.AssignRelationSuite.testAssignments(int).arr",
			lineNum:     131,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "arr[0] = 99", m[constants.RelRawText])
			},
		},
		// 9. Lambda 内部赋值
		{
			sourceMatch: "com.example.rel.AssignRelationSuite.testAssignments(int).lambda$1",
			targetMatch: "com.example.rel.AssignRelationSuite.count",
			lineNum:     145,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "300", m[constants.RelAssignRightExpression])
			},
		},
		{
			sourceMatch: "com.example.rel.AssignRelationSuite.testAssignments(int).lambda$1",
			targetMatch: "com.example.rel.AssignRelationSuite.testAssignments(int).lambda$1.temp",
			lineNum:     146,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelAssignIsInitializer])
				assert.Equal(t, "1", m[constants.RelAssignRightExpression])
			},
		},
		{
			sourceMatch: "com.example.rel.AssignRelationSuite.testAssignments(int).lambda$1",
			targetMatch: "com.example.rel.AssignRelationSuite.testAssignments(int).lambda$1.temp",
			lineNum:     147,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "2", m[constants.RelAssignRightExpression])
			},
		},
		// 10. 构造函数赋值
		{
			sourceMatch: "com.example.rel.AssignRelationSuite.AssignRelationSuite(int)",
			targetMatch: "com.example.rel.AssignRelationSuite.count",
			lineNum:     163,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, false, m[constants.RelAssignIsInitializer])
				assert.Equal(t, "initialCount", m[constants.RelAssignRightExpression])
			},
		},
	}

	RunCases(t, expectedRels, allRelations, model.Assign)
}

// ============================================================================
// === 2. 面向对象类级变量、Receiver 场景测试 ===
// ============================================================================

func TestJavaExtractor_AssignClass(t *testing.T) {
	testFile := test.GetTestFilePath(filepath.Join("extractor", "assign", "AssignRelationForClassSuite.java"))
	files := []string{testFile}

	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Assign})

	expectedRels := []ExpectedCase{
		// 0. 字段声明处的初始化 (count = 0)
		{
			sourceMatch: "com.example.rel.AssignRelationForClassSuite.count",
			targetMatch: "com.example.rel.AssignRelationForClassSuite.count",
			lineNum:     8,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelAssignIsInitializer])
				assert.Equal(t, "0", m[constants.RelAssignRightExpression])
			},
		},
		// 1. 局部变量初始化 (local = 10)
		{
			sourceMatch: "com.example.rel.AssignRelationForClassSuite.testAssignments(int)",
			targetMatch: "com.example.rel.AssignRelationForClassSuite.testAssignments(int).local",
			lineNum:     18,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelAssignIsInitializer])
				assert.Equal(t, "10", m[constants.RelAssignRightExpression])
			},
		},
		// 2. 隐式 this 字段赋值 (count += 5)
		{
			sourceMatch: "com.example.rel.AssignRelationForClassSuite.testAssignments(int)",
			targetMatch: "com.example.rel.AssignRelationForClassSuite.count",
			lineNum:     24,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "+=", m[constants.RelAssignOperator])
				assert.Equal(t, "this", m[constants.RelAssignReceiver])
				assert.Equal(t, "5", m[constants.RelAssignRightExpression])
			},
		},
		// 3. 显式 this 字段赋值 (this.count = 100)
		{
			sourceMatch: "com.example.rel.AssignRelationForClassSuite.testAssignments(int)",
			targetMatch: "com.example.rel.AssignRelationForClassSuite.count",
			lineNum:     30,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "this", m[constants.RelAssignReceiver])
				assert.Equal(t, "100", m[constants.RelAssignRightExpression])
			},
		},
		// 4. 静态字段赋值 (AssignRelationForClassSuite.TAG = "UPDATED")
		{
			sourceMatch: "com.example.rel.AssignRelationForClassSuite.testAssignments(int)",
			targetMatch: "com.example.rel.AssignRelationForClassSuite.TAG",
			lineNum:     36,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "AssignRelationForClassSuite", m[constants.RelAssignReceiver])
				assert.Equal(t, "\"UPDATED\"", m[constants.RelAssignRightExpression])
			},
		},
		// 5. 跨对象字段赋值 (node.name = "NewName")
		{
			sourceMatch: "com.example.rel.AssignRelationForClassSuite.testAssignments(int)",
			targetMatch: "com.example.rel.AssignRelationForClassSuite.DataNode.name",
			lineNum:     43,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "node", m[constants.RelAssignReceiver])
				assert.Equal(t, "\"NewName\"", m[constants.RelAssignRightExpression])
			},
		},
		// 6. 参数二次赋值 (param = 200)
		{
			sourceMatch: "com.example.rel.AssignRelationForClassSuite.testAssignments(int)",
			targetMatch: "com.example.rel.AssignRelationForClassSuite.testAssignments(int).param",
			lineNum:     49,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, false, m[constants.RelAssignIsInitializer])
				assert.Equal(t, "200", m[constants.RelAssignRightExpression])
			},
		},
	}

	RunCases(t, expectedRels, allRelations, model.Assign)
}

// ============================================================================
// === 3. 数据流(DataFlow)赋值传递场景测试 ===
// ============================================================================

func TestJavaExtractor_AssignDataFlow(t *testing.T) {
	testFile := test.GetTestFilePath(filepath.Join("extractor", "assign", "AssignRelationForDataFlow.java"))
	files := []string{testFile}

	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Assign})

	expectedRels := []ExpectedCase{
		// 1. 常量赋值 (this.data = "CONST")
		{
			sourceMatch: "com.example.rel.AssignRelationForDataFlow.testDataFlow()",
			targetMatch: "com.example.rel.AssignRelationForDataFlow.data",
			lineNum:     17,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelNodeAstKind])
				assert.Equal(t, "=", m[constants.RelAssignOperator])
				assert.Equal(t, "this", m[constants.RelAssignReceiver])
				assert.Equal(t, "\"CONST\"", m[constants.RelAssignRightExpression])
			},
		},
		// 2. 返回值流向 (Object localObj = fetch())
		{
			sourceMatch: "com.example.rel.AssignRelationForDataFlow.testDataFlow()",
			targetMatch: "com.example.rel.AssignRelationForDataFlow.testDataFlow().localObj",
			lineNum:     30,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelNodeAstKind])
				assert.Equal(t, true, m[constants.RelAssignIsInitializer])
				assert.Equal(t, "fetch()", m[constants.RelAssignRightExpression])
			},
		},
		// 3. 转换流向 (String msg = (String) localObj)
		{
			sourceMatch: "com.example.rel.AssignRelationForDataFlow.testDataFlow()",
			targetMatch: "com.example.rel.AssignRelationForDataFlow.testDataFlow().msg",
			lineNum:     43,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelNodeAstKind])
				assert.Equal(t, true, m[constants.RelAssignIsInitializer])
				assert.Equal(t, "msg", m[constants.RelAssignTargetName])
				assert.Equal(t, "(String) localObj", m[constants.RelAssignRightExpression])
			},
		},
	}

	RunCases(t, expectedRels, allRelations, model.Assign)
}
