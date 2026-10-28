// Package test e2e 黄金回归测试：跑完整五阶段流水线（Collect+Bind+Extract+Linker+NoiseFilter），
// 将最终元素与关系序列化为确定性快照，与 testdata/e2e 下的 .golden.jsonl 基线比对。
//
// 快照确定性策略：原始输出受 map 迭代顺序影响（元素/关系的集合与内部 Mores 键序），
// 因此生成快照前统一排序——元素按 QualifiedName 排序，关系按 (Type, SourceQN, TargetQN, Location) 排序，
// Mores 由 encoding/json 按键排序序列化，从而同一输入必然产出同一快照。
//
// 更新基线：go test ./x/java/test -run TestE2EGolden -update
package test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// updateGolden 用于刷新黄金基线（go test -update）。
var updateGolden = flag.Bool("update", false, "update golden files")

// goldenDir 黄金基线目录（相对仓库根）。
const goldenDir = "x/java/testdata/e2e"

// goldenEntry 是快照中的一行：元素或关系，统一按 QN/Type 序列化为确定性 JSON。
type goldenEntry struct {
	QN       string `json:"QN"`
	Type     string `json:"Type,omitempty"`
	Kind     string `json:"Kind,omitempty"`
	Source   string `json:"Source,omitempty"`
	Target   string `json:"Target,omitempty"`
	Location string `json:"Location,omitempty"`
	Mores    string `json:"Mores,omitempty"`
	IsSource bool   `json:"IsSource"`
}

// e2eSnapshot 收集流水线最终结果的确定性快照。
type e2eSnapshot struct {
	entries []goldenEntry
}

// addElement 记录一个元素。
func (s *e2eSnapshot) addElement(el *model.CodeElement) {
	if el == nil {
		return
	}
	s.entries = append(s.entries, goldenEntry{
		QN:       el.QualifiedName,
		Kind:     string(el.Kind),
		IsSource: true,
	})
}

// addRelation 记录一条关系。
func (s *e2eSnapshot) addRelation(r *model.DependencyRelation) {
	if r == nil {
		return
	}
	src, tgt := "", ""
	if r.Source != nil {
		src = r.Source.QualifiedName
	}
	if r.Target != nil {
		tgt = r.Target.QualifiedName
	}
	loc := ""
	if r.Location != nil {
		loc = r.Location.FilePath + ":" + itoa(r.Location.StartLine)
	}
	mores, _ := json.Marshal(r.Mores)
	s.entries = append(s.entries, goldenEntry{
		QN:       src,
		Type:     string(r.Type),
		Source:   src,
		Target:   tgt,
		Location: loc,
		Mores:    string(mores),
	})
}

// sortAndRender 排序并渲染为逐行 JSON 文本。
func (s *e2eSnapshot) sortAndRender() string {
	sort.SliceStable(s.entries, func(i, j int) bool {
		a, b := s.entries[i], s.entries[j]
		if a.IsSource != b.IsSource {
			return a.IsSource // 元素在前
		}
		if a.QN != b.QN {
			return a.QN < b.QN
		}
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		if a.Target != b.Target {
			return a.Target < b.Target
		}
		if a.Location != b.Location {
			return a.Location < b.Location
		}
		return a.Mores < b.Mores
	})
	var out []string
	for _, e := range s.entries {
		b, _ := json.Marshal(e)
		out = append(out, string(b))
	}
	return strings.Join(out, "\n")
}

// TestE2EGolden 跑完整流水线并对拍黄金基线。
func TestE2EGolden(t *testing.T) {
	paths := resolveAllTestdata(t)

	snap := &e2eSnapshot{}

	// 完整流水线（等价 CLI：含 Linker + NoiseFilter Level1=LevelBalanced）。
	ec := RunPipeline(t, "x/java/testdata", paths, core.LevelBalanced)

	// 元素（从全局上下文收集，去重）。
	seen := map[string]bool{}
	for _, def := range ec.GC.Definitions() {
		el := def.Element
		if el == nil {
			continue
		}
		if seen[el.QualifiedName] {
			continue
		}
		seen[el.QualifiedName] = true
		snap.addElement(el)
	}

	// 关系。
	for _, r := range ec.Relations {
		snap.addRelation(r)
	}

	got := snap.sortAndRender()
	goldenPath := filepath.Join(RepoRoot(), goldenDir, "baseline.golden.jsonl")

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
			t.Fatalf("mkdir golden: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got+"\n"), 0644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("golden updated: %s (%d lines)", goldenPath, len(snap.entries))
		return
	}

	goldenData, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update to create)", goldenPath, err)
	}
	want := strings.TrimSuffix(string(goldenData), "\n")

	if got != want {
		t.Errorf("e2e snapshot mismatch:\n  got  %d entries\n  want %d entries\n  run: go test ./x/java/test -run TestE2EGolden -update",
			len(snap.entries), strings.Count(want, "\n")+1)
		dumpDiff(t, got, want)
	}
}

// resolveAllTestdata 收集全部 testdata 下的 .java 文件（e2e 覆盖全集）。
func resolveAllTestdata(t *testing.T) []string {
	t.Helper()
	var files []string
	err := filepath.Walk(TestDataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".java" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk testdata: %v", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatalf("no testdata java files found")
	}
	return files
}

// dumpDiff 输出差异首部（定位用）。
func dumpDiff(t *testing.T, got, want string) {
	gl := strings.Split(got, "\n")
	wl := strings.Split(want, "\n")
	max := len(gl)
	if len(wl) > max {
		max = len(wl)
	}
	shown := 0
	for i := 0; i < max && shown < 20; i++ {
		var g, w string
		if i < len(gl) {
			g = gl[i]
		}
		if i < len(wl) {
			w = wl[i]
		}
		if g != w {
			t.Errorf("  line %d:\n    got  %s\n    want %s", i+1, abbrev(g), abbrev(w))
			shown++
		}
	}
}

// abbrev 截断长行便于展示。
func abbrev(s string) string {
	if len(s) <= 160 {
		return s
	}
	return s[:160] + "..."
}
