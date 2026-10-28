package core

import (
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// Linker 负责补齐层级关系（Contain/Extend/Implement），构建全局符号树。
// 该阶段串行执行（依赖全局拓扑）。
type Linker interface {
	// LinkHierarchy 生成层级拓扑关系。
	LinkHierarchy(ctx *GlobalContext) []*model.DependencyRelation
}
