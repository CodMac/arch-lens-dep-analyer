package collector

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

func TestJavaCollector_Annotation(t *testing.T) {
	// 1. 获取测试文件路径
	filePath := test.GetTestFilePath(filepath.Join("collector", "annotation", "Loggable.java"))

	// 2. 解析源码与运行 Collector
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, false)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	test.PrintCodeElements(fCtx)

	// 验证 1: Annotation Type Declaration & 注释提取
	t.Run("Verify Annotation Declaration and Doc", func(t *testing.T) {
		qn := "com.example.base.annotation.Loggable"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Annotation Loggable not found with QN: %s", qn)
		}

		elem := defs[0].Element
		// 验证注释提取 (Doc)
		if !strings.Contains(elem.Doc, "Annotation Type Declaration") || !strings.Contains(elem.Doc, "Meta-Annotations") {
			t.Errorf("Doc comment not correctly extracted, got: %s", elem.Doc)
		}

		// 验证元注解 (Meta-Annotations)
		annos := elem.Extra.Annotations
		hasRetention := false
		hasTarget := false
		for _, a := range annos {
			if strings.Contains(a, "@Retention") {
				hasRetention = true
			}
			if strings.Contains(a, "@Target") {
				hasTarget = true
			}
		}
		if !hasRetention || !hasTarget {
			t.Errorf("Missing meta-annotations. Found: %v", annos)
		}
	})

	// 验证 2: 语义化 Import ("*" 通配符)
	t.Run("Verify Wildcard Import", func(t *testing.T) {
		// 在 map[string][]*ImportEntry 中，通配符导入的 key 通常是 "*"
		imports, ok := fCtx.Imports["*"]
		if !ok || len(imports) == 0 {
			t.Fatalf("Wildcard imports not found in FileContext")
		}

		found := false
		for _, imp := range imports {
			if imp.RawImportPath == "java.lang.annotation.*" && imp.IsWildcard {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected wildcard import 'java.lang.annotation.*' not found under key '*'")
		}
	})

	// 验证 3: 注解的函数定义、默认返回值及特殊属性
	t.Run("Verify Annotation Members", func(t *testing.T) {
		// 调整：注解成员在 QN 中不带括号，因为它们不是真正的 method_declaration
		levelQN := "com.example.base.annotation.Loggable.level()"
		levelDefs := test.FindDefinitionsByQN(fCtx, levelQN)
		if len(levelDefs) == 0 {
			t.Fatalf("Annotation member level not found with QN: %s", levelQN)
		}

		levelElem := levelDefs[0].Element
		if isAnno, _ := levelElem.Extra.Mores[java.MethodIsAnnotation].(bool); !isAnno {
			t.Errorf("level should have MethodIsAnnotation = true")
		}
		if defVal := levelElem.Extra.Mores[java.MethodDefaultValue]; defVal != "\"INFO\"" {
			t.Errorf("Expected default value \"INFO\", got %v", defVal)
		}

		// 验证 trace 及其默认值
		traceQN := "com.example.base.annotation.Loggable.trace()"
		traceDefs := test.FindDefinitionsByQN(fCtx, traceQN)
		if len(traceDefs) == 0 {
			t.Fatalf("Annotation member trace not found")
		}

		traceElem := traceDefs[0].Element
		if defVal := traceElem.Extra.Mores[java.MethodDefaultValue]; defVal != "false" {
			t.Errorf("Expected default value false, got %v", traceElem.Extra.Mores[java.MethodDefaultValue])
		}
	})
}
