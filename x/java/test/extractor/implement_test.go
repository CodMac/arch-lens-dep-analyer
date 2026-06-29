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

func TestJavaExtractor_Implement(t *testing.T) {
	testFile := test.GetTestFilePath(filepath.Join("extractor", "implement", "ImplementRelationSuite.java"))
	files := []string{testFile}

	// 运行第一阶段采集以填充 GlobalContext
	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Implement})

	basePkg := "com.example.rel"

	expectedRels := []struct {
		relType    model.DependencyType
		sourceQN   string
		targetQN   string
		targetKind model.ElementKind
	}{
		// --- 1. 接口继承接口 (BaseApi extends Serializable) ---
		{
			relType:    model.Extend,
			sourceQN:   basePkg + ".BaseApi",
			targetQN:   "Serializable",
			targetKind: model.Interface, // 继承接口在模型中通常映射为 Implement 关系
		},
		// --- 2. 多接口实现 (MultiImpl implements BaseApi, Runnable, SingleInterface) ---
		{
			relType:    model.Implement,
			sourceQN:   basePkg + ".MultiImpl",
			targetQN:   "BaseApi",
			targetKind: model.Interface,
		},
		{
			relType:    model.Implement,
			sourceQN:   basePkg + ".MultiImpl",
			targetQN:   "Runnable",
			targetKind: model.Interface,
		},
		{
			relType:    model.Implement,
			sourceQN:   basePkg + ".MultiImpl",
			targetQN:   "SingleInterface",
			targetKind: model.Interface,
		},
		// --- 3. 抽象类实现接口 (AbstractTask implements BaseApi) ---
		{
			relType:    model.Implement,
			sourceQN:   basePkg + ".AbstractTask",
			targetQN:   "BaseApi",
			targetKind: model.Interface,
		},
		// --- 4. 匿名内部类实现 (new Runnable() { ... }) ---
		{
			relType:    model.Extend,
			sourceQN:   basePkg + ".ImplementRelationSuite.test().anonymousClass$1", // 匹配你 Collector 中的匿名类命名规则
			targetQN:   "Runnable",
			targetKind: model.Class,
		},
	}

	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 使用 HasSuffix 匹配 QN，确保包名路径正确
			if rel.Type == exp.relType &&
				rel.Target.Name == exp.targetQN &&
				strings.HasSuffix(rel.Source.QualifiedName, exp.sourceQN) {

				found = true
				assert.Equal(t, exp.targetKind, rel.Target.Kind, "Kind mismatch for target %s", exp.targetQN)
				break
			}
		}
		assert.True(t, found, "Missing expected Implement relation: %s -> %s", exp.sourceQN, exp.targetQN)
	}
}
