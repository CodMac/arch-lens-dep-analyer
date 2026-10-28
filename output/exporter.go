package output

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// Exporter 负责将分析结果以 JSONL 写入指定输出目录。
type Exporter struct {
	OutDir string
}

// NewExporter 创建导出器。
func NewExporter(dir string) *Exporter {
	return &Exporter{OutDir: dir}
}

// ExportJsonL 将元素与关系分别导出为 JSONL 文件，返回（元素数, 关系数）。
func (e *Exporter) ExportJsonL(elements []*model.CodeElement, rels []*model.DependencyRelation) (int, int, error) {
	if err := os.MkdirAll(e.OutDir, 0755); err != nil {
		return 0, 0, err
	}

	// 元素文件。
	elPath := filepath.Join(e.OutDir, "element.jsonl")
	f, err := os.Create(elPath)
	if err != nil {
		return 0, 0, err
	}
	w := NewJSONLWriter(f)
	for _, el := range elements {
		if el == nil {
			continue
		}
		if err := w.Write(el); err != nil {
			f.Close()
			return 0, 0, err
		}
	}
	if err := f.Close(); err != nil {
		return 0, 0, err
	}

	// 按关系类型分文件。
	byType := map[model.DependencyType][]*model.DependencyRelation{}
	for _, r := range rels {
		if r == nil {
			continue
		}
		byType[r.Type] = append(byType[r.Type], r)
	}
	types := make([]model.DependencyType, 0, len(byType))
	for t := range byType {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })

	relCount := 0
	for _, t := range types {
		relPath := filepath.Join(e.OutDir, "relation_"+strings.ToLower(string(t))+".jsonl")
		rf, err := os.Create(relPath)
		if err != nil {
			return 0, 0, err
		}
		rw := NewJSONLWriter(rf)
		for _, r := range byType[t] {
			if err := rw.Write(r); err != nil {
				rf.Close()
				return 0, 0, err
			}
			relCount++
		}
		if err := rf.Close(); err != nil {
			return 0, 0, err
		}
	}
	return len(elements), relCount, nil
}
