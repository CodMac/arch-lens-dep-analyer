package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/parser"
)

type FileProcessor struct {
	Language    core.Language
	OutputAST   bool
	FormatAST   bool
	Concurrency int
	FilterLevel core.FilterLevel
}

func NewFileProcessor(lang core.Language, outputAST, formatAST bool, concurrency int, filterLevel core.FilterLevel) *FileProcessor {
	if concurrency <= 0 {
		concurrency = 4
	}
	return &FileProcessor{
		Language:    lang,
		OutputAST:   outputAST,
		FormatAST:   formatAST,
		Concurrency: concurrency,
		FilterLevel: filterLevel,
	}
}

func (fp *FileProcessor) ProcessFiles(rootPath string, filePaths []string) ([]*model.DependencyRelation, *core.GlobalContext, error) {
	resolver, err := core.GetSymbolResolver(fp.Language)
	if err != nil {
		return nil, nil, err
	}

	gc := core.NewGlobalContext(resolver)
	absRoot, _ := filepath.Abs(rootPath)
	var allRelations []*model.DependencyRelation

	// --- 阶段 1: 并行收集 (Collector) ---
	start := time.Now()
	err = fp.runParallel(filePaths, func(path string, p parser.Parser) error {
		// file -> ast
		root, source, err := p.ParseFile(path, fp.OutputAST, fp.FormatAST)
		if err != nil {
			return err
		}

		// absPath -> relPath
		relPath, _ := filepath.Rel(absRoot, path)

		// collector
		cot, err := core.GetCollector(fp.Language)
		if err != nil {
			return err
		}

		// collect
		fc, err := cot.CollectDefinitions(root, relPath, source)
		if err != nil {
			return err
		}

		gc.RegisterFileContext(fc)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	fmt.Fprintf(os.Stderr, "      > [Step 1/5] Collector (Parallel): %v\n", time.Since(start).Round(time.Millisecond))

	// --- 阶段 2: 符号绑定 (Binder) ---
	start = time.Now()
	binder, err := core.GetBinder(fp.Language)
	if err != nil {
		return nil, nil, err
	}
	binder.BindSymbols(gc)
	fmt.Fprintf(os.Stderr, "      > [Step 2/5] Symbol Binder: %v\n", time.Since(start).Round(time.Millisecond))

	// --- 阶段 3: 并行提取依赖 (Extractor) ---
	start = time.Now()
	var mu sync.Mutex
	err = fp.runParallel(filePaths, func(path string, p parser.Parser) error {
		// extractor
		ext, err := core.GetExtractor(fp.Language)
		if err != nil {
			return err
		}

		// absPath -> relPath
		relPath, _ := filepath.Rel(absRoot, path)

		// extract
		rels, err := ext.Extract(relPath, gc)
		if err != nil {
			return err
		}

		mu.Lock()
		defer mu.Unlock()
		for _, rel := range rels {
			// 清理临时字段
			delete(rel.Mores, constants.TmpCtxNode)
			delete(rel.Mores, constants.TmpNode)

			allRelations = append(allRelations, rel)
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	fmt.Fprintf(os.Stderr, "      > [Step 3/5] Extractor (Parallel): %v\n", time.Since(start).Round(time.Millisecond))

	// --- 阶段 4: 拓扑链接 (Linker) ---
	start = time.Now()
	linker, err := core.GetLinker(fp.Language)
	if err != nil {
		return nil, nil, err
	}
	hierarchyRelations := linker.LinkHierarchy(gc)
	allRelations = append(allRelations, hierarchyRelations...)
	fmt.Fprintf(os.Stderr, "      > [Step 4/5] Hierarchy Linker: %v\n", time.Since(start).Round(time.Millisecond))

	// --- 阶段 5: 噪音过滤 (Noise Filtering) ---
	start = time.Now()
	filteredRelations := fp.filterNoise(allRelations)
	fmt.Fprintf(os.Stderr, "      > [Step 5/5] Noise Filter: %v\n", time.Since(start).Round(time.Millisecond))

	return filteredRelations, gc, nil
}

// filterNoise 调用语言特定的过滤器进行数据清洗
func (fp *FileProcessor) filterNoise(rels []*model.DependencyRelation) []*model.DependencyRelation {
	filter := core.GetNoiseFilter(fp.Language)
	filter.SetLevel(fp.FilterLevel)

	// 如果是 Raw 级别，直接返回
	if fp.FilterLevel == core.LevelRaw {
		return rels
	}

	result := make([]*model.DependencyRelation, 0, len(rels))
	for _, rel := range rels {
		// 如果不是噪音，则保留
		if !filter.IsNoise(*rel) {
			result = append(result, rel)
		}
	}
	return result
}

// runParallel 内部并发调度器
func (fp *FileProcessor) runParallel(paths []string, task func(string, parser.Parser) error) error {
	pathChan := make(chan string, len(paths))
	for _, p := range paths {
		pathChan <- p
	}
	close(pathChan)

	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once

	for i := 0; i < fp.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := parser.NewParser(fp.Language)
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
