package engine

import (
	"sync"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/core/ast"
)

// Scheduler 提供文件级并发调度：将任务并行分发给 N 个 worker。
// 每个 worker 独立持有语言资源（parser），避免并发争用。
// 首个错误即终止，通过 errOnce 保证只记录一次。
type Scheduler struct {
	concurrency int
	parserFn    func() (core.Parser, error)
}

// NewScheduler 创建调度器。parserFn 负责为每个 worker 创建一个语言解析器。
func NewScheduler(concurrency int, parserFn func() (core.Parser, error)) *Scheduler {
	if concurrency <= 0 {
		concurrency = 4
	}
	return &Scheduler{concurrency: concurrency, parserFn: parserFn}
}

// Run 并行执行 task（对每个 path 调用一次），并返回首个错误。
func (s *Scheduler) Run(paths []string, task func(path string, p core.Parser) error) error {
	if len(paths) == 0 {
		return nil
	}

	pathChan := make(chan string, len(paths))
	for _, p := range paths {
		pathChan <- p
	}
	close(pathChan)

	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once

	workers := s.concurrency
	if workers > len(paths) {
		workers = len(paths)
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := s.parserFn()
			if err != nil {
				errOnce.Do(func() { firstErr = err })
				return
			}
			defer p.Close()

			for path := range pathChan {
				if err := task(path, p); err != nil {
					errOnce.Do(func() { firstErr = err })
					return
				}
			}
		}()
	}
	wg.Wait()
	return firstErr
}

// ParseAll 便捷方法：并行解析全部文件为 SourceFile。
func ParseAll(s *Scheduler, paths []string) (map[string]ast.SourceFile, error) {
	files := make(map[string]ast.SourceFile)
	var mu sync.Mutex
	err := s.Run(paths, func(path string, p core.Parser) error {
		src, err := p.ParseFile(path)
		if err != nil {
			return err
		}
		mu.Lock()
		files[path] = src
		mu.Unlock()
		return nil
	})
	return files, err
}
