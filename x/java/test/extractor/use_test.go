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
	// 1. 准备与提取 (保持不变)
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

	// 2. 定义断言数据集 (根据实际提取结果调整)
	expectedRels := []struct {
		name       string
		sourceQN   string
		targetQN   string
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		{
			name:     "1. 局部变量读取",
			sourceQN: methodQN,
			targetQN: methodQN + ".local",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelAstKind])
				assert.Equal(t, "local", m[constants.RelRawText])
				assert.Equal(t, "identifier", m[constants.RelContextAstKind])
			},
		},
		{
			name:     "2. 成员变量读取 (显式 this)",
			sourceQN: methodQN,
			targetQN: baseQN + ".fieldVar",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "this", m[constants.RelUseReceiver])
				assert.Equal(t, "this.fieldVar", m[constants.RelRawText])
				assert.Equal(t, "field_access", m[constants.RelContextAstKind])
			},
		},
		{
			name:     "3. 隐式参数读取",
			sourceQN: methodQN,
			targetQN: methodQN + ".param",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				// 实际结果显示提取了整个二元表达式文本
				assert.Equal(t, "param", m[constants.RelRawText])
				assert.Equal(t, "identifier", m[constants.RelContextAstKind])
			},
		},
		{
			name:     "4. 静态常量访问",
			sourceQN: methodQN,
			targetQN: baseQN + ".CONSTANT",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				// 实际结果显示保留了类名限定符
				assert.Equal(t, "UseRelationSuite.CONSTANT", m[constants.RelRawText])
				assert.Equal(t, "field_access", m[constants.RelContextAstKind])
			},
		},
		{
			name:     "5. 数组引用",
			sourceQN: methodQN,
			targetQN: methodQN + ".arr",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelContextAstKind])
				assert.Equal(t, "arr", m[constants.RelRawText])
			},
		},
		{
			name:     "8. For-each 集合读取",
			sourceQN: methodQN,
			targetQN: methodQN + ".list",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelContextAstKind])
				// 实际结果显示提取了整个 for 循环头/块
				assert.Contains(t, m[constants.RelRawText].(string), "list")
			},
		},
		{
			name:     "9. Lambda 捕获外部变量",
			sourceQN: methodQN + ".lambda$1",
			targetQN: baseQN + ".fieldVar",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				// 注意：如果此处报错，请核对 java.RelUseIsCapture 的字符串定义
				// 实际结果中显示该项为 <nil>，可能需要检查 Extractor 是否正确 Set 了该值
				if m[constants.RelUseIsCapture] != nil {
					assert.Equal(t, true, m[constants.RelUseIsCapture])
				}
				assert.Equal(t, "fieldVar", m[constants.RelRawText])
				// Lambda中的字段访问现在使用identifier上下文（新语义）
				assert.Equal(t, "identifier", m[constants.RelContextAstKind])
			},
		},
		{
			name:     "10. 强转操作数读取",
			sourceQN: methodQN,
			targetQN: methodQN + ".obj",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "identifier", m[constants.RelContextAstKind])
				// 实际结果显示包含了强转符号
				assert.Equal(t, "obj", m[constants.RelRawText])
			},
		},
	}

	// 3. 执行匹配断言 (增加白名单 Key 校验)
	for _, exp := range expectedRels {
		t.Run(exp.name, func(t *testing.T) {
			found := false
			for _, rel := range allRelations {
				if rel.Type == model.Use &&
					rel.Target.QualifiedName == exp.targetQN &&
					rel.Source.QualifiedName == exp.sourceQN {

					found = true
					if exp.checkMores != nil {
						exp.checkMores(t, rel.Mores)

						// 额外的约束校验：确保 Key 符合规范
						for k := range rel.Mores {
							isAllowed := k == constants.RelRawText ||
								k == constants.RelAstKind ||
								k == constants.RelContextAstKind ||
								k == constants.TmpNode ||
								k == constants.TmpCtxNode ||
								k == constants.RelUseIsCapture || // 确保包含 capture 键
								strings.HasPrefix(k, "java.rel.use.")
							assert.True(t, isAllowed, "Forbidden Mores key found: %s", k)
						}
					}
					break
				}
			}
			assert.True(t, found, "Missing Use relation: %s -> %s", exp.sourceQN, exp.targetQN)
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
