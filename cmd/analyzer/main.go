// Package main 是 arch-lens-dep-analyer 的命令行入口（薄 CLI 层）。
// 只做参数解析、文件扫描、调用 engine 流水线并交给 output 导出，不承载任何分析逻辑。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/engine"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/output"
	_ "github.com/CodMac/arch-lens-dep-analyer/x/java" // 注册 Java 语言实现
)

// Config 命令行配置。
type Config struct {
	Lang        string
	SourcePath  string
	Filter      string
	Jobs        int
	OutDir      string
	FilterLevel int
}

func main() {
	cfg := parseFlags()
	totalStart := time.Now()

	// 1. 扫描文件
	stepStart := time.Now()
	fmt.Fprintf(os.Stderr, "[1/4] 🔍 正在扫描目录: %s\n", cfg.SourcePath)
	files, err := scanFiles(cfg.SourcePath, cfg.Filter, cfg.Lang)
	if err != nil {
		exitWithError("扫描文件失败", err)
	}
	fmt.Fprintf(os.Stderr, "    找到 %d 个候选文件 (耗时: %v)\n", len(files), time.Since(stepStart).Round(time.Millisecond))

	if len(files) == 0 {
		exitWithError("没有找到待分析的源文件", fmt.Errorf("路径 %s 下无匹配文件", cfg.SourcePath))
	}

	// 2. 组装语言实现并执行五阶段流水线
	stepStart = time.Now()
	fmt.Fprintf(os.Stderr, "[2/4] ⚙️  正在分析代码符号与关系 (Level: %d)...\n", cfg.FilterLevel)
	impl, err := core.Get(core.Language(cfg.Lang))
	if err != nil {
		exitWithError("获取语言实现失败", err)
	}

	ec, err := engine.RunDefault(engine.Config{
		Language:    core.Language(cfg.Lang),
		RootPath:    cfg.SourcePath,
		FilePaths:   files,
		Concurrency: cfg.Jobs,
		FilterLevel: core.FilterLevel(cfg.FilterLevel),
	}, func() (core.Parser, error) {
		return impl.Parser.NewParser(), nil
	})
	if err != nil {
		exitWithError("分析执行失败", err)
	}
	fmt.Fprintf(os.Stderr, "    分析完成 (耗时: %v)\n", time.Since(stepStart).Round(time.Millisecond))

	// 3. 执行导出逻辑
	stepStart = time.Now()
	fmt.Fprintf(os.Stderr, "[3/4] 💾 正在写入结果文件...\n")
	ecCount, relCount, err := runExport(cfg, ec)
	if err != nil {
		exitWithError("导出失败", err)
	}
	fmt.Fprintf(os.Stderr, "    ✅ 完成: 导出实体=%d, 最终关系=%d (耗时: %v)\n", ecCount, relCount, time.Since(stepStart).Round(time.Millisecond))

	// 4. 总计
	fmt.Fprintf(os.Stderr, "\n[4/4] ✨ 所有任务执行完毕! 总耗时: %v\n", time.Since(totalStart).Round(time.Millisecond))
}

// parseFlags 解析命令行参数。
func parseFlags() Config {
	c := Config{}
	flag.StringVar(&c.Lang, "lang", "java", "分析语言")
	flag.StringVar(&c.SourcePath, "path", ".", "源码根路径")
	flag.StringVar(&c.Filter, "filter", "", "文件过滤正则")
	flag.IntVar(&c.Jobs, "jobs", runtime.NumCPU(), "并发数")
	flag.StringVar(&c.OutDir, "out-dir", "./output", "输出目录")
	flag.IntVar(&c.FilterLevel, "level", 1, "过滤等级: 0(Raw), 1(Balanced), 2(Pure)")
	flag.Parse()
	return c
}

// runExport 将分析结果以 JSONL 导出到输出目录。
func runExport(cfg Config, ec *engine.EngineContext) (int, int, error) {
	if err := os.MkdirAll(cfg.OutDir, 0755); err != nil {
		return 0, 0, err
	}

	exporter := output.NewExporter(cfg.OutDir)
	rels := ec.Relations

	// 元素列表：GlobalContext 的全部定义。
	defs := ec.GC.Definitions()
	elements := make([]*model.CodeElement, 0, len(defs))
	for _, def := range defs {
		elements = append(elements, def.Element)
	}

	return exporter.ExportJsonL(elements, rels)
}

// scanFiles 按正则过滤扫描目录下的源文件。
func scanFiles(rootPath, filter, lang string) ([]string, error) {
	if filter == "" {
		filter = fmt.Sprintf(`.*\.%s$`, lang)
	}
	re, err := regexp.Compile(filter)
	if err != nil {
		return nil, err
	}

	absRoot, _ := filepath.Abs(rootPath)

	var files []string
	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && re.MatchString(path) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// exitWithError 输出错误并退出。
func exitWithError(msg string, err error) {
	fmt.Fprintf(os.Stderr, "❌ %s: %v\n", msg, err)
	os.Exit(1)
}
