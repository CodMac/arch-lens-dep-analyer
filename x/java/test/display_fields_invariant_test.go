package test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// TestCollectorDisplayFieldsInvariant 对全部 collector 样例断言展示层三字段的通用不变量。
// 覆盖现有 15+ 组 testdata 的全部元素，避免逐一手写断言遗漏。
func TestCollectorDisplayFieldsInvariant(t *testing.T) {
	paths := resolveJavaTree(t, "collector")
	gc := RunCollection(t, "x/java/testdata", paths)

	fileContexts := gc.FileContexts()
	if len(fileContexts) == 0 {
		t.Fatal("no file contexts collected")
	}

	checked := 0
	for _, fc := range fileContexts {
		pkg := ""
		if pfc, ok := fc.(core.PackageAwareFileContext); ok {
			pkg = pfc.PackageName()
		}
		for _, entry := range fc.Definitions() {
			elem := entry.Element
			checked++

			// 不变量 1：File 元素三字段全空
			if elem.Kind == model.File {
				if elem.NamespaceQN != "" || elem.EnclosingQN != "" || elem.TypeQN != "" {
					t.Errorf("[1] File %s 三字段应为空，got NS=%q ENC=%q TYPE=%q",
						elem.QualifiedName, elem.NamespaceQN, elem.EnclosingQN, elem.TypeQN)
				}
				continue
			}

			// 不变量 2：NamespaceQN == 文件 package
			if elem.NamespaceQN != pkg {
				t.Errorf("[2] %s NamespaceQN = %q, want 文件 package %q",
					elem.QualifiedName, elem.NamespaceQN, pkg)
			}

			// 不变量 3：EnclosingQN == ParentQN
			if elem.EnclosingQN != entry.ParentQN {
				t.Errorf("[3] %s EnclosingQN = %q, want ParentQN %q",
					elem.QualifiedName, elem.EnclosingQN, entry.ParentQN)
			}

			// 不变量 4：TypeQN 为空，或指向存在的具名类型
			if elem.TypeQN != "" {
				target, ok := gc.FindByQualifiedName(elem.TypeQN)
				if !ok {
					t.Errorf("[4] %s TypeQN = %q 不在 GlobalContext 中",
						elem.QualifiedName, elem.TypeQN)
				} else if !isNamedTypeKindForTest(target.Element.Kind) {
					t.Errorf("[4] %s TypeQN = %q kind = %s，不是具名类型",
						elem.QualifiedName, elem.TypeQN, target.Element.Kind)
				}

				// 不变量 5：TypeQN 是元素 QN 的段边界前缀
				if elem.TypeQN != elem.QualifiedName &&
					!strings.HasPrefix(elem.QualifiedName, elem.TypeQN+".") {
					t.Errorf("[5] %s TypeQN = %q 不是 QN 的段边界前缀",
						elem.QualifiedName, elem.TypeQN)
				}
			}
		}
	}

	t.Logf("checked %d elements across %d files", checked, len(fileContexts))
	if checked < 500 {
		t.Errorf("only %d elements checked, expected a broad sweep (>=500)", checked)
	}
}

// isNamedTypeKindForTest 复刻 Collector 的具名类型判定（测试侧独立定义，避免依赖内部函数）。
func isNamedTypeKindForTest(k model.ElementKind) bool {
	switch k {
	case model.Class, model.Interface, model.Enum, model.KAnnotation:
		return true
	}
	return false
}

// resolveJavaTree 递归收集 testdata 子目录下全部 .java 文件（含嵌套子目录）。
func resolveJavaTree(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	err := filepath.Walk(filepath.Join(TestDataDir, dir), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".java" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatalf("no java files found under %s", dir)
	}
	return files
}
