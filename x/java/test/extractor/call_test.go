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
	expectedRels := []ExpectedCase{
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll()",
			targetMatch: "com.example.rel.CallRelationSuite.simpleMethod()",
			lineNum:     21,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, false, m[constants.RelCallIsStatic])
				assert.Equal(t, "this", m[constants.RelCallReceiver])
			},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll()",
			targetMatch: "com.example.rel.CallRelationSuite.staticMethod()",
			lineNum:     31,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelCallIsStatic])
				assert.Equal(t, m[constants.RelCallReceiver], "CallRelationSuite")
				assert.Equal(t, m[constants.RelCallReceiverType], "com.example.rel.CallRelationSuite")
			},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll()",
			targetMatch: "System.currentTimeMillis()",
			lineNum:     41,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "System", m[constants.RelCallReceiver])
				assert.Equal(t, true, m[constants.RelCallIsStatic])
			},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll()",
			targetMatch: "com.example.rel.CallRelationSuite.getList()",
			lineNum:     50,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelCallIsChained])
			},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll()",
			targetMatch: "java.util.List.add()",
			lineNum:     50,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelCallIsChained])
			},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll()",
			targetMatch: "com.example.rel.BaseClass.baseMethod()",
			lineNum:     60,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelCallIsChained])
				assert.Equal(t, "super", m[constants.RelCallReceiver])
			},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll()",
			targetMatch: "com.example.rel.BaseClass.baseMethod()",
			lineNum:     69,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "this", m[constants.RelCallReceiver])
			},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll()",
			targetMatch: "java.util.ArrayList.ArrayList()",
			lineNum:     78,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelCallIsConstructor])
			},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll().lambda$1",
			targetMatch: "com.example.rel.CallRelationSuite.simpleMethod()",
			lineNum:     89,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "com.example.rel.CallRelationSuite.executeAll()", m[constants.RelCallEnclosingMethod])
				assert.Equal(t, "this", m[constants.RelCallReceiver])
			},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll()",
			targetMatch: "java.util.List.of()",
			lineNum:     100,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "List", m[constants.RelCallReceiver])
				assert.Equal(t, "java.util.List", m[constants.RelCallReceiverType])
			},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll()",
			targetMatch: "com.example.rel.CallRelationSuite.simpleMethod()",
			lineNum:     100,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "this", m[constants.RelCallReceiver])
				assert.Equal(t, "com.example.rel.CallRelationSuite", m[constants.RelCallReceiverType])
				assert.Equal(t, true, m[constants.RelCallIsFunctional])

			},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll()",
			targetMatch: "com.example.rel.CallRelationSuite.genericMethod(T)",
			lineNum:     109,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "this", m[constants.RelCallReceiver])
				assert.Equal(t, "com.example.rel.CallRelationSuite", m[constants.RelCallReceiverType])
			},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll().anonymousClass$1.run()",
			targetMatch: "com.example.rel.CallRelationSuite.simpleMethod()",
			lineNum:     121,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "com.example.rel.CallRelationSuite.executeAll()", m[constants.RelCallEnclosingMethod])
			},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll()",
			targetMatch: "com.example.rel.CallRelationSuite.customLog(String, Object...)",
			lineNum:     127,
			checkMores:  func(t *testing.T, m map[string]interface{}) {},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll()",
			targetMatch: "com.example.rel.CallRelationSuite.CallRelationSuite()",
			lineNum:     130,
			checkMores:  func(t *testing.T, m map[string]interface{}) {},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll()",
			targetMatch: "com.example.rel.CallRelationSuite.simpleMethod()",
			lineNum:     133,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "obj", m[constants.RelCallReceiver])
			},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll()",
			targetMatch: "RuntimeException()",
			lineNum:     139,
			checkMores:  func(t *testing.T, m map[string]interface{}) {},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.executeAll()",
			targetMatch: "com.example.rel.CallRelationSuite.MyEnum.values()",
			lineNum:     145,
			checkMores:  func(t *testing.T, m map[string]interface{}) {},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.getList()",
			targetMatch: "java.util.ArrayList.ArrayList()",
			lineNum:     151,
			checkMores:  func(t *testing.T, m map[string]interface{}) {},
		},
		{
			sourceMatch: "com.example.rel.CallRelationSuite.SubClass.SubClass()",
			targetMatch: "com.example.rel.BaseClass.BaseClass()",
			lineNum:     160,
			checkMores:  func(t *testing.T, m map[string]interface{}) {},
		},
	}

	// 3. 执行断言
	RunCases(t, expectedRels, allRelations, model.Call)
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

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Call})

	// 2. 定义断言数据集
	expectedRels := []ExpectedCase{
		{
			sourceMatch: "com.example.chained.ChainedCallExample.Builder.build()",
			targetMatch: "com.example.chained.ChainedCallExample.ChainedCallExample(Builder)",
			lineNum:     20,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelCallIsConstructor])
			},
		},
		{
			sourceMatch: "com.example.chained.ChainedCallExample.testChainedCalls()",
			targetMatch: "com.example.chained.ChainedCallExample.Builder.Builder()",
			lineNum:     38,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[constants.RelCallIsConstructor])
			},
		},
		{
			sourceMatch: "com.example.chained.ChainedCallExample.testChainedCalls()",
			targetMatch: "com.example.chained.ChainedCallExample.Builder.name(String)",
			lineNum:     39,
			checkMores:  func(t *testing.T, m map[string]interface{}) {},
		},
		{
			sourceMatch: "com.example.chained.ChainedCallExample.testChainedCalls()",
			targetMatch: "com.example.chained.ChainedCallExample.Builder.age(int)",
			lineNum:     40,
			checkMores:  func(t *testing.T, m map[string]interface{}) {},
		},
		{
			sourceMatch: "com.example.chained.ChainedCallExample.testChainedCalls()",
			targetMatch: "com.example.chained.ChainedCallExample.Builder.build()",
			lineNum:     41,
			checkMores:  func(t *testing.T, m map[string]interface{}) {},
		},
		{
			sourceMatch: "com.example.chained.ChainedCallExample.testChainedCalls()",
			targetMatch: "com.example.chained.ChainedCallExample.getName()",
			lineNum:     44,
			checkMores:  func(t *testing.T, m map[string]interface{}) {},
		},
		{
			sourceMatch: "com.example.chained.ChainedCallExample.testChainedCalls()",
			targetMatch: "String.toUpperCase()",
			lineNum:     44,
			checkMores:  func(t *testing.T, m map[string]interface{}) {},
		},
		{
			sourceMatch: "com.example.chained.ChainedCallExample.testChainedCalls()",
			targetMatch: "com.example.chained.ChainedCallExample.getName()",
			lineNum:     47,
			checkMores:  func(t *testing.T, m map[string]interface{}) {},
		},
		{
			sourceMatch: "com.example.chained.ChainedCallExample.testChainedCalls()",
			targetMatch: "String.toUpperCase()",
			lineNum:     47,
			checkMores:  func(t *testing.T, m map[string]interface{}) {},
		},
		{
			sourceMatch: "com.example.chained.ChainedCallExample.testChainedCalls()",
			targetMatch: "trim()", // 由于 `obj1.getName().toUpperCase()` 中的 `toUpperCase()` 非源码，无法得到返回值。这里只能默认返回兜底方法
			lineNum:     47,
			checkMores:  func(t *testing.T, m map[string]interface{}) {},
		},
	}

	// 3. 执行断言
	RunCases(t, expectedRels, allRelations, model.Call)
}
