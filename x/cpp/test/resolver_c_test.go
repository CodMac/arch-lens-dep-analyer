package test

import (
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/constants"
)

// TestResolverAnonymousTypedefField 覆盖匿名 struct 承载 typedef 的字段解析：
// typedef struct {...} Point; 的变量 p 其 type_with_qn 必须是 Point（而非匿名结构体文本），
// 否则 p.x 会退化为全局短名扫描，命中同名参数 x（并行收集顺序不定 → 结果不稳定）。
func TestResolverAnonymousTypedefField(t *testing.T) {
	paths := ResolvePaths(t, "resolver/c")
	gc, rels := RunExtraction(t, "x/cpp/testdata", paths)

	// Binder 必须把 p 的类型绑定到 Point 本身，而不是匿名结构体文本。
	if entry, ok := gc.FindByQualifiedName("process()::p"); !ok {
		t.Fatal("process()::p not found")
	} else if got, _ := entry.Element.GetMores()[constants.VariableTypeWithQN].(string); got != "Point" {
		t.Errorf("process()::p type_with_qn = %q, want Point", got)
	}

	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Assign && r.Source.QualifiedName == "process()" &&
			r.Target.QualifiedName == "Point::x"
	})
	FindRel(t, rels, func(r *model.DependencyRelation) bool {
		return r.Type == model.Use && r.Source.QualifiedName == "process()" &&
			r.Target.QualifiedName == "Point::x"
	})
}
