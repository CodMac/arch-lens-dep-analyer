// Package test e2e 黄金回归测试：跑完整五阶段流水线，将最终元素与关系序列化为
// 确定性快照，与 testdata/e2e 下的 .golden.jsonl 基线比对。
//
// 更新基线：go test ./x/cpp/test -run TestE2EGolden -update
package test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

var updateGolden = flag.Bool("update", false, "update golden files")

const goldenDir = "x/cpp/testdata/e2e"

// goldenEntry 是快照中的一行：元素或关系。
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

type e2eSnapshot struct {
	entries []goldenEntry
}

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
		loc = r.Location.FilePath + ":" + strconv.Itoa(r.Location.StartLine)
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

func (s *e2eSnapshot) sortAndRender() string {
	sort.SliceStable(s.entries, func(i, j int) bool {
		a, b := s.entries[i], s.entries[j]
		if a.IsSource != b.IsSource {
			return a.IsSource
		}
		if a.QN != b.QN {
			return a.QN < b.QN
		}
		// 声明/定义同 QN 共存（FUNCTION_DECL vs FUNCTION）：以 Kind 作确定性次序。
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
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

	ec := RunPipeline(t, "x/cpp/testdata", paths, core.LevelBalanced)

	// 声明/定义同 QN 共存（FUNCTION_DECL vs FUNCTION），去重键含 Kind。
	seen := map[string]bool{}
	for _, def := range ec.GC.Definitions() {
		el := def.Element
		if el == nil {
			continue
		}
		key := el.QualifiedName + "|" + string(el.Kind)
		if seen[key] {
			continue
		}
		seen[key] = true
		snap.addElement(el)
	}
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
		t.Errorf("e2e snapshot mismatch:\n  got  %d entries\n  want %d entries\n  run: go test ./x/cpp/test -run TestE2EGolden -update",
			len(snap.entries), strings.Count(want, "\n")+1)
		dumpDiff(t, got, want)
	}
}

// resolveAllTestdata 收集全部 testdata 下的 C/C++ 源文件。
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
		switch filepath.Ext(path) {
		case ".c", ".h", ".cpp", ".hpp", ".cc", ".cxx":
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk testdata: %v", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatalf("no testdata source files found")
	}
	return files
}

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

func abbrev(s string) string {
	if len(s) <= 160 {
		return s
	}
	return s[:160] + "..."
}
