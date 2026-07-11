package extractor

import (
	"path/filepath"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
	"github.com/stretchr/testify/assert"
)

func TestJavaExtractor_Use(t *testing.T) {
	// 1. 准备与提取
	testFile := test.GetTestFilePath(filepath.Join("extractor", "use", "UseRelationSuite.java"))
	files := []string{testFile}

	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Use})

	// 2. 定义断言数据集 (引入 rawText 辅助排他性精确定位)
	expectedRels := []ExpectedCase{
		{
			name:        "Case 1: 局部变量读取 (local)",
			sourceMatch: "com.example.rel.UseRelationSuite.testUseCases(int)",
			targetMatch: "com.example.rel.UseRelationSuite.testUseCases(int).local",
			lineNum:     16,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelNodeAstKind])
				assert.Equal(t, "binary_expression", m[constants.RelContextAstKind])
				assert.Equal(t, "local + 2", m[constants.RelRawText])
			},
		},
		{
			name:        "Case 2: 显式成员变量读取 (this.fieldVar)",
			sourceMatch: "com.example.rel.UseRelationSuite.testUseCases(int)",
			targetMatch: "com.example.rel.UseRelationSuite.fieldVar",
			lineNum:     22,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelNodeAstKind])
				assert.Equal(t, "field_access", m[constants.RelContextAstKind])
				assert.Equal(t, "this", m[constants.RelUseReceiver])
				assert.Equal(t, "this.fieldVar", m[constants.RelRawText])
			},
		},
		{
			name:        "Case 3: 隐式成员变量读取右值中的参数 (param)",
			sourceMatch: "com.example.rel.UseRelationSuite.testUseCases(int)",
			targetMatch: "com.example.rel.UseRelationSuite.testUseCases(int).param",
			lineNum:     29,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelNodeAstKind])
				assert.Equal(t, "binary_expression", m[constants.RelContextAstKind])
				assert.Equal(t, "param + CONSTANT", m[constants.RelRawText])
			},
		},
		{
			name:        "Case 4: 静态常量读取作为实参 (CONSTANT)",
			sourceMatch: "com.example.rel.UseRelationSuite.testUseCases(int)",
			targetMatch: "com.example.rel.UseRelationSuite.CONSTANT",
			lineNum:     35,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelNodeAstKind])
				assert.Equal(t, "method_invocation", m[constants.RelContextAstKind])
				assert.Equal(t, "System.out.println(CONSTANT)", m[constants.RelRawText])
			},
		},
		{
			name:        "Case 5: 数组元素读取访问 (args)",
			sourceMatch: "com.example.rel.UseRelationSuite.testUseCases(int)",
			targetMatch: "com.example.rel.UseRelationSuite.testUseCases(int).args",
			lineNum:     42,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelNodeAstKind])
				assert.Equal(t, "array_access", m[constants.RelContextAstKind])
				assert.Equal(t, "args[0]", m[constants.RelRawText])
			},
		},
		{
			name:        "Case 6: 作为泛型方法调用实参读取 (local)",
			sourceMatch: "com.example.rel.UseRelationSuite.testUseCases(int)",
			targetMatch: "com.example.rel.UseRelationSuite.testUseCases(int).local",
			lineNum:     48,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelNodeAstKind])
				assert.Equal(t, "method_invocation", m[constants.RelContextAstKind])
				assert.Equal(t, "genericMethod(local)", m[constants.RelRawText])
			},
		},
		{
			name:        "Case 7: 三元表达式判定条件/结果分支读取 (local)",
			sourceMatch: "com.example.rel.UseRelationSuite.testUseCases(int)",
			targetMatch: "com.example.rel.UseRelationSuite.testUseCases(int).local",
			lineNum:     54,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelNodeAstKind])
				assert.Equal(t, "ternary_expression", m[constants.RelContextAstKind])
				assert.Equal(t, "(local > 0) ? local : 0", m[constants.RelRawText])
			},
		},
		{
			name:        "Case 8: 增强 for 循环中的集合读取 (list)",
			sourceMatch: "com.example.rel.UseRelationSuite.testUseCases(int)",
			targetMatch: "com.example.rel.UseRelationSuite.testUseCases(int).list",
			lineNum:     61,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelNodeAstKind])
				assert.Equal(t, "enhanced_for_statement", m[constants.RelContextAstKind])
			},
		},
		{
			name:        "Case 8: 增强 for 循环体内的迭代项引用 (item)",
			sourceMatch: "com.example.rel.UseRelationSuite.testUseCases(int)",
			targetMatch: "com.example.rel.UseRelationSuite.testUseCases(int).block$1.item",
			lineNum:     65,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelNodeAstKind])
				assert.Equal(t, "method_invocation", m[constants.RelContextAstKind])
			},
		},
		{
			name:        "Case 9: Lambda 闭包外部变量捕获引用 (fieldVar)",
			sourceMatch: "com.example.rel.UseRelationSuite.testUseCases(int)",
			targetMatch: "com.example.rel.UseRelationSuite.fieldVar",
			lineNum:     73,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelNodeAstKind])
				assert.Equal(t, "method_invocation", m[constants.RelContextAstKind])
				assert.Equal(t, true, m[constants.RelUseIsCapture])
				assert.Equal(t, "System.out.println(fieldVar)", m[constants.RelRawText])
			},
		},
		{
			name:        "Case 10: 类型强制转换中的读取对象 (obj)",
			sourceMatch: "com.example.rel.UseRelationSuite.testUseCases(int)",
			targetMatch: "com.example.rel.UseRelationSuite.testUseCases(int).obj",
			lineNum:     81,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelNodeAstKind])
				assert.Equal(t, "cast_expression", m[constants.RelContextAstKind])
			},
		},
	}

	RunCases(t, expectedRels, allRelations, model.Use)
}

