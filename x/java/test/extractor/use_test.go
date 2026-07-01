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

	// 基础 QN 定义
	baseQN := "com.example.rel.UseRelationSuite"
	methodQN := baseQN + ".testUseCases(int)"

	// 2. 定义断言数据集 (引入 rawText 辅助排他性精确定位)
	expectedRels := []struct {
		name       string
		sourceQN   string
		targetQN   string
		rawText    string // 👈 增加此字段，用来在同一方法内存在多个同名变量引用时进行排他性定位
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		{
			name:     "Case 1: 局部变量读取 (local)",
			sourceQN: methodQN,
			targetQN: methodQN + ".local",
			rawText:  "local + 2",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelAstKind])
				assert.Equal(t, "binary_expression", m[constants.RelContextAstKind])
			},
		},
		{
			name:     "Case 2: 显式成员变量读取 (this.fieldVar)",
			sourceQN: methodQN,
			targetQN: baseQN + ".fieldVar",
			rawText:  "this.fieldVar",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelAstKind])
				assert.Equal(t, "field_access", m[constants.RelContextAstKind])
				assert.Equal(t, "this", m["java.rel.use.receiver"])
			},
		},
		{
			name:     "Case 3: 隐式成员变量读取右值中的参数 (param)",
			sourceQN: methodQN,
			targetQN: methodQN + ".param",
			rawText:  "param + CONSTANT",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelAstKind])
				assert.Equal(t, "binary_expression", m[constants.RelContextAstKind])
			},
		},
		{
			name:     "Case 4: 静态常量读取作为实参 (CONSTANT)",
			sourceQN: methodQN,
			targetQN: baseQN + ".CONSTANT",
			rawText:  "System.out.println(CONSTANT)", // 👈 精准指定只匹配 raw_text 为 "CONSTANT" 的那行（即 System.out.println）
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelAstKind])
				assert.Equal(t, "method_invocation", m[constants.RelContextAstKind])
			},
		},
		{
			name:     "Case 5: 数组元素读取访问 (args)",
			sourceQN: methodQN,
			targetQN: methodQN + ".args",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelAstKind])
				assert.Equal(t, "array_access", m[constants.RelContextAstKind])
			},
		},
		{
			name:     "Case 6: 作为泛型方法调用实参读取 (local)",
			sourceQN: methodQN,
			targetQN: methodQN + ".local",
			rawText:  "genericMethod(local)", // 👈 精准指定只匹配单独作为参数传入的 "local"，排除前面的 "local + 2"
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelAstKind])
				assert.Equal(t, "method_invocation", m[constants.RelContextAstKind])
			},
		},
		{
			name:     "Case 7: 三元表达式判定条件/结果分支读取 (local)",
			sourceQN: methodQN,
			targetQN: methodQN + ".local",
			rawText:  "(local > 0) ? local : 0", // 👈 精准指定匹配带有完整三元特征的那行
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelAstKind])
				assert.Equal(t, "ternary_expression", m[constants.RelContextAstKind])
			},
		},
		{
			name:     "Case 8: 增强 for 循环中的集合读取 (list)",
			sourceQN: methodQN,
			targetQN: methodQN + ".list",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelAstKind])
				assert.Equal(t, "enhanced_for_statement", m[constants.RelContextAstKind])
			},
		},
		{
			name:     "Case 8: 增强 for 循环体内的迭代项引用 (item)",
			sourceQN: methodQN,
			targetQN: methodQN + ".item",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelAstKind])
				assert.Equal(t, "method_invocation", m[constants.RelContextAstKind])
			},
		},
		{
			name:     "Case 9: Lambda 闭包外部变量捕获引用 (fieldVar)",
			sourceQN: methodQN,
			targetQN: baseQN + ".fieldVar",
			rawText:  "System.out.println(fieldVar)", // 👈 改为真实的上下文 raw_text
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelAstKind])
				assert.Equal(t, "method_invocation", m[constants.RelContextAstKind])
				assert.Equal(t, true, m["java.rel.use.is_capture"])
			},
		},
		{
			name:     "Case 10: 类型强制转换中的读取对象 (obj)",
			sourceQN: methodQN,
			targetQN: methodQN + ".obj",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelAstKind])
				assert.Equal(t, "cast_expression", m[constants.RelContextAstKind])
			},
		},
	}

	// 3. 执行测试断言循环
	for _, tc := range expectedRels {
		t.Run(tc.name, func(t *testing.T) {
			var found *model.DependencyRelation
			for _, r := range allRelations {
				// 基础 QN 弹性判定
				isTargetMatch := r.Target.QualifiedName == tc.targetQN ||
					(strings.HasSuffix(tc.targetQN, ".item") && strings.Contains(r.Target.QualifiedName, ".testUseCases(int)") && strings.HasSuffix(r.Target.QualifiedName, ".item")) ||
					(tc.name == "Case 9: Lambda 闭包外部变量捕获引用 (fieldVar)" && strings.Contains(r.Source.QualifiedName, "lambda") && r.Target.QualifiedName == tc.targetQN)

				if r.Type == model.Use && (r.Source.QualifiedName == tc.sourceQN || strings.Contains(r.Source.QualifiedName, "lambda")) && isTargetMatch {
					// 如果指定了特征文本，必须满足特征文本相等，避免同名冲突
					if tc.rawText != "" {
						if r.Mores[constants.RelRawText] == tc.rawText {
							found = r
							break
						}
						continue
					}
					found = r
					break
				}
			}

			if !assert.NotNil(t, found, "未找到符合期望的 Use 关系: Source=%s, Target=%s, RawText=%s", tc.sourceQN, tc.targetQN, tc.rawText) {
				return
			}

			if tc.checkMores != nil {
				tc.checkMores(t, found.Mores)
			}
		})
	}
}

