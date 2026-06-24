package helper

import (
	"slices"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// IsScopeContainer 是否容器kind
func IsScopeContainer(k model.ElementKind) bool {
	switch k {
	case model.Class, model.Interface, model.Enum, model.KAnnotation,
		model.Method, model.Lambda, model.ScopeBlock, model.AnonymousClass:
		return true
	}
	return false
}

// PreciseResolve 根据symbol查找符号定义
func PreciseResolve(gc *core.GlobalContext, fc *core.FileContext, symbol string) []*core.DefinitionEntry {
	gc.RLock()
	defer gc.RUnlock()

	// 短符号, 文件内查找
	if defs, ok := fc.FindByShortName(symbol); ok {
		return defs
	}

	// 短符号, 升级为确切QN, 全局查找
	if imps, ok := fc.Imports[symbol]; ok {
		for _, imp := range imps {
			if def, found := gc.FindByQualifiedName(imp.RawImportPath); found {
				return []*core.DefinitionEntry{def}
			}
		}
	}

	// 短符号, 包内查找
	pkgQN := gc.BuildQualifiedName(fc.PackageName, symbol)
	if def, ok := gc.FindByQualifiedName(pkgQN); ok {
		return []*core.DefinitionEntry{def}
	}

	// 短符号, 升级为匹配式QN, 全局查找
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

// GetBestElement 根据node和kinds，匹配合适的Element
func GetBestElement(fc *core.FileContext, node *sitter.Node, kinds []model.ElementKind) *model.CodeElement {
	if node == nil {
		return nil
	}
	var best *model.CodeElement
	var minSize uint32 = 0xFFFFFFFF
	row := int(node.StartPosition().Row + 1)
	for _, entry := range fc.Definitions {
		if slices.Contains(kinds, entry.Element.Kind) {
			if row >= entry.Element.Location.StartLine && row <= entry.Element.Location.EndLine {
				size := uint32(entry.Element.Location.EndLine - entry.Element.Location.StartLine)
				if size < minSize {
					minSize, best = size, entry.Element
				}
			}
		}
	}
	return best
}

// GetRealPackage 获得Element的包
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
			// 如果全局上下文没找到，继续向上找，直到匹配已知的 Package 模式
			continue
		}
	}
}

// GetOwnerClassQN 获得Element的持有类QN
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

// GetOutermostClassQN 获取最外层的类名 (例如把 A.B.C$1 还原为 A)
func GetOutermostClassQN(qn string) string {
	// 逻辑：在 Java 中，类名通常是大写开头
	parts := strings.Split(qn, ".")
	for i, part := range parts {
		// 简单判定：首字母大写通常是类名 (Java 规范)
		if len(part) > 0 && part[0] >= 'A' && part[0] <= 'Z' {
			return strings.Join(parts[:i+1], ".")
		}
	}
	return ""
}
