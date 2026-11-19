package test

import (
	"sort"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/constants"
)

// =============================================================================
// 共享断言与收集辅助：供 base / C / C++ 三组测试文件复用。
// =============================================================================

// coreGlobalContext 是 core.GlobalContext 的别名（测试辅助签名用）。
type coreGlobalContext = core.GlobalContext

// collectC 收集指定 C 场景目录下的全部 .c/.h 文件。
func collectC(t *testing.T, group string) core.GlobalContext {
	t.Helper()
	paths := ResolvePaths(t, "collector/c/"+group, "*.c", "*.h")
	return RunCollection(t, "x/cpp/testdata", paths)
}

// collectCpp 收集指定 C++ 场景目录下的全部源文件。
func collectCpp(t *testing.T, group string) core.GlobalContext {
	t.Helper()
	paths := ResolvePaths(t, "collector/cpp/"+group, "*.c", "*.h", "*.cpp", "*.hpp")
	return RunCollection(t, "x/cpp/testdata", paths)
}

// AssertMods 断言元素携带指定修饰符（如 static/inline/extern）。
func AssertMods(t *testing.T, gc core.GlobalContext, qn, mod string) {
	t.Helper()
	elem := AssertDef(t, gc, qn, "")
	if elem == nil {
		return
	}
	if elem.Extra == nil || !containsStr(elem.Extra.Modifiers, mod) {
		var got []string
		if elem.Extra != nil {
			got = elem.Extra.Modifiers
		}
		t.Errorf("%s modifiers = %v, want contains %q", qn, got, mod)
	}
}

// AssertNoMores 断言元素不带指定 Mores 键（或值为 false 视为未标记）。
func AssertNoMores(t *testing.T, gc core.GlobalContext, qn, key string) {
	t.Helper()
	elem := AssertDef(t, gc, qn, "")
	if elem == nil {
		return
	}
	if v, ok := elem.GetMores()[key]; ok {
		if b, isBool := v.(bool); isBool && !b {
			return
		}
		t.Errorf("%s Mores[%s] should be absent, got %v", qn, key, v)
	}
}

// AssertNoKind 断言指定 kind 下不存在该 QN 的元素。
func AssertNoKind(t *testing.T, gc core.GlobalContext, qn string, kind model.ElementKind) {
	t.Helper()
	for _, def := range gc.Definitions() {
		if def.Element.QualifiedName == qn && def.Element.Kind == kind {
			t.Errorf("%s should NOT exist as kind %s", qn, kind)
			return
		}
	}
}

// AssertFieldsOf 断言某承载实体的字段名集合完全匹配（排序后比较）。
func AssertFieldsOf(t *testing.T, gc core.GlobalContext, parentQN string, want []string) {
	t.Helper()
	var got []string
	for _, def := range gc.Definitions() {
		if def.ParentQN == parentQN && def.Element.Kind == model.Field {
			got = append(got, def.Element.Name)
		}
	}
	sort.Strings(got)
	sortedWant := append([]string(nil), want...)
	sort.Strings(sortedWant)
	if !moresEqual(got, sortedWant) {
		t.Errorf("%s fields = %v, want %v", parentQN, got, sortedWant)
	}
}

// AssertSignature 断言函数元素的参数签名 Mores（类型 + 参数名列表）。
func AssertSignature(t *testing.T, gc core.GlobalContext, qn string, want []string) {
	t.Helper()
	AssertMores(t, gc, qn, constants.FunctionParameters, want)
}

// assertElemInFile 在指定文件上下文中按 QN + kind 查找元素（声明/定义共存时按 kind 区分）。
func assertElemInFile(t *testing.T, gc core.GlobalContext, filePath, qn string, kind model.ElementKind) *model.CodeElement {
	t.Helper()
	fc, ok := gc.FileContext(filePath)
	if !ok {
		t.Fatalf("file context not found: %s", filePath)
	}
	for _, def := range fc.Definitions() {
		if def.Element.QualifiedName == qn && def.Element.Kind == kind {
			return def.Element
		}
	}
	return nil
}

// AssertMoresInFile 断言指定文件上下文中某元素的 Mores。
func AssertMoresInFile(t *testing.T, gc core.GlobalContext, filePath, qn string, kind model.ElementKind, key string, want interface{}) {
	t.Helper()
	elem := assertElemInFile(t, gc, filePath, qn, kind)
	if elem == nil {
		t.Errorf("%s (kind=%s) not found in %s", qn, kind, filePath)
		return
	}
	got, exists := elem.GetMores()[key]
	if !exists {
		t.Errorf("%s Mores[%s] missing in %s", qn, key, filePath)
		return
	}
	if want != nil && !moresEqual(got, want) {
		t.Errorf("%s Mores[%s] = %v, want %v", qn, key, got, want)
	}
}

// findDefInFile 在指定文件上下文中按 QN + kind 查找定义（同文件声明/定义共存时按 kind 区分）。
func findDefInFile(gc coreGlobalContext, filePath, qn string, kind model.ElementKind) *model.CodeElement {
	fc, ok := gc.FileContext(filePath)
	if !ok {
		return nil
	}
	for _, def := range fc.Definitions() {
		if def.Element.QualifiedName == qn && (kind == "" || def.Element.Kind == kind) {
			return def.Element
		}
	}
	return nil
}

// assertDeclInFile 在指定文件上下文中断言声明元素存在。
func assertDeclInFile(t *testing.T, gc coreGlobalContext, filePath, qn string, wantKind model.ElementKind) {
	t.Helper()
	if findDefInFile(gc, filePath, qn, wantKind) == nil {
		t.Errorf("%s declaration (kind=%s) not found in %s", qn, wantKind, filePath)
	}
}

// assertDeclMores 在指定文件上下文中断言声明元素的 Mores。
func assertDeclMores(t *testing.T, gc coreGlobalContext, filePath, qn, key string, want interface{}) {
	t.Helper()
	elem := findDefInFile(gc, filePath, qn, model.MethodDecl)
	if elem == nil {
		t.Errorf("declaration not found in %s: %s", filePath, qn)
		return
	}
	got, exists := elem.GetMores()[key]
	if !exists {
		t.Errorf("%s Mores[%s] missing", qn, key)
		return
	}
	if !moresEqual(got, want) {
		t.Errorf("%s Mores[%s] = %v, want %v", qn, key, got, want)
	}
}

func isTrue(v interface{}) bool {
	b, _ := v.(bool)
	return b
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