func TestJavaExtractor_Use_Case1(t *testing.T) {
	// 1. 准备与提取
	testFile := test.GetTestFilePath(filepath.Join("extractor", "use", "case1", "ScopeTest.java"))
	files := []string{testFile}

	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Use})

	// 2. 定义断言数据集 (引入 rawText 辅助排他性精确定位)
	expectedRels := []ExpectedCase{
		{
			sourceMatch: "com.example.rel.use.case1.ScopeTest.test(String)",
			targetMatch: "com.example.rel.use.case1.ScopeTest.test(String).block$1.name",
			lineNum:     10,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "System.out.println(name)", m[constants.RelRawText])
			},
		},
		{
			sourceMatch: "com.example.rel.use.case1.ScopeTest.test(String)",
			targetMatch: "com.example.rel.use.case1.ScopeTest.test(String).name",
			lineNum:     12,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "System.out.println(name)", m[constants.RelRawText])
			},
		},
	}

	RunCases(t, expectedRels, allRelations, model.Use)
}

func TestJavaExtractor_Use_Case2(t *testing.T) {
	// 1. 准备与提取
	case2Parent := test.GetTestFilePath(filepath.Join("extractor", "use", "case2", "Parent.java"))
	case2Child := test.GetTestFilePath(filepath.Join("extractor", "use", "case2", "Child.java"))
	files := []string{case2Parent, case2Child}

	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(case2Child, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Use})

	// 2. 定义断言数据集 (引入 rawText 辅助排他性精确定位)
	expectedRels := []ExpectedCase{
		{
			sourceMatch: "com.example.rel.use.case2.Child.print()",
			targetMatch: "com.example.rel.use.case2.Child.count",
			lineNum:     7,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "System.out.println(count)", m[constants.RelRawText])
			},
		},
		{
			sourceMatch: "com.example.rel.use.case2.Child.print()",
			targetMatch: "com.example.rel.use.case2.Parent.TAG",
			lineNum:     8,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "System.out.println(TAG)", m[constants.RelRawText])
			},
		},
	}

	RunCases(t, expectedRels, allRelations, model.Use)
}

func TestJavaExtractor_Use_Case3(t *testing.T) {
	// 1. 准备与提取
	case3Base := test.GetTestFilePath(filepath.Join("extractor", "use", "case3", "Base.java"))
	case3Sub := test.GetTestFilePath(filepath.Join("extractor", "use", "case3", "Sub.java"))
	files := []string{case3Base, case3Sub}

	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(case3Sub, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Use})

	// 2. 定义断言数据集 (引入 rawText 辅助排他性精确定位)
	expectedRels := []ExpectedCase{
		{
			sourceMatch: "com.example.rel.use.case3.pk2.Sub.check()",
			targetMatch: "com.example.rel.use.case3.pk1.Base.protectedVar",
			lineNum:     7,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "System.out.println(protectedVar)", m[constants.RelRawText])
			},
		},
		// 应解析失败(跨包不可见)
		//{
		//	sourceMatch: "com.example.rel.use.case3.pk2.Sub.check()",
		//	targetMatch: "com.example.rel.use.case3.pk1.Base.packageVar",
		//	lineNum:     8,
		//	checkMores: func(t *testing.T, m map[string]interface{}) {
		//		assert.Equal(t, "System.out.println(packageVar)", m[constants.RelRawText])
		//	},
		//},
	}

	RunCases(t, expectedRels, allRelations, model.Use)
}

