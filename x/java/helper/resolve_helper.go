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

	// 点分嵌套名：Class.NestedInterface 形态（如 CodecConfigurer.DefaultCodecs）。
	// import 键是外层短名（CodecConfigurer），整串查 import 必然 miss；
	// 拆外层逐级解析：先按标准检索解析外层，再在解析 QN 下拼后缀查全局定义。
	if strings.Contains(symbol, ".") {
		if def := resolveDottedNested(gc, fc, symbol); def != nil {
			return []*core.DefinitionEntry{def}
		}
	}

	return nil
}

// resolveDottedNested 解析点分嵌套类型名（Class.NestedInterface / A.B.C 多级嵌套）。
// 从右向左尝试每个拆分点：拆为 outer + "." + suffix，先按标准检索解析 outer（短名/
// import/包内/通配符/全局），再在 outer 解析 QN 下拼 suffix 查全局定义。
// 递归调用 PreciseResolve 处理多级嵌套（外层自身可能也是嵌套名），字符串逐级缩短保证终止。
func resolveDottedNested(gc *core.GlobalContext, fc *core.FileContext, symbol string) *core.DefinitionEntry {
	for i := len(symbol) - 1; i > 0; i-- {
		if symbol[i] != '.' {
			continue
		}
		outer := symbol[:i]
		suffix := symbol[i+1:]
		outerDefs := PreciseResolve(gc, fc, outer)
		if len(outerDefs) == 0 {
			continue
		}
		elem := outerDefs[0].Element
		if !isTypeKind(elem.Kind) {
			continue
		}
		fullQN := elem.QualifiedName + "." + suffix
		if def, ok := gc.FindByQualifiedName(fullQN); ok {
			return def
		}
	}
	return nil
}

// isTypeKind 判断元素是否为类型类（可作为点分嵌套名拆分的有效外层）。
func isTypeKind(k model.ElementKind) bool {
	switch k {
	case model.Class, model.Interface, model.Enum, model.KAnnotation, model.AnonymousClass:
		return true
	}
	return false
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
