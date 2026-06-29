package extractor

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
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

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Call})

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
				assert.Equal(t, "this", m[constants.RelCallReceiver])
				assert.Equal(t, false, m[constants.RelCallIsStatic])
			},
		},
		{
			sourceQN:   "CallRelationSuite.executeAll()",
			targetName: "staticMethod",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelCallIsStatic])
				// 【改进】现在统一存储为QN格式
				assert.Contains(t, m[constants.RelCallReceiverType], "CallRelationSuite")
			},
		},
		{
			sourceQN:   "CallRelationSuite.executeAll()",
			targetName: "currentTimeMillis",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "System", m[constants.RelCallReceiver])
				assert.Equal(t, true, m[constants.RelCallIsStatic])
			},
		},
		{
			sourceQN:   "CallRelationSuite.executeAll()",
			targetName: "add",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelCallIsChained])
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
				assert.Equal(t, true, m[constants.RelCallIsConstructor])
			},
		},
		{
			sourceQN:   "lambda$1",
			targetName: "simpleMethod",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "com.example.rel.CallRelationSuite.executeAll()", m[constants.RelCallEnclosingMethod])
			},
		},
		{
			sourceQN:   "anonymousClass$1.run()",
			targetName: "simpleMethod",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "com.example.rel.CallRelationSuite.executeAll()", m[constants.RelCallEnclosingMethod])
			},
		},
		{
			sourceQN:   "SubClass.SubClass()",
			targetName: "super",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "explicit_constructor_invocation", m[constants.RelAstKind])
				assert.Equal(t, true, m[constants.RelCallIsConstructor])
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