func TestJavaExtractor_Use_Case4(t *testing.T) {
	// 1. 准备与提取
	testFile := test.GetTestFilePath(filepath.Join("extractor", "use", "case4", "StaticTest.java"))
	files := []string{testFile}

	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Use})

	// 2. 定义断言数据集 (引入 rawText 辅助排他性精确定位)
	expectedRels := []ExpectedCase{
		// 应解析成功
		{
			sourceMatch: "com.example.rel.use.case4.StaticTest.staticMethod()",
			targetMatch: "com.example.rel.use.case4.StaticTest.staticVar",
			lineNum:     8,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "System.out.println(staticVar)", m[constants.RelRawText])
			},
		},
		// 应解析失败 (静态方法不能引用非静态变量)
		{
			sourceMatch: "com.example.rel.use.case4.StaticTest.staticMethod()",
			targetMatch: "com.example.rel.use.case4.StaticTest.instanceVar",
			lineNum:     9,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "System.out.println(instanceVar)", m[constants.RelRawText])
			},
		},
	}

	RunCases(t, expectedRels, allRelations, model.Use)
}

func TestJavaExtractor_Use_Case5(t *testing.T) {
	// 1. 准备与提取
	testFile := test.GetTestFilePath(filepath.Join("extractor", "use", "case5", "ClosureTest.java"))
	files := []string{testFile}

	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Use})

	// 2. 定义断言数据集 (引入 rawText 辅助排他性精确定位)
	expectedRels := []ExpectedCase{
		{
			sourceMatch: "com.example.rel.use.case5.ClosureTest.run().anonymousClass$1.run()",
			targetMatch: "com.example.rel.use.case5.ClosureTest.run().anonymousClass$1.context",
			lineNum:     11,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "System.out.println(context)", m[constants.RelRawText])
			},
		},
		{
			sourceMatch: "com.example.rel.use.case5.ClosureTest.run().lambda$1",
			targetMatch: "com.example.rel.use.case5.ClosureTest.context",
			lineNum:     17,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "System.out.println(context)", m[constants.RelRawText])
			},
		},
	}

	RunCases(t, expectedRels, allRelations, model.Use)
}

func TestJavaExtractor_Use_Case6(t *testing.T) {
	// 1. 准备与提取
	case6ReceiverTest := test.GetTestFilePath(filepath.Join("extractor", "use", "case6", "ReceiverTest.java"))
	case6User := test.GetTestFilePath(filepath.Join("extractor", "use", "case6", "User.java"))
	files := []string{case6ReceiverTest, case6User}

	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(case6ReceiverTest, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Use, model.Call, model.Assign})

	// 2. 定义断言数据集 (引入 rawText 辅助排他性精确定位)
	expectedRelsForAssign := []ExpectedCase{
		{
			sourceMatch: "com.example.rel.use.case6.ReceiverTest.test()",
			targetMatch: "com.example.rel.use.case6.ReceiverTest.test().user",
			lineNum:     5,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "user = new User()", m[constants.RelRawText])
			},
		},
	}
	RunCases(t, expectedRelsForAssign, allRelations, model.Assign)

	expectedRelsForCall := []ExpectedCase{
		{
			sourceMatch: "com.example.rel.use.case6.ReceiverTest.test()",
			targetMatch: "com.example.rel.use.case6.User.getName()",
			lineNum:     6,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "user.getName().trim()", m[constants.RelRawText])
			},
		},
		{
			sourceMatch: "com.example.rel.use.case6.ReceiverTest.test()",
			targetMatch: "String.trim()",
			lineNum:     6,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "user.getName().trim()", m[constants.RelRawText])
			},
		},
	}
	RunCases(t, expectedRelsForCall, allRelations, model.Call)
}
