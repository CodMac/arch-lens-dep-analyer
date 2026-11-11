package test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	cppctx "github.com/CodMac/arch-lens-dep-analyer/x/cpp/context"
)

// TestCollectorCppDisplayFieldsInvariant 对全部 C/C++ collector 样例断言展示层三字段的通用不变量。
// 覆盖现有 18+ 组 testdata 的全部元素，避免逐一手写断言遗漏。
func TestCollectorCppDisplayFieldsInvariant(t *testing.T) {
	paths := resolveCppTree(t, "collector/cpp")
	gc := RunCollection(t, "x/cpp/testdata", paths)

	fileContexts := gc.FileContexts()
	if len(fileContexts) == 0 {
		t.Fatal("no file contexts collected")
	}

	checked := 0
	for _, fc := range fileContexts {
		nsSet := make(map[string]bool)
		if nfc, ok := fc.(cppctx.NamespaceAware); ok {
			for _, ns := range nfc.Namespaces() {
				nsSet[ns] = true
			}
		}

		// 展示层 QN 索引：namespace 重开消歧后缀（$N）在展示层被剥离，
		// 故 TypeQN 的可见性/前缀校验须用归一化后的 QN。
		displayIndex := make(map[string]*core.DefinitionEntry)
		for _, entry := range fc.Definitions() {
			displayIndex[normalizeDisplayQNForTest(fc, nsSet, entry.Element.QualifiedName)] = entry
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

			// 不变量 2：NamespaceQN 为空，或指向一个真实的 namespace
			if elem.NamespaceQN != "" && !isNamespaceQNForTest(gc, fc, nsSet, elem.NamespaceQN) {
				t.Errorf("[2] %s NamespaceQN = %q 不是 namespace",
					elem.QualifiedName, elem.NamespaceQN)
			}

			// 不变量 3：EnclosingQN == ParentQN（父级是 namespace 时剥离重开消歧后缀 $N）
			wantEnc := entry.ParentQN
			if wantEnc != "" {
				if parent, ok := fc.FindByQualifiedName(wantEnc); ok && parent.Element.Kind == model.Namespace {
					wantEnc = stripNamespaceDisambigForTest(wantEnc)
				}
			}
			if elem.EnclosingQN != wantEnc {
				t.Errorf("[3] %s EnclosingQN = %q, want %q",
					elem.QualifiedName, elem.EnclosingQN, wantEnc)
			}

			// 不变量 4：TypeQN 为空，或指向存在的具名类型
			if elem.TypeQN != "" {
				target, ok := gc.FindByQualifiedName(elem.TypeQN)
				if !ok {
					target, ok = displayIndex[elem.TypeQN]
				}
				if !ok {
					t.Errorf("[4] %s TypeQN = %q 不在 GlobalContext 中",
						elem.QualifiedName, elem.TypeQN)
				} else if !isCppNamedTypeKindForTest(target.Element.Kind) {
					t.Errorf("[4] %s TypeQN = %q kind = %s，不是具名类型",
						elem.QualifiedName, elem.TypeQN, target.Element.Kind)
				}

				// 不变量 5：TypeQN 是元素展示层 QN 的段边界前缀
				displayQN := normalizeDisplayQNForTest(fc, nsSet, elem.QualifiedName)
				if elem.TypeQN != displayQN &&
					!strings.HasPrefix(displayQN, elem.TypeQN+"::") {
					t.Errorf("[5] %s TypeQN = %q 不是 QN 的段边界前缀",
						elem.QualifiedName, elem.TypeQN)
				}

				// 不变量 6：TypeQN 不得是函数/方法 QN（函数签名含括号）
				if strings.Contains(elem.TypeQN, "(") {
					t.Errorf("[6] %s TypeQN = %q 是函数 QN，不是类型",
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

// isCppNamedTypeKindForTest 复刻 Collector 的具名类型判定（测试侧独立定义）。
func isCppNamedTypeKindForTest(k model.ElementKind) bool {
	switch k {
	case model.Class, model.Struct, model.Enum:
		return true
	}
	return false
}

// isNamespaceQNForTest 判定 QN 是否指向 namespace（全局上下文或本文件 namespace 列表）。
func isNamespaceQNForTest(gc core.GlobalContext, fc core.FileContext, nsSet map[string]bool, qn string) bool {
	if nsSet[qn] {
		return true
	}
	if gc != nil {
		if e, ok := gc.FindByQualifiedName(qn); ok {
			return e.Element.Kind == model.Namespace
		}
	}
	if e, ok := fc.FindByQualifiedName(qn); ok {
		return e.Element.Kind == model.Namespace
	}
	return false
}

// normalizeDisplayQNForTest 复刻 Collector 的展示层 QN 归一化：
// namespace 段的重开消歧后缀（$N）被剥离。
func normalizeDisplayQNForTest(fc core.FileContext, nsSet map[string]bool, qn string) string {
	parts := strings.Split(qn, "::")
	raw := make([]string, 0, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		raw = append(raw, p)
		seg := p
		if isNamespaceQNForTest(nil, fc, nsSet, strings.Join(raw, "::")) {
			seg = stripNamespaceDisambigForTest(p)
		}
		out = append(out, seg)
	}
	return strings.Join(out, "::")
}

// stripNamespaceDisambigForTest 剥离 QN 末段的数字消歧后缀：ns$2 -> ns。
func stripNamespaceDisambigForTest(seg string) string {
	i := strings.LastIndex(seg, "$")
	if i <= 0 || i == len(seg)-1 {
		return seg
	}
	for _, c := range seg[i+1:] {
		if c < '0' || c > '9' {
			return seg
		}
	}
	return seg[:i]
}

// resolveCppTree 递归收集 testdata 子目录下全部 C/C++ 源文件（含嵌套子目录）。
func resolveCppTree(t *testing.T, dir string) []string {
	t.Helper()
	exts := map[string]bool{".c": true, ".h": true, ".cpp": true, ".hpp": true, ".cc": true, ".hh": true}
	var files []string
	err := filepath.Walk(filepath.Join(TestDataDir, dir), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !exts[filepath.Ext(path)] {
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
		t.Fatalf("no c/cpp files found under %s", dir)
	}
	return files
}
