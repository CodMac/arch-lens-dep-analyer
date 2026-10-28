// Package derived 提供 Java 语言的「派生关系」提取能力。
// TypeArg/Capture 属此类关系：既非静态结构（Extend/Implement），也非动作（Call/Create），
// 而是从已提取的关系或元素元数据推导/合成而来。
package derived

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// DerivedExtractor 派生关系提取器：从已提取的关系或元素元数据推导新关系。
type DerivedExtractor interface {
	Extract(ctx *Context) []*model.DependencyRelation
}

// Context 派生提取上下文。
type Context struct {
	GC       *core.GlobalContext
	FC       *core.FileContext
	Resolver core.SymbolResolver
	Deps     []*model.DependencyRelation // 已提取并经元数据增强的源关系（Capture 消费 is_capture）
}

type registry map[model.DependencyType]DerivedExtractor

// extractors 派生提取器注册表。
var extractors = registry{
	model.TypeArg: &TypeArgExtractor{},
	model.Capture: &CaptureExtractor{},
}

// ExtractAll 依次运行全部派生提取器（TypeArg 先行，Capture 依赖 enrich 后的 Deps）。
func ExtractAll(ctx *Context) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	for _, typ := range []model.DependencyType{model.TypeArg, model.Capture} {
		if ex, ok := extractors[typ]; ok {
			rels = append(rels, ex.Extract(ctx)...)
		}
	}
	return rels
}
