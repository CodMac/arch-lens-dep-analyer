// Package linker 实现 Java 语言 core.Linker 接口：
// 构建 Java 特有的拓扑树（Package->SubPackage、Package->File、File->TopLevel 元素），
// 全部以 Contain 关系产出并全局去重。
package linker

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// Linker 实现 core.Linker。
type Linker struct{}

// New 创建 Java Linker。
func New() *Linker {
	return &Linker{}
}

// LinkHierarchy 构建 Java 拓扑树：
// 1. Package -> SubPackage (例如 com -> com.example)
// 2. Package -> File (例如 com.example -> Main.java)
// 3. File -> TopLevelClass (例如 Main.java -> com.example.Main)
// 注意：GlobalContext 内部自加锁，此处通过快照遍历，不持有任何锁。
func (l *Linker) LinkHierarchy(gc *core.GlobalContext) []*model.DependencyRelation {
	// 使用 map key 进行全局去重： "type:sourceQN->targetQN"
	relMap := make(map[string]*model.DependencyRelation)

	for _, fCtx := range gc.FileContexts() {
		// --- 1. 处理 Package 层级 ---
		if fCtx.PackageName != "" {
			// A. Package -> File
			l.addRel(relMap, model.Package, fCtx.PackageName, model.File, fCtx.FilePath)

			// B. Package -> SubPackage (递归拆解包名)
			// 如 "com.example.util" -> "com", "com.example", "com.example.util"
			parts := strings.Split(fCtx.PackageName, ".")
			for i := len(parts) - 1; i > 0; i-- {
				parentPkg := strings.Join(parts[:i], ".")
				subPkg := strings.Join(parts[:i+1], ".")
				l.addRel(relMap, model.Package, parentPkg, model.Package, subPkg)
			}
		}

		// --- 2. 处理 File -> TopLevelElements ---
		// 遍历该文件中的所有定义
		for _, entry := range fCtx.Definitions {
			// 逻辑：只有当一个元素的父级是包（或者是空的顶级元素）时，它才直接挂在文件节点下
			// 这样可以避免方法、内部类也平铺在文件下
			isTopLevel := entry.ParentQN == "" || entry.ParentQN == fCtx.PackageName

			if isTopLevel {
				l.addRel(relMap, model.File, fCtx.FilePath, entry.Element.Kind, entry.Element.QualifiedName)
			}
		}
	}

	// 转换为切片
	result := make([]*model.DependencyRelation, 0, len(relMap))
	for _, rel := range relMap {
		result = append(result, rel)
	}
	return result
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
			Kind:          srcKind,
			QualifiedName: srcQN,
			IsFormSource:  true,
		},
		Target: &model.CodeElement{
			Kind:          tgtKind,
			QualifiedName: tgtQN,
			IsFormSource:  true,
		},
	}
}
