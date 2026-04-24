package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/output"
	_ "github.com/CodMac/arch-lens-dep-analyer/x/java"
)

const (
	MaxMermaidNodes = 200
	MaxMermaidEdges = 400
)

type Config struct {
	Lang        string
	SourcePath  string
	Filter      string
	Jobs        int
	OutDir      string
	Format      string
	FilterLevel int // 对应 core.FilterLevel
}

func main() {
	cfg := parseFlags()
	startTime := time.Now()

	// 1. 扫描文件
	fmt.Fprintf(os.Stderr, "[1/4] 🔍 正在扫描目录: %s\n", cfg.SourcePath)
	files, err := scanFiles(cfg.SourcePath, cfg.Filter, cfg.Lang)
	if err != nil {
		exitWithError("扫描文件失败", err)
	}
	fmt.Fprintf(os.Stderr, "    找到 %d 个候选文件\n", len(files))

	// 2. 执行核心分析过程 (内部会自动进行 NoiseFilter)
	fmt.Fprintf(os.Stderr, "[2/4] ⚙️  正在分析代码符号与关系 (Level: %d)...\n", cfg.FilterLevel)
	proc := NewFileProcessor(
		core.Language(cfg.Lang),
		false,
		false,
		cfg.Jobs,
		core.FilterLevel(cfg.FilterLevel), // 传入过滤等级
	)

	rels, gCtx, err := proc.ProcessFiles(cfg.SourcePath, files)
	if err != nil {
		exitWithError("分析执行失败", err)
	}

	// 3. 执行导出逻辑
	fmt.Fprintf(os.Stderr, "[3/4] 💾 正在写入结果文件...\n")
	ec, rc, err := runExport(cfg, gCtx, rels)
	if err != nil {
		exitWithError("导出失败", err)
	}

	fmt.Fprintf(os.Stderr, "    ✅ 完成: 导出实体=%d, 最终关系=%d\n", ec, rc)
	fmt.Fprintf(os.Stderr, "\n[4/4] ✨ 分析结束! 总耗时: %v\n", time.Since(startTime).Round(time.Millisecond))
}

func parseFlags() Config {
	c := Config{}
	flag.StringVar(&c.Lang, "lang", "java", "分析语言")
	flag.StringVar(&c.SourcePath, "path", ".", "源码根路径")
	flag.StringVar(&c.Filter, "filter", "", "文件过滤正则")
	flag.IntVar(&c.Jobs, "jobs", 4, "并发数")
	flag.StringVar(&c.OutDir, "out-dir", "./output", "输出目录")
	flag.StringVar(&c.Format, "format", "jsonl", "格式: jsonl, mermaid")
	flag.IntVar(&c.FilterLevel, "level", 1, "过滤等级: 0(Raw), 1(Balanced), 2(Pure)")
	flag.Parse()
	return c
}

func runExport(cfg Config, gCtx *core.GlobalContext, rels []*model.DependencyRelation) (int, int, error) {
	_ = os.MkdirAll(cfg.OutDir, 0755)

	format := cfg.Format
	if format == "mermaid" {
		if len(gCtx.Definitions) > MaxMermaidNodes || len(rels) > MaxMermaidEdges {
			fmt.Fprintf(os.Stderr, "    ⚠️  规模过大(%d 节点)，Mermaid 渲染可能失败，自动降级为 jsonl\n", len(gCtx.Definitions))
			format = "jsonl"
		}
	}

	exporter := output.NewExporter(cfg.OutDir, output.OutType(format))

	if format == "mermaid" {
		return exporter.ExportMermaidHTML(gCtx, rels)
	}
	return exporter.ExportJsonL(gCtx, rels)
}

// scanFiles 保持不变...
func scanFiles(root, filter, lang string) ([]string, error) {
	if filter == "" {
		filter = fmt.Sprintf(`.*\.%s$`, lang)
	}
	re, err := regexp.Compile(filter)
	if err != nil {
		return nil, err
	}

	var files []string
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && re.MatchString(path) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func exitWithError(msg string, err error) {
	fmt.Fprintf(os.Stderr, "❌ %s: %v\n", msg, err)
	os.Exit(1)
}
