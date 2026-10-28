// Package engine 实现五阶段流水线编排与并发调度。
package engine

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// Config 描述一次完整分析运行的配置。
type Config struct {
	// Language 目标语言。
	Language core.Language

	// RootPath 项目根路径（用于计算相对路径）。
	RootPath string

	// FilePaths 待分析的源文件路径列表。
	FilePaths []string

	// Concurrency 文件级并行度（<=0 时使用默认 4）。
	Concurrency int

	// FilterLevel 噪音过滤等级。
	FilterLevel core.FilterLevel
}

// EngineContext 承载流水线各阶段之间的共享数据。
type EngineContext struct {
	Config Config

	// Impl 语言实现（从注册表获取，缓存避免重复查找）。
	Impl *core.LanguageImpl

	// GC 全局符号上下文。
	GC *core.GlobalContext

	// Relations 已提取的依赖关系集合（阶段 3+ 填充）。
	Relations []*model.DependencyRelation
}

// NewContext 根据配置创建引擎上下文。
func NewContext(cfg Config) (*EngineContext, error) {
	impl, err := core.Get(cfg.Language)
	if err != nil {
		return nil, err
	}
	return &EngineContext{
		Config: cfg,
		Impl:   impl,
		GC:     core.NewGlobalContext(impl.SymbolResolver, impl.SymbolBuilder),
	}, nil
}

// ClearTempFields 清空全部关系的私有临时字段（流水线结束、导出前调用）。
func (ec *EngineContext) ClearTempFields() {
	for _, rel := range ec.Relations {
		if rel != nil {
			rel.ClearTemp()
		}
	}
}
