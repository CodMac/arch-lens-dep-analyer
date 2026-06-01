package collector

import (
	"path/filepath"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

func TestJavaCollector_Imports(t *testing.T) {
	filePath := test.GetTestFilePath(filepath.Join("collector", "import", "ImportTest.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	test.PrintCodeElements(fCtx)

	// 1. 验证 Class 导入: import java.util.List;
	t.Run("Verify Specific Class Import", func(t *testing.T) {
		imps, exists := fCtx.Imports["List"]
		if !exists || len(imps) == 0 {
			t.Fatalf("Import 'List' not found")
		}
		// 取第一个（通常 List 在此类中只有一个导入）
		imp := imps[0]
		if imp.RawImportPath != "java.util.List" {
			t.Errorf("Expected java.util.List, got %s", imp.RawImportPath)
		}
		if imp.Kind != model.Class {
			t.Errorf("Expected Kind Class, got %s", imp.Kind)
		}
	})

	// 2. 验证通配符导入 (重点：验证 slice 中包含两个通配符)
	t.Run("Verify Multiple Wildcard Imports", func(t *testing.T) {
		imps, exists := fCtx.Imports["*"]
		if !exists || len(imps) < 2 {
			t.Fatalf("Expected at least 2 wildcard imports, found %d", len(imps))
		}

		paths := make(map[string]bool)
		for _, imp := range imps {
			paths[imp.RawImportPath] = true
		}

		// 验证 java.util.*
		if !paths["java.util.*"] {
			t.Errorf("Wildcard import java.util.* not found in %v", paths)
		}
		// 验证 static java.lang.Math.*
		if !paths["java.lang.Math.*"] {
			t.Errorf("Static wildcard import java.lang.Math.* not found in %v", paths)
		}
	})

	// 3. 验证具体静态方法导入: import static java.util.Collections.sort;
	t.Run("Verify Static Method Import", func(t *testing.T) {
		imps, exists := fCtx.Imports["sort"]
		if !exists || len(imps) == 0 {
			t.Fatalf("Static import 'sort' not found")
		}
		imp := imps[0]
		if imp.RawImportPath != "java.util.Collections.sort" {
			t.Errorf("Expected java.util.Collections.sort, got %s", imp.RawImportPath)
		}
		// 静态非通配符导入，handleImport 标记为 Constant
		if imp.Kind != model.Constant {
			t.Errorf("Expected Kind Constant, got %s", imp.Kind)
		}
	})
}