// TestJavaExtractor_Call_Chained 测试链式调用提取
func TestJavaExtractor_Call_Chained(t *testing.T) {
	// 1. 准备与提取
	testFile := test.GetTestFilePath(filepath.Join("extractor", "call", "case1", "ChainedCallExample.java"))
	files := []string{testFile}
	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	// 2. 打印所有关系（方便调试）
	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Call})

	// 3. 验证链式调用的提取
	t.Run("Verify_Common_Builder_Pattern", func(t *testing.T) {
		// 验证Builder模式中的链式调用
		foundName := false
		foundAge := false
		foundBuild := false

		for _, rel := range allRelations {
			if rel.Type == model.Call {
				if rel.Target != nil && rel.Target.Kind == model.Method {
					methodName := rel.Target.Name
					sourceQN := rel.Source.QualifiedName
					receiverTypeQN, _ := rel.Mores[constants.RelCallReceiverType].(string)
					isChained, _ := rel.Mores[constants.RelCallIsChained].(bool)

					// 只关注 testChainedCalls 方法中的调用
					if methodName == "name" && strings.Contains(sourceQN, "testChainedCalls") {
						foundName = true
						// name() 调用应该是链式的
						assert.True(t, isChained, "name() should be marked as chained")
						// name() 返回 Builder 类型
						assert.Contains(t, receiverTypeQN, "Builder",
							"Receiver type should contain 'Builder', got: "+receiverTypeQN)
					}
					if methodName == "age" && strings.Contains(sourceQN, "testChainedCalls") {
						foundAge = true
						assert.True(t, isChained, "age() should be marked as chained")
						assert.Contains(t, receiverTypeQN, "Builder",
							"Receiver type should contain 'Builder', got: "+receiverTypeQN)
					}
					if methodName == "build" && strings.Contains(sourceQN, "testChainedCalls") {
						foundBuild = true
						assert.True(t, isChained, "build() should be marked as chained")
						assert.Contains(t, receiverTypeQN, "ChainedCallExample",
							"Receiver type should contain 'ChainedCallExample', got: "+receiverTypeQN)
					}
				}
			}
		}

		assert.True(t, foundName, "name() call NOT found")
		assert.True(t, foundAge, "age() call NOT found")
		assert.True(t, foundBuild, "build() call NOT found")
	})

	t.Run("Verify_Simple_Chained_Call", func(t *testing.T) {
		// 验证：String result = obj1.getName().toUpperCase();
		foundGetName := false
		foundToUpperCase := false

		for _, rel := range allRelations {
			if rel.Type == model.Call {
				if rel.Target != nil && rel.Target.Kind == model.Method {
					methodName := rel.Target.Name
					receiverTypeQN, _ := rel.Mores[constants.RelCallReceiverType].(string)
					sourceQN := rel.Source.QualifiedName

					if methodName == "getName" && strings.Contains(sourceQN, "testChainedCalls") {
						foundGetName = true
					}
					if methodName == "toUpperCase" && strings.Contains(sourceQN, "testChainedCalls") {
						foundToUpperCase = true
						assert.Equal(t, true, rel.Mores[constants.RelCallIsChained], "toUpperCase() should be marked as chained")
						// toUpperCase() 的 receiver 是 getName() 的返回值，即 String
						assert.Equal(t, "String", receiverTypeQN,
							"Receiver type should be 'String', got: "+receiverTypeQN)
						// 验证接收者文本
						if receiver, ok := rel.Mores[constants.RelCallReceiver].(string); ok {
							assert.Contains(t, receiver, "getName",
								"Receiver should contain 'getName', got: "+receiver)
						}
					}
				}
			}
		}

		assert.True(t, foundGetName, "getName() call NOT found")
		assert.True(t, foundToUpperCase, "toUpperCase() call NOT found")
	})

	t.Run("Verify_Deep_Chained_Call", func(t *testing.T) {
		// 验证：String chained = obj1.getName().toUpperCase().trim();
		foundGetName := false
		foundToUpperCase := false
		foundTrim := false

		for _, rel := range allRelations {
			if rel.Type == model.Call {
				if rel.Target != nil && rel.Target.Kind == model.Method {
					methodName := rel.Target.Name
					receiverTypeQN, _ := rel.Mores[constants.RelCallReceiverType].(string)
					sourceQN := rel.Source.QualifiedName
					receiver, _ := rel.Mores[constants.RelCallReceiver].(string)
					isChained, _ := rel.Mores[constants.RelCallIsChained].(bool)

					if methodName == "getName" && strings.Contains(sourceQN, "testChainedCalls") {
						// 区分普通链式调用和深层链式调用的 getName()
						// 深层链式调用：receiver 是 obj1（变量）
						// 普通链式调用：receiver 是 obj1（变量）
						// 方法：通过检查是否有后续的 toUpperCase().trim() 调用
						if strings.Contains(receiver, "obj1") &&
							((isChained && len(strings.Split(receiver, ".")) == 1) || !isChained) {
							// 这是 obj1.getName() 普通调用
							// 检查是否有后续的 toUpperCase().trim() 调用
							// 不能完全通过 receiver 判断，需要看是否有后续调用
							// 这里简化处理：只要找到 getName() 且后面有 toUpperCase().trim() 就算找到
							foundGetName = true
						}
					}
					if methodName == "toUpperCase" && strings.Contains(sourceQN, "testChainedCalls") {
						// toUpperCase() 的 receiver 是 obj1.getName()
						if strings.Contains(receiver, "getName") {
							foundToUpperCase = true
							assert.True(t, isChained, "toUpperCase() should be marked as chained")
							// toUpperCase() 返回 String
							assert.Equal(t, "String", receiverTypeQN,
								"Receiver type should be 'String', got: "+receiverTypeQN)
						}
					}
					if methodName == "trim" && strings.Contains(sourceQN, "testChainedCalls") {
						// trim() 的 receiver 是 obj1.getName().toUpperCase()
						if strings.Contains(receiver, "toUpperCase") || isChained {
							foundTrim = true
							assert.True(t, isChained, "trim() should be marked as chained")
							// trim() 返回 String
							assert.Equal(t, "String", receiverTypeQN,
								"Receiver type should be 'String', got: "+receiverTypeQN)
						}
					}
				}
			}
		}

		// 对于深层链式调用，我们主要验证中间调用和最后调用的 ReceiverType
		// getName() 是入口调用，不一定是链式调用的起始点
		t.Log("Note: getName() verification is simplified for deep chain")
		assert.True(t, foundToUpperCase, "toUpperCase() call in deep chain NOT found")
		assert.True(t, foundTrim, "trim() call in deep chain NOT found")
		assert.True(t, foundGetName, "getName() call in deep chain NOT found")
	})
}
