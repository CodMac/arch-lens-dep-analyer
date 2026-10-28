package helper

import (
	"slices"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// IsScopeContainer 是否容器 kind。
func IsScopeContainer(k model.ElementKind) bool {
	switch k {
	case model.Class, model.Interface, model.Enum, model.KAnnotation,
		model.Method, model.Lambda, model.ScopeBlock, model.AnonymousClass:
		return true
	}
	return false
}

// PreciseResolve 根据 symbol 查找符号定义（4 级检索）。
// v2 中 GlobalContext 内部自加锁，此处不再持锁（避免嵌套锁死锁）。
func PreciseResolve(gc *core.GlobalContext, fc *core.FileContext, symbol string) []*core.DefinitionEntry {
	// 短符号，文件内查找
	if defs, ok := fc.FindByShortName(symbol); ok {
		return defs
	}

	// 短符号，升级为确切 QN，全局查找
	if imps, ok := fc.Imports[symbol]; ok {
		for _, imp := range imps {
			if def, found := gc.FindByQualifiedName(imp.RawImportPath); found {
				return []*core.DefinitionEntry{def}
			}
		}
	}

	// 短符号，包内查找
	pkgQN := gc.BuildQualifiedName(fc.PackageName, symbol)
	if def, ok := gc.FindByQualifiedName(pkgQN); ok {
		return []*core.DefinitionEntry{def}
	}

	// 短符号，升级为匹配式 QN，全局查找（通配符导入）
	for _, imps := range fc.Imports {
		for _, imp := range imps {
			if imp.IsWildcard {
				basePath := strings.TrimSuffix(imp.RawImportPath, "*")
				if def, ok := gc.FindByQualifiedName(basePath + symbol); ok {
					return []*core.DefinitionEntry{def}
				}
			}
		}
	}
	if def, ok := gc.FindByQualifiedName(symbol); ok {
		return []*core.DefinitionEntry{def}
	}

	return nil
}

// GetBestElement 根据 node 所在行与 kinds，匹配合适的 Element。
func GetBestElement(fc *core.FileContext, node coreast.Node, kinds []model.ElementKind) *model.CodeElement {
	if node == nil {
		return nil
	}
	var best *model.CodeElement
	var minSize = 1 << 30
	row := int(node.StartPoint().Row + 1)
	for _, entry := range fc.Definitions {
		if slices.Contains(kinds, entry.Element.Kind) {
			loc := entry.Element.Location
			if loc != nil && row >= loc.StartLine && row <= loc.EndLine {
				size := loc.EndLine - loc.StartLine
				if size < minSize {
					minSize, best = size, entry.Element
				}
			}
		}
	}
	return best
}

// GetRealPackage 获得 Element 的包名。
func GetRealPackage(gc *core.GlobalContext, elem *model.CodeElement) string {
	curr := elem.QualifiedName
	for {
		idx := strings.LastIndex(curr, ".")
		if idx == -1 {
			return ""
		}
		curr = curr[:idx]

		if entry, ok := gc.FindByQualifiedName(curr); ok {
			if entry.Element.Kind == model.Package {
				return curr
			}
		} else {
			continue
		}
	}
}

// GetOwnerClassQN 获得 Element 的持有类 QN。
func GetOwnerClassQN(gc *core.GlobalContext, elem *model.CodeElement) string {
	curr := elem
	for curr != nil {
		if curr.Kind == model.Class || curr.Kind == model.Interface || curr.Kind == model.AnonymousClass {
			return curr.QualifiedName
		}
		if entry, ok := gc.FindByQualifiedName(curr.QualifiedName); ok && entry.ParentQN != "" {
			if next, ok := gc.FindByQualifiedName(entry.ParentQN); ok {
				curr = next.Element
				continue
			}
		}
		break
	}
	return ""
}

// GetOwnerClass 获得 Element 的持有类 CodeElement。
func GetOwnerClass(gc *core.GlobalContext, elem *model.CodeElement) *model.CodeElement {
	qn := GetOwnerClassQN(gc, elem)
	if qn == "" {
		return nil
	}
	if entry, ok := gc.FindByQualifiedName(qn); ok {
		return entry.Element
	}
	return nil
}

// GetOutermostClassQN 获取最外层的类名（例如把 A.B.C$1 还原为 A）。
func GetOutermostClassQN(qn string) string {
	parts := strings.Split(qn, ".")
	for i, part := range parts {
		if len(part) > 0 && part[0] >= 'A' && part[0] <= 'Z' {
			return strings.Join(parts[:i+1], ".")
		}
	}
	return ""
}

// ExtractLocation 从 AST 节点提取位置信息。
func ExtractLocation(node coreast.Node, path string) *model.Location {
	if node == nil {
		return nil
	}
	sp := node.StartPoint()
	ep := node.EndPoint()
	return &model.Location{
		FilePath:    path,
		StartLine:   sp.Row + 1,
		EndLine:     ep.Row + 1,
		StartColumn: sp.Column,
		EndColumn:   ep.Column,
	}
}

// FindNamedChildOfType 在具名子节点中查找指定类型。
func FindNamedChildOfType(node coreast.Node, nodeType string) coreast.Node {
	if node == nil {
		return nil
	}
	for i := 0; i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.Kind() == nodeType {
			return child
		}
	}
	return nil
}

// IsSubClassOf 判断 srcQN 类是否为 superQN 的子类（含自身）。
func IsSubClassOf(gc *core.GlobalContext, fc *core.FileContext, srcQN, superQN string) bool {
	return isSubClassOfRecursive(gc, fc, srcQN, superQN, make(map[string]bool))
}

func isSubClassOfRecursive(gc *core.GlobalContext, fc *core.FileContext, srcQN, superQN string, visited map[string]bool) bool {
	if srcQN == "" || superQN == "" || visited[srcQN] {
		return false
	}
	visited[srcQN] = true

	if srcQN == superQN || strings.HasSuffix(srcQN, "."+superQN) {
		return true
	}

	entry, ok := gc.FindByQualifiedName(srcQN)
	if !ok || entry.Element.Extra == nil {
		return false
	}

	mores := entry.Element.Extra.Mores
	var supers []string
	if sc, ok := mores["java.class.superclass"].(string); ok && sc != "" {
		supers = append(supers, sc)
	}
	if itfs, ok := mores["java.class.interfaces"].([]string); ok {
		supers = append(supers, itfs...)
	}

	for _, parentName := range supers {
		parents := PreciseResolve(gc, fc, Clean(parentName))
		for _, p := range parents {
			if isSubClassOfRecursive(gc, fc, p.Element.QualifiedName, superQN, visited) {
				return true
			}
		}
	}
	return false
}
