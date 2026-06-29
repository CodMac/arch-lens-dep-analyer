package extractor

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
	"github.com/stretchr/testify/assert"
)

func TestJavaExtractor_Extend(t *testing.T) {
	testFile := test.GetTestFilePath(filepath.Join("extractor", "extend", "ExtendRelationSuite.java"))
	files := []string{testFile}

	// 假设 test.RunPhase1Collection 已经处理了 Collector 阶段
	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Extend})

	expectedRels := []struct {
		relType    model.DependencyType
		sourceQN   string // 匹配后缀
		targetQN   string // 匹配目标名
		targetKind model.ElementKind
		targetFull string // 预期的完整 QN (用于验证 Import 解析)
	}{
		// --- 1. 类继承 ---
		{
			relType:    model.Extend,
			sourceQN:   "com.example.rel.ExtendRelationSuite",
			targetQN:   "ArrayList",
			targetKind: model.Class,           // 外部类，默认都是Class，不去猜测修正
			targetFull: "java.util.ArrayList", // 验证 clean() 擦除了 <String> 并通过 import 补全
		},
		// --- 2. 接口继承接口 ---
		{
			relType:    model.Extend,
			sourceQN:   "com.example.rel.ExtendRelationSuite.SubInterface",
			targetQN:   "Runnable",
			targetKind: model.Interface, // 外部类，默认都是Class，不去猜测修正
			targetFull: "Runnable",
		},
		{
			relType:    model.Extend,
			sourceQN:   "com.example.rel.ExtendRelationSuite.SubInterface",
			targetQN:   "Serializable",
			targetKind: model.Interface, // 外部类，默认都是Class，不去猜测修正
			targetFull: "java.io.Serializable",
		},
		// --- 3. 匿名类继承 ---
		{
			relType:    model.Extend,
			sourceQN:   "anonymousClass$1", // 匹配 Collector 生成的匿名类名
			targetQN:   "Runnable",
			targetKind: model.Class, // 外部类，默认都是Class，不去猜测修正
			targetFull: "Runnable",
		},
	}

	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 匹配关系类型
			if rel.Type != exp.relType {
				continue
			}
			// 匹配 Source (支持 QN 后缀匹配)
			if !strings.HasSuffix(rel.Source.QualifiedName, exp.sourceQN) {
				continue
			}
			// 匹配 Target
			if rel.Target.Name == exp.targetQN {
				found = true
				assert.Equal(t, exp.targetKind, rel.Target.Kind, "Kind mismatch for target %s", exp.targetQN)
				assert.Equal(t, exp.targetFull, rel.Target.QualifiedName, "QualifiedName resolution failed for %s", exp.targetQN)
				break
			}
		}
		assert.True(t, found, "Missing expected relation: [%s] %s -> %s", exp.relType, exp.sourceQN, exp.targetQN)
	}
}
