package output

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

type OutType string

const (
	JsonL   OutType = "jsonl"
	Mermaid OutType = "mermaid"
)

type Exporter struct {
	outputDir  string
	outputType OutType
}

func NewExporter(outputDir string, outputType OutType) *Exporter {
	return &Exporter{outputDir: outputDir, outputType: outputType}
}

func (p *Exporter) ExportJsonL(gCtx *core.GlobalContext, rels []*model.DependencyRelation) (int, int, error) {
	fmt.Fprintf(os.Stderr, "Export jsonl, entry-size: %d , rels-size: %d\n", len(gCtx.Definitions), len(rels))

	var wg sync.WaitGroup
	var relCount int64
	var elemCount int64
	var finalErr error
	var mu sync.Mutex

	setErr := func(err error) {
		mu.Lock()
		if finalErr == nil {
			finalErr = err
		}
		mu.Unlock()
	}

	// 1. 并行导出元素
	wg.Add(1)
	go func() {
		defer wg.Done()
		count, err := p.exportElements(gCtx)
		if err != nil {
			setErr(err)
			return
		}
		atomic.AddInt64(&elemCount, int64(count))
	}()

	// 2. 按关系类型分组
	relsByType := make(map[model.DependencyType][]*model.DependencyRelation)
	for _, rel := range rels {
		relsByType[rel.Type] = append(relsByType[rel.Type], rel)
	}

	// 3. 并行导出每种类型的关系
	for t, rs := range relsByType {
		wg.Add(1)
		go func(relType model.DependencyType, typeRels []*model.DependencyRelation) {
			defer wg.Done()
			count, err := p.exportRelationType(relType, typeRels)
			if err != nil {
				setErr(err)
				return
			}
			atomic.AddInt64(&relCount, int64(count))
		}(t, rs)
	}

	wg.Wait()
	return int(elemCount), int(relCount), finalErr
}

func (p *Exporter) exportElements(gCtx *core.GlobalContext) (int, error) {
	elemPath := filepath.Join(p.outputDir, "element.jsonl")
	elemFile, err := os.Create(elemPath)
	if err != nil {
		return 0, err
	}
	defer elemFile.Close()

	bw := bufio.NewWriter(elemFile)
	defer bw.Flush()

	elemWriter := NewJSONLWriter(bw)
	count := 0
	for _, entry := range gCtx.Definitions {
		if err := elemWriter.Write(entry.Element); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (p *Exporter) exportRelationType(relType model.DependencyType, rels []*model.DependencyRelation) (int, error) {
	fileName := fmt.Sprintf("relation_%s.jsonl", strings.ToLower(string(relType)))
	relPath := filepath.Join(p.outputDir, fileName)
	relFile, err := os.Create(relPath)
	if err != nil {
		return 0, err
	}
	defer relFile.Close()

	bw := bufio.NewWriter(relFile)
	defer bw.Flush()

	relWriter := NewJSONLWriter(bw)
	count := 0
	for _, rel := range rels {
		if err := relWriter.Write(rel); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (p *Exporter) ExportMermaidHTML(gCtx *core.GlobalContext, rels []*model.DependencyRelation) (int, int, error) {
	htmlPath := filepath.Join(p.outputDir, "visualization.html")

	f, err := os.Create(htmlPath)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	bw := bufio.NewWriter(f)
	defer bw.Flush()

	fmt.Fprintln(bw, `<!DOCTYPE html><html><head><meta charset="UTF-8"><script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script></head>
<body><div class="mermaid">graph LR`)

	elemCount := 0
	// 1. 绘制子图结构 (File -> Elements)
	for _, fCtx := range gCtx.FileContexts {
		fmt.Fprintf(bw, "  subgraph %s [📄 %s]\n", safeID(fCtx.FilePath), fCtx.FilePath)
		for _, entry := range fCtx.Definitions {
			nodeID := safeID(entry.Element.QualifiedName)
			fmt.Fprintf(bw, "    %s%s\n", nodeID, getNodeShape(entry.Element))
			elemCount++
		}
		fmt.Fprintln(bw, "  end")
	}

	// 2. 绘制依赖线条
	relCount := 0
	for _, rel := range rels {
		// 跳过包含关系，因为 subgraph 已经体现了
		if rel.Type == model.Contain {
			continue
		}

		srcID, tgtID := safeID(rel.Source.QualifiedName), safeID(rel.Target.QualifiedName)
		if srcID == tgtID {
			continue
		}

		// 如果目标是外部符号且通过了过滤，给它一个特殊样式
		edgeStyle := ""
		if rel.Target.IsFormExternal {
			edgeStyle = "---" // 外部依赖用虚线或不同颜色区分
		}

		fmt.Fprintf(bw, "  %s -- %s --> %s%s\n", srcID, rel.Type, tgtID, edgeStyle)
		relCount++
	}

	fmt.Fprintln(bw, `</div><script>mermaid.initialize({startOnLoad:true, maxTextSize:1000000});</script></body></html>`)

	return elemCount, relCount, nil
}

func safeID(id string) string {
	r := strings.NewReplacer(".", "_", "(", "_", ")", "_", "[", "_", "]", "_", " ", "_", "@", "at", "$", "_")
	return "n_" + r.Replace(id)
}

func getNodeShape(el *model.CodeElement) string {
	name := el.Name
	if el.IsFormExternal {
		name = name + " (ext)"
	}
	switch el.Kind {
	case model.Interface:
		return fmt.Sprintf("([\"%s <small>(%s)</small>\"])", name, el.Kind)
	case model.Class:
		return fmt.Sprintf("[\"%s <small>(%s)</small>\"]", name, el.Kind)
	case model.Method:
		return fmt.Sprintf("[/\"%s <small>(%s)</small>\"/]", name, el.Kind)
	default:
		return fmt.Sprintf("[\"%s <small>(%s)</small>\"]", name, el.Kind)
	}
}
