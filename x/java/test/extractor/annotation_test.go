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

func TestJavaExtractor_Annotation(t *testing.T) {
	testFile := test.GetTestFilePath(filepath.Join("extractor", "annotation", "AnnotationRelationSuite.java"))
	files := []string{testFile}

	gCtx := test.RunPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Annotation})

	expectedRels := []struct {
		relType    model.DependencyType
		sourceQN   string
		targetQN   string
		targetKind model.ElementKind
		checkMores func(t *testing.T, mores map[string]interface{})
	}{
		// --- 1. 类注解 ---
		{
			relType:    model.Annotation,
			sourceQN:   "com.example.rel.AnnotationRelationSuite",
			targetQN:   "Entity",
			targetKind: model.KAnnotation,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "TYPE", m[constants.RelAnnotationTarget])
			},
		},
		{
			relType:    model.Annotation,
			sourceQN:   "com.example.rel.AnnotationRelationSuite",
			targetQN:   "SuppressWarnings",
			targetKind: model.KAnnotation,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "TYPE", m[constants.RelAnnotationTarget])
			},
		},
		// --- 2. 字段注解 ---
		{
			relType:    model.Annotation,
			sourceQN:   "com.example.rel.AnnotationRelationSuite.id",
			targetQN:   "Id",
			targetKind: model.KAnnotation,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "FIELD", m[constants.RelAnnotationTarget])
			},
		},
		// --- 3. 方法注解 ---
		{
			relType:    model.Annotation,
			sourceQN:   "com.example.rel.AnnotationRelationSuite.save(String)",
			targetQN:   "Transactional",
			targetKind: model.KAnnotation,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "METHOD", m[constants.RelAnnotationTarget])
				// 注意：RelAnnotationParams 已移至 Extended，此处不再断言
			},
		},
		// --- 4. 局部变量注解 ---
		{
			relType:    model.Annotation,
			sourceQN:   "com.example.rel.AnnotationRelationSuite.save(String).local",
			targetQN:   "NonEmpty",
			targetKind: model.KAnnotation,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "LOCAL_VARIABLE", m[constants.RelAnnotationTarget])
			},
		},
	}

	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			if rel.Type == exp.relType &&
				rel.Target.Name == exp.targetQN &&
				strings.HasSuffix(rel.Source.QualifiedName, exp.sourceQN) {

				found = true
				assert.Equal(t, exp.targetKind, rel.Target.Kind)
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing expected relation: [%s] %s -> %s", exp.relType, exp.sourceQN, exp.targetQN)
	}
}
