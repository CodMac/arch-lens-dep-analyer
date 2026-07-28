package helper

import (
	"slices"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"

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

// PreciseResolve 根据 symbol 查找符号定义（支持 Outer.User 形式的内部类查找，统一使用 "." 分隔 QN）
func PreciseResolve(gc *core.GlobalContext, fc *core.FileContext, symbol string) []*core.DefinitionEntry {
	gc.RLock()
	defer gc.RUnlock()

	// 内部闭包：处理单一顶级短符号或绝对 QN 的 4 级检索策略
	resolveTopLevelSymbol := func(sym string) []*core.DefinitionEntry {
		// 短符号, 文件内查找
		if defs, ok := fc.FindByShortName(sym); ok {
			return defs
		}

		// 短符号, 升级为确切 QN, 全局查找 (精准 Import)
		if imps, ok := fc.Imports[sym]; ok {
			for _, imp := range imps {
				if def, found := gc.FindByQualifiedName(imp.RawImportPath); found {
					return []*core.DefinitionEntry{def}
				}
			}
		}

		// 短符号, 包内查找
		pkgQN := gc.BuildQualifiedName(fc.PackageName, sym)
		if def, ok := gc.FindByQualifiedName(pkgQN); ok {
			return []*core.DefinitionEntry{def}
		}

		// 短符号, 升级为匹配式 QN, 全局查找 (通配符 Import)
		for _, imps := range fc.Imports {
			for _, imp := range imps {
				if imp.IsWildcard {
					basePath := strings.TrimSuffix(imp.RawImportPath, "*")
					if def, ok := gc.FindByQualifiedName(basePath + sym); ok {
						return []*core.DefinitionEntry{def}
					}
				}
			}
		}

		// 直接按字符串 QN 全局查找 (覆盖了绝对 QN 如 "com.test.case1.Outer.User" 的直接匹配)
		if def, ok := gc.FindByQualifiedName(sym); ok {
			return []*core.DefinitionEntry{def}
		}

		return nil
	}

	// 1. 如果 symbol 包含点号（如 "Outer.User" 或 "Outer.User.Nested"）
	if strings.Contains(symbol, ".") {
		parts := strings.SplitN(symbol, ".", 2)
		baseName := parts[0] // "Outer"
		tailPath := parts[1] // "User" 或 "User.Nested"

		// A. 尝试先将 baseName ("Outer") 当作顶层类按常规策略解析
		if baseDefs := resolveTopLevelSymbol(baseName); len(baseDefs) > 0 {
			baseQN := baseDefs[0].Element.QualifiedName
			// 直接使用 "." 拼接内部类全路径 (如 "com.test.case1.Outer.User")
			innerQN := baseQN + "." + tailPath
			if innerDef, found := gc.FindByQualifiedName(innerQN); found {
				return []*core.DefinitionEntry{innerDef}
			}
		}
	}

	// 2. 普通单段短符号（如 "User"）或标准的绝对 QN（如 "com.test.case1.Outer.User"）查找
	return resolveTopLevelSymbol(symbol)
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

// GetOwnerClass 获得Element的持有类
func GetOwnerClass(gc *core.GlobalContext, elem *model.CodeElement) *model.CodeElement {
	curr := elem
	for curr != nil {
		if curr.Kind == model.Class || curr.Kind == model.Interface || curr.Kind == model.AnonymousClass {
			return curr
		}

		if entry, ok := gc.FindByQualifiedName(curr.QualifiedName); ok && entry.ParentQN != "" {
			if next, ok := gc.FindByQualifiedName(entry.ParentQN); ok {
				curr = next.Element
				continue
			}
		}

		break
	}
	return curr
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

// IsSubClassOf 是否子类
func IsSubClassOf(gc *core.GlobalContext, fc *core.FileContext, subQN, superQN string) bool {
	if subQN == "" || superQN == "" || subQN == superQN {
		return subQN == superQN
	}

	entry, ok := gc.FindByQualifiedName(subQN)
	if !ok || entry.Element.Extra == nil {
		return false
	}

	if sc, ok := entry.Element.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
		parents := PreciseResolve(gc, fc, Clean(sc))
		for _, p := range parents {
			if p.Element.QualifiedName == superQN || IsSubClassOf(gc, fc, p.Element.QualifiedName, superQN) {
				return true
			}
		}
	}
	return false
}
