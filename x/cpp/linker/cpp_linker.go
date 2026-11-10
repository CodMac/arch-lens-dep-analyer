// Package linker 实现 C/C++ 语言 core.Linker 接口：
//   - Contain 拓扑（Namespace 层级 / File → 顶层元素）
//   - ImplLink 声明-实现链接（函数级签名配对，定义 → 声明）
package linker

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/helper"
)

// Linker 实现 core.Linker。
type Linker struct{}

// New 创建 C/C++ Linker。
func New() *Linker { return &Linker{} }

// LinkHierarchy 构建 C/C++ 拓扑树并补齐 ImplLink。
func (l *Linker) LinkHierarchy(gc core.GlobalContext) []*model.DependencyRelation {
	relMap := make(map[string]*model.DependencyRelation)

	// --- 1. File → 顶层元素 ---
	for _, fCtx := range gc.FileContexts() {
		filePath := fCtx.FilePath()
		for _, entry := range fCtx.Definitions() {
			if entry.Element.Kind == model.File {
				continue
			}
			if isTopLevel(entry, fCtx) {
				l.addRel(relMap, model.File, filePath, entry.Element.Kind, entry.Element.QualifiedName)
			}
		}
	}

	// --- 2. Namespace 层级（C++） ---
	l.linkNamespaces(gc, relMap)

	// --- 3. ImplLink（定义 → 声明） ---
	l.linkImpl(gc, relMap)

	result := make([]*model.DependencyRelation, 0, len(relMap))
	for _, rel := range relMap {
		result = append(result, rel)
	}
	return result
}

// isTopLevel 判定元素是否直接挂在文件节点下。
// C++：无 namespace 前缀（ParentQN 为空）的元素；C：ParentQN 为空。
func isTopLevel(entry *core.DefinitionEntry, fCtx core.FileContext) bool {
	if entry.ParentQN == "" {
		return true
	}
	// 父级是命名空间元素（非类/结构体）→ 也属文件顶层拓扑。
	if p, ok := fCtx.FindByQualifiedName(entry.ParentQN); ok {
		return p.Element.Kind == model.Namespace
	}
	return false
}

// linkNamespaces 构建 Namespace → SubNamespace、Namespace → 元素、Namespace → File 关系。
func (l *Linker) linkNamespaces(gc core.GlobalContext, relMap map[string]*model.DependencyRelation) {
	for _, def := range gc.Definitions() {
		if def.Element.Kind != model.Namespace {
			continue
		}
		qn := def.Element.QualifiedName
		parent := helper.PkgOf(qn)
		if parent != "" {
			l.addRel(relMap, model.Namespace, parent, model.Namespace, qn)
		}
	}

	// Namespace → 直接子元素由 Extractor 的 Contain 产出（按 ParentQN），此处不重复生成。

	// Namespace → File（文件内出现该 namespace 即建边）。
	for _, fCtx := range gc.FileContexts() {
		nfc, ok := fCtx.(core.NamespaceAwareFileContext)
		if !ok {
			continue
		}
		for _, ns := range nfc.Namespaces() {
			l.addRel(relMap, model.Namespace, ns, model.File, fCtx.FilePath())
		}
	}
}

// linkImpl 按函数签名配对声明与定义，产出 ImplLink（定义 → 声明）。
// 签名 = 完整 QN（含 ns::Class）+ 参数类型列表；static 函数无声明不参与；
// 只声明无定义不建链接。
func (l *Linker) linkImpl(gc core.GlobalContext, relMap map[string]*model.DependencyRelation) {
	type sigInfo struct {
		defs  []*model.CodeElement
		decls []*model.CodeElement
	}
	index := make(map[string]*sigInfo)

	for _, def := range gc.Definitions() {
		elem := def.Element
		if !isFunctionKind(elem.Kind) {
			continue
		}
		if isStaticFunc(elem) {
			continue
		}
		sig := implSignature(elem)
		if sig == "" {
			continue
		}
		info := index[sig]
		if info == nil {
			info = &sigInfo{}
			index[sig] = info
		}
		if isDeclKind(elem.Kind) {
			info.decls = append(info.decls, elem)
		} else {
			info.defs = append(info.defs, elem)
		}
	}

	for sig, info := range index {
		if len(info.defs) == 0 || len(info.decls) == 0 {
			continue
		}
		decl := info.decls[0]
		for _, def := range info.defs {
			key := string(model.ImplLink) + ":" + def.QualifiedName + "->" + decl.QualifiedName
			if _, exists := relMap[key]; exists {
				continue
			}
			relMap[key] = &model.DependencyRelation{
				Type:   model.ImplLink,
				Source: def,
				Target: decl,
				Mores:  map[string]interface{}{constants.RelImplLinkSignature: sig},
			}
		}
	}
}

// implSignature 构造 ImplLink 配对签名：完整 QN + 参数类型列表。
func implSignature(elem *model.CodeElement) string {
	if elem.Extra == nil {
		return ""
	}
	params, _ := elem.Extra.Mores[constants.FunctionParameters].([]string)
	types := make([]string, 0, len(params))
	for _, p := range params {
		types = append(types, paramTypeOnly(p))
	}
	return elem.QualifiedName + "(" + strings.Join(types, ",") + ")"
}

// paramTypeOnly 从 "int x" 提取 "int"。
func paramTypeOnly(p string) string {
	p = helper.NormalizeTypeText(p)
	if i := strings.LastIndexAny(p, " \t"); i > 0 {
		candidate := strings.TrimSpace(p[i+1:])
		if candidate != "" && !strings.ContainsAny(candidate, "*&[]<>,()") {
			return strings.TrimSpace(p[:i])
		}
	}
	return p
}

// isStaticFunc 判定函数是否为 static（不参与 ImplLink 配对）。
func isStaticFunc(elem *model.CodeElement) bool {
	if elem.Extra == nil {
		return false
	}
	for _, m := range elem.Extra.Modifiers {
		if m == "static" {
			return true
		}
	}
	return false
}

// addRel 生成唯一 Key 并加入 Map（已存在则跳过）。
func (l *Linker) addRel(m map[string]*model.DependencyRelation, srcKind model.ElementKind, srcQN string, tgtKind model.ElementKind, tgtQN string) {
	key := string(srcKind) + ":" + srcQN + "->" + tgtQN
	if _, exists := m[key]; exists {
		return
	}
	m[key] = &model.DependencyRelation{
		Type: model.Contain,
		Source: &model.CodeElement{
			Kind: srcKind, QualifiedName: srcQN, IsFormSource: true,
		},
		Target: &model.CodeElement{
			Kind: tgtKind, QualifiedName: tgtQN, IsFormSource: true,
		},
	}
}

func isFunctionKind(k model.ElementKind) bool {
	switch k {
	case model.Function, model.Method, model.FunctionDecl, model.MethodDecl:
		return true
	}
	return false
}

func isDeclKind(k model.ElementKind) bool {
	return k == model.FunctionDecl || k == model.MethodDecl
}