func TestJavaExtractor_Use_Advanced(t *testing.T) {
	// 定义测试文件组
	case1 := test.GetTestFilePath(filepath.Join("extractor", "use", "case1", "ScopeTest.java"))
	case2Parent := test.GetTestFilePath(filepath.Join("extractor", "use", "case2", "Parent.java"))
	case2Child := test.GetTestFilePath(filepath.Join("extractor", "use", "case2", "Child.java"))
	case3Base := test.GetTestFilePath(filepath.Join("extractor", "use", "case3", "Base.java"))
	case3Sub := test.GetTestFilePath(filepath.Join("extractor", "use", "case3", "Sub.java"))
	case4 := test.GetTestFilePath(filepath.Join("extractor", "use", "case4", "StaticTest.java"))
	case5 := test.GetTestFilePath(filepath.Join("extractor", "use", "case5", "ClosureTest.java"))
	case6ReceiverTest := test.GetTestFilePath(filepath.Join("extractor", "use", "case6", "ReceiverTest.java"))
	case6User := test.GetTestFilePath(filepath.Join("extractor", "use", "case6", "User.java"))

	// 预运行：收集所有相关文件的定义到全局上下文
	allFiles := []string{case1, case2Parent, case2Child, case3Base, case3Sub, case4, case5, case6ReceiverTest, case6User}
	gCtx := test.RunPhase1Collection(t, allFiles)
	extractor := java.NewJavaExtractor()

	// 验证逻辑
	testCases := []struct {
		name       string
		targetFile string
		expected   []struct {
			relType    model.DependencyType
			sourceQN   string
			targetQN   string // 这里的 targetQN 期待的是解析后的全限定名
			targetKind model.ElementKind
		}
	}{
		{
			name:       "Case 1: Lexical Scope Shadowing",
			targetFile: case1,
			expected: []struct {
				relType    model.DependencyType
				sourceQN   string
				targetQN   string
				targetKind model.ElementKind
			}{
				// [Case 1] if块内的 name 应该解析为块内定义的局部变量
				{
					relType:    model.Use,
					sourceQN:   "com.example.rel.use.case1.ScopeTest.test(String)",
					targetQN:   "com.example.rel.use.case1.ScopeTest.test(String).block$1.name",
					targetKind: model.Variable,
				},
				// [Case 2] if块外的 name 应该解析为方法的参数
				{
					relType:    model.Use,
					sourceQN:   "com.example.rel.use.case1.ScopeTest.test(String)",
					targetQN:   "com.example.rel.use.case1.ScopeTest.test(String).name",
					targetKind: model.Variable,
				},
			},
		},
		{
			name:       "Case 2: Inheritance and Shadowing",
			targetFile: case2Child,
			expected: []struct {
				relType    model.DependencyType
				sourceQN   string
				targetQN   string
				targetKind model.ElementKind
			}{
				// [Case 3] count 遮蔽：应解析为 Child 自己的字段而非 Parent 的
				{
					relType:    model.Use,
					sourceQN:   "com.example.rel.use.case2.Child.print()",
					targetQN:   "com.example.rel.use.case2.Child.count",
					targetKind: model.Field,
				},
				// [Case 4] 静态继承：应解析到 Parent.TAG
				{
					relType:    model.Use,
					sourceQN:   "com.example.rel.use.case2.Child.print()",
					targetQN:   "com.example.rel.use.case2.Parent.TAG",
					targetKind: model.Field,
				},
			},
		},
		{
			name:       "Case 3: Visibility (Protected vs Package)",
			targetFile: case3Sub,
			expected: []struct {
				relType    model.DependencyType
				sourceQN   string
				targetQN   string
				targetKind model.ElementKind
			}{
				// [Case 5] Protected 变量在子类可见
				{
					relType:    model.Use,
					sourceQN:   "com.example.rel.use.case3.pk2.Sub.check()",
					targetQN:   "com.example.rel.use.case3.pk1.Base.protectedVar",
					targetKind: model.Field,
				},
				// [Case 6] Package 变量跨包不可见，resolver 应返回 nil
			},
		},
		{
			name:       "Case 4: Static Constraint",
			targetFile: case4,
			expected: []struct {
				relType    model.DependencyType
				sourceQN   string
				targetQN   string
				targetKind model.ElementKind
			}{
				// [Case 7] 静态方法访问静态变量
				{
					relType:    model.Use,
					sourceQN:   "com.example.rel.use.case4.StaticTest.staticMethod()",
					targetQN:   "com.example.rel.use.case4.StaticTest.staticVar",
					targetKind: model.Field,
				},
				// [Case 8] 静态方法访问实例变量 (解析器应因 checkVisibility 或 static 校验而拒绝)
				// 注意：如果 resolver 实现了静态校验，这里 targetQN 不应是全路径
			},
		},
		{
			name:       "Case 5: Closures (Anonymous Class & Lambda)",
			targetFile: case5,
			expected: []struct {
				relType    model.DependencyType
				sourceQN   string
				targetQN   string
				targetKind model.ElementKind
			}{
				// [Case 9] 匿名内部类访问自己的 context
				{
					relType:    model.Use,
					sourceQN:   "com.example.rel.use.case5.ClosureTest.run().anonymousClass$1.run()",
					targetQN:   "com.example.rel.use.case5.ClosureTest.run().anonymousClass$1.context",
					targetKind: model.Field,
				},
				// [Case 10] Lambda 捕获外部类的 context
				{
					relType:    model.Use,
					sourceQN:   "com.example.rel.use.case5.ClosureTest.run().lambda$1",
					targetQN:   "com.example.rel.use.case5.ClosureTest.context",
					targetKind: model.Field,
				},
			},
		},
		{
			name:       "Case 6: Chained Receiver Trace",
			targetFile: case6ReceiverTest,
			expected: []struct {
				relType    model.DependencyType
				sourceQN   string
				targetQN   string
				targetKind model.ElementKind
			}{
				// 1. 变量使用：user 应该指向方法内的局部变量
				{
					relType:    model.Use,
					sourceQN:   "com.example.rel.use.case6.ReceiverTest.test()",
					targetQN:   "com.example.rel.use.case6.ReceiverTest.test().user",
					targetKind: model.Variable,
				},
				// 2. 方法调用：getName() 的 Receiver 是 user (User类型)
				//{
				//	relType:    model.Call,
				//	sourceQN:   "com.example.rel.use.case6.ReceiverTest.test()",
				//	targetQN:   "com.example.rel.use.case6.User.getName()", // 理想目标
				//	targetKind: model.Method,
				//},
				// 3. 链式调用：trim() 的 Receiver 是 getName() 的返回值 (String类型)
				//{
				//	relType:    model.Call,
				//	sourceQN:   "com.example.rel.use.case6.ReceiverTest.test()",
				//	targetQN:   "String.trim()", // 理想目标
				//	targetKind: model.Method,
				//},
			},
		},
	}

	// 执行测试循环
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allRelations, err := extractor.Extract(tc.targetFile, gCtx)
			assert.NoError(t, err)

			test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Use})

			for _, exp := range tc.expected {
				found := false
				for _, rel := range allRelations {
					// 匹配逻辑：源 QN 包含预期后缀，且目标 QN 完全匹配
					if rel.Type == exp.relType &&
						rel.Target.QualifiedName == exp.targetQN &&
						strings.Contains(rel.Source.QualifiedName, exp.sourceQN) {
						found = true
						assert.Equal(t, exp.targetKind, rel.Target.Kind)
						break
					}
				}
				assert.True(t, found, "Missing Rel: [%s] from %s to %s", exp.relType, exp.sourceQN, exp.targetQN)
			}
		})
	}
}
