package core

import (
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// Extractor 负责从单个文件提取全部依赖关系，是引擎的核心与最复杂阶段。
// 该阶段在文件级并行执行。
type Extractor interface {
	// Extract 提取指定文件（相对路径）的全部依赖关系。
	Extract(relPath string, ctx *GlobalContext) ([]*model.DependencyRelation, error)
}
