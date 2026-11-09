package helper

import (
	"path/filepath"
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
func PreciseResolve(gc core.GlobalContext, fc core.FileContext, symbol string) []*core.DefinitionEntry {
	// 短符号，文件内查找
	if defs, ok := fc.FindByShortName(symbol); ok {
		return defs
	}

	// 短符号，升级为确切 QN，全局查找。
	// JLS §7.5.1：单类型导入遮蔽同包类型。若存在非通配符单类型导入，
	// 即便其 QN 未收录进 GlobalContext（外部库），也【不得回退】到同包/
	// 裸全局查找——直接返回 nil，由调用方（ResolveType）外部兜底用 import
	// 包前缀分配归属（如 io.r2dbc.spi.Parameter）。
	if imps, ok := fc.Imports()[symbol]; ok {
		found := false
		for _, imp := range imps {
			if imp.IsWildcard {
				continue
			}
			found = true
			if def, defOK := gc.FindByQualifiedName(imp.RawImportPath); defOK {
				return []*core.DefinitionEntry{def}
			}
		}
		if found {
			return nil
		}
	}

	// 短符号，包内查找
	pkgQN := gc.BuildQualifiedName(fc.PackageName(), symbol)
	if def, ok := gc.FindByQualifiedName(pkgQN); ok {
		return []*core.DefinitionEntry{def}
	}

	// 短符号，升级为匹配式 QN，全局查找（通配符导入）
	for _, imps := range fc.Imports() {
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
func resolveDottedNested(gc core.GlobalContext, fc core.FileContext, symbol string) *core.DefinitionEntry {
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

// ResolveInScope 在文件内同名定义中，按引用点所在的外层类作用域消歧（D-1）。
// 根因：同一文件存在多个同名嵌套类型（如 DataBufferUtils 的 ReadCompletionHandler.Attachment
// 与 WriteCompletionHandler.Attachment），FindByShortName 返回全部定义，调用方取 [0] 会按
// 文件收集顺序（先定义者优先）错配，而非按引用点所在嵌套类的作用域选择。
// 算法：引用点 node 定位当前外层类 -> 优先匹配当前类作用域下的类型候选（等于 scope 或
// scope 直接子成员，且必须是类型 kind，排除构造器 Method/字段等同名非类型）-> 未命中沿
// 词法外层类回溯 -> 全部未命中返回 nil（由调用方保持 [0] 原语义，外部兜底不受影响）。
// 注意：仅当 node 非空且候选多于 1 个时启用消歧；单一定义 / 无 node 时原样返回候选。
func ResolveInScope(gc core.GlobalContext, fc core.FileContext, node coreast.Node, defs []*core.DefinitionEntry) []*core.DefinitionEntry {
	if len(defs) <= 1 || node == nil {
		return defs
	}
	cls := GetBestElement(fc, node, []model.ElementKind{model.Class, model.Interface, model.Enum, model.AnonymousClass})
	if cls == nil {
		return defs
	}
	// 类型候选过滤：消歧只针对类型 kind（Class/Interface/Enum/KAnnotation/AnonymousClass），
	// 排除构造器 Method/字段/变量等同名非类型（否则 FindByShortName 混入构造器时会误命中）。
	var typeDefs []*core.DefinitionEntry
	for _, d := range defs {
		if isTypeKind(d.Element.Kind) {
			typeDefs = append(typeDefs, d)
		}
	}
	if len(typeDefs) <= 1 {
		return defs
	}
	seen := make(map[string]bool)
	for scope := cls.QualifiedName; scope != ""; {
		if !seen[scope] {
			seen[scope] = true
			for _, d := range typeDefs {
				// 匹配当前类作用域下的类型：嵌套成员（scope.Name）或顶层类型（等于 scope）。
				qn := d.Element.QualifiedName
				if qn == scope || strings.HasPrefix(qn, scope+".") {
					return []*core.DefinitionEntry{d}
				}
			}
		}
		parentEntry, ok := gc.FindByQualifiedName(scope)
		if !ok {
			break
		}
		scope = parentEntry.ParentQN
	}
	return defs
}

// ResolveInheritanceParent 解析继承链上的父类/接口名（C-5 继承方法链）。
// 根因：父类裸名（如 ReaderContext）可能在调用方文件未 import，只在父类元素所在
// 文件 import（跨包继承链第 2/3 级或链式返回类型的父类）。调用方 fc 解析失败导致
// 继承链断裂、父类方法 CALL 缺失。
// 策略：优先用 elem.Path 反查父类元素所在文件的 fc 解析（该文件 import 了下一级
// 父类），失败回退调用方 fc 的常规检索，再失败返回 nil（调用方按原逻辑兜底）。
// 注意：引擎 normalizeFilePath 会把 elem.Path 相对化为 gc.FileContext key；但测试
// helper（RunExtraction）只相对化 fc.FilePath，元素 Path 仍为绝对路径。此处先按
// key 精确查找，miss 时按文件 basename 兜底反查（绝对/相对路径 basename 一致）。
func ResolveInheritanceParent(gc core.GlobalContext, fc core.FileContext, elem *model.CodeElement, name string) []*core.DefinitionEntry {
	// 1. 优先用元素所在文件解析：elem.Path 与 gc.FileContext key 一致（真实引擎）。
	//    注意：parentFC == fc 时（elem 就在调用方文件）【不得】跳过——此时应继续用
	//    elem 所在文件解析，而不是落入 basename 兜底（全量多同名文件时 basename 可能
	//    命中错误文件，如 servlet/reactive 各有 AbstractUrlHandlerMapping.java）。
	if elem != nil && elem.Path != "" {
		if parentFC, ok := gc.FileContext(elem.Path); ok {
			if defs := PreciseResolve(gc, parentFC, name); len(defs) > 0 {
				return defs
			}
		}
	}
	// 2. 测试环境兜底：elem.Path 为绝对路径（key 是相对路径），按 basename 反查。
	//    仅当步骤 1 未命中（gc.FileContext 无此 key）时执行，避免全量多同名文件误匹配。
	if elem != nil && elem.Path != "" {
		if _, ok := gc.FileContext(elem.Path); !ok {
			base := filepath.Base(elem.Path)
			for key, candidate := range gc.FileContexts() {
				if filepath.Base(key) == base {
					if defs := PreciseResolve(gc, candidate, name); len(defs) > 0 {
						return defs
					}
				}
			}
		}
	}
	// 3. 回退调用方 fc 常规检索。
	return PreciseResolve(gc, fc, name)
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
func GetBestElement(fc core.FileContext, node coreast.Node, kinds []model.ElementKind) *model.CodeElement {
	if node == nil {
		return nil
	}
	var best *model.CodeElement
	var minSize = 1 << 30
	row := int(node.StartPoint().Row + 1)
	for _, entry := range fc.Definitions() {
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
func GetRealPackage(gc core.GlobalContext, elem *model.CodeElement) string {
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
func GetOwnerClassQN(gc core.GlobalContext, elem *model.CodeElement) string {
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
func GetOwnerClass(gc core.GlobalContext, elem *model.CodeElement) *model.CodeElement {
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
func IsSubClassOf(gc core.GlobalContext, fc core.FileContext, srcQN, superQN string) bool {
	return isSubClassOfRecursive(gc, fc, srcQN, superQN, make(map[string]bool))
}

func isSubClassOfRecursive(gc core.GlobalContext, fc core.FileContext, srcQN, superQN string, visited map[string]bool) bool {
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
