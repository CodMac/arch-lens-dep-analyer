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

func TestJavaExtractor_Create(t *testing.T) {
	testFile := test.GetTestFilePath(filepath.Join("extractor", "create", "CreateRelationSuite.java"))
	files := []string{testFile}

	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Create})

	expectedRels := []ExpectedCase{
		// 1. 成员变量声明时实例化 (有 Import，保持全称)
		{
			sourceMatch: "com.example.rel.CreateRelationSuite.fieldInstance",
			targetMatch: "java.util.ArrayList",
			lineNum:     14,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "fieldInstance", m[constants.RelCreateVariableName])
			},
		},
		// 2. 静态成员变量实例化
		{
			sourceMatch: "com.example.rel.CreateRelationSuite.staticMap",
			targetMatch: "java.util.HashMap",
			lineNum:     20,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "staticMap", m[constants.RelCreateVariableName])
			},
		},
		// 3. 局部变量实例化
		{
			sourceMatch: "com.example.rel.CreateRelationSuite.testCreateCases()",
			targetMatch: "StringBuilder",
			lineNum:     27,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "sb", m[constants.RelCreateVariableName])
				assert.Equal(t, "new StringBuilder(\"init\")", m[constants.RelRawText])
			},
		},
		// 4. 匿名内部类 (无 Import，保持简写)
		{
			sourceMatch: "com.example.rel.CreateRelationSuite.testCreateCases()",
			targetMatch: "Runnable",
			lineNum:     33,
			checkMores:  func(t *testing.T, m map[string]interface{}) {},
		},
		// 5. 数组实例化
		{
			sourceMatch: "com.example.rel.CreateRelationSuite.testCreateCases()",
			targetMatch: "String",
			lineNum:     44,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "strings", m[constants.RelCreateVariableName])
				assert.Equal(t, "new String[5]", m[constants.RelRawText])
			},
		},
		// 6. 链式调用中的实例化
		{
			sourceMatch: "com.example.rel.CreateRelationSuite.testCreateCases()",
			targetMatch: "com.example.rel.CreateRelationSuite",
			lineNum:     50,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "new CreateRelationSuite()", m[constants.RelRawText])
			},
		},
		// 7. 构造函数内部实例化 (super)
		{
			sourceMatch: "com.example.rel.CreateRelationSuite.CreateRelationSuite()",
			targetMatch: "Object",
			lineNum:     58,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "super();", m[constants.RelRawText])
			},
		},
	}

	RunCases(t, expectedRels, allRelations, model.Create)
}
