package engine

import (
	"fmt"
)

// Stage 表示流水线中的一个处理阶段。
// 实现方需提供阶段名、是否支持文件级并行，以及执行逻辑。
type Stage interface {
	// Name 返回阶段名称（用于日志与调试）。
	Name() string

	// Concurrent 返回该阶段是否在文件级并行执行。
	Concurrent() bool

	// Execute 执行阶段逻辑。
	Execute(ec *EngineContext) error
}

// Pipeline 是阶段编排器：按注册顺序串行执行各阶段。
// 并发阶段内部通过 Scheduler 做文件级并行。
type Pipeline struct {
	stages []Stage
}

// NewPipeline 创建空流水线。
func NewPipeline() *Pipeline {
	return &Pipeline{stages: make([]Stage, 0)}
}

// AddStage 追加一个阶段。
func (p *Pipeline) AddStage(s Stage) *Pipeline {
	p.stages = append(p.stages, s)
	return p
}

// Run 按序执行全部阶段。
func (p *Pipeline) Run(ec *EngineContext) error {
	for _, s := range p.stages {
		if err := s.Execute(ec); err != nil {
			return fmt.Errorf("stage %q failed: %w", s.Name(), err)
		}
	}
	return nil
}

// StageCount 返回已注册的阶段数。
func (p *Pipeline) StageCount() int {
	return len(p.stages)
}
