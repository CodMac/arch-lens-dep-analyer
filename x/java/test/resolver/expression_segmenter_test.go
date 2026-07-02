package resolver

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/resolver"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// SegmentExpectation 定义单个分段测试的断言指标
type SegmentExpectation struct {
	Name        string
	LineNum     int
	TargetText  string // 触发捕获的原始核心文本
	ExpHeadType resolver.ExpressionHeadType
	ExpHeadName string
	ExpSegments []resolver.ExpressionSegment // 期望被拉平求值的后续分段顺序
}

// --- 🎯 1. CALL 动作类型的分段测试用例 ---
func TestExpressionSegmenter_ResolveCall(t *testing.T) {
	testFile := test.GetTestFilePath(filepath.Join("resolver", "ExpressionSegmentCase.java"))
	gCtx := test.RunPhase1Collection(t, []string{testFile})
	fCtx := gCtx.FileContexts[testFile]

	tsLang, _ := core.GetLanguage(core.LangJava)
	q, err := sitter.NewQuery(tsLang, java.JavaActionQuery)
	if err != nil {
		t.Fatalf("Failed to compile JavaActionQuery: %v", err)
	}
	defer q.Close()

	ctxResolver := resolver.NewNodeContextResolver(fCtx)
	segmenter := resolver.NewExpressionSegmenter(fCtx.SourceBytes)

	// 精准覆盖场景 1, 场景 2, 场景 6 等 CALL 链条
	expectations := []SegmentExpectation{
		{
			Name:        "场景1: 经典字段后方法调用 (CALL)",
			LineNum:     82,
			TargetText:  "processData",
			ExpHeadType: resolver.HeadIdent,
			ExpHeadName: "container",
			ExpSegments: []resolver.ExpressionSegment{
				{Kind: resolver.SegmentField, Name: "inner1"},
				{Kind: resolver.SegmentMethod, Name: "processData"},
			},
		},
		{
			Name:        "场景2: 连续链式方法调用 - 尾部format()调用 (CALL)",
			LineNum:     100,
			TargetText:  "format",
			ExpHeadType: resolver.HeadIdent,
			ExpHeadName: "container",
			ExpSegments: []resolver.ExpressionSegment{
				{Kind: resolver.SegmentMethod, Name: "getInner1"},
				{Kind: resolver.SegmentMethod, Name: "getInner2"},
				{Kind: resolver.SegmentMethod, Name: "getInner3"},
				{Kind: resolver.SegmentMethod, Name: "format"},
			},
		},
		{
			Name:        "场景6: 静态入口出发的混合链 (CALL)",
			LineNum:     161,
			TargetText:  "valueOf",
			ExpHeadType: resolver.HeadIdent,
			ExpHeadName: "String", // 静态类名作为 Head
			ExpSegments: []resolver.ExpressionSegment{
				{Kind: resolver.SegmentMethod, Name: "valueOf"},
			},
		},
	}

	runSegmentTestLoops(t, fCtx, q, ctxResolver, segmenter, model.Call, expectations)
}

// --- 🎯 2. ASSIGN 动作类型的分段测试用例 ---
func TestExpressionSegmenter_ResolveAssign(t *testing.T) {
	testFile := test.GetTestFilePath(filepath.Join("resolver", "ExpressionSegmentCase.java"))
	gCtx := test.RunPhase1Collection(t, []string{testFile})
	fCtx := gCtx.FileContexts[testFile]

	tsLang, _ := core.GetLanguage(core.LangJava)
	q, err := sitter.NewQuery(tsLang, java.JavaActionQuery)
	if err != nil {
		t.Fatalf("Failed to compile JavaActionQuery: %v", err)
	}
	defer q.Close()

	ctxResolver := resolver.NewNodeContextResolver(fCtx)
	segmenter := resolver.NewExpressionSegmenter(fCtx.SourceBytes)

	// 精准覆盖场景 1、场景 4 中的被赋值左值或赋值右值主链
	expectations := []SegmentExpectation{
		{
			Name:        "场景1: 赋值语句左侧字段链路 (ASSIGN)",
			LineNum:     78,
			TargetText:  "inner1",
			ExpHeadType: resolver.HeadIdent,
			ExpHeadName: "container",
			ExpSegments: []resolver.ExpressionSegment{
				{Kind: resolver.SegmentField, Name: "inner1"},
			},
		},
		{
			Name:        "场景4: 方法返回后接字段的混合赋值右值 (ASSIGN)",
			LineNum:     135,
			TargetText:  "transform",
			ExpHeadType: resolver.HeadIdent,
			ExpHeadName: "container",
			ExpSegments: []resolver.ExpressionSegment{
				{Kind: resolver.SegmentMethod, Name: "getInnerList"},
				{Kind: resolver.SegmentMethod, Name: "get"},
				{Kind: resolver.SegmentArray, Name: ""}, // 包含集成的数组降维/提取
				{Kind: resolver.SegmentField, Name: "inner2"},
				{Kind: resolver.SegmentMethod, Name: "transform"},
			},
		},
	}

	runSegmentTestLoops(t, fCtx, q, ctxResolver, segmenter, model.Assign, expectations)
}

// --- 🎯 3. USE 动作类型的分段测试用例 ---
func TestExpressionSegmenter_ResolveUse(t *testing.T) {
	testFile := test.GetTestFilePath(filepath.Join("resolver", "ExpressionSegmentCase.java"))
	gCtx := test.RunPhase1Collection(t, []string{testFile})
	fCtx := gCtx.FileContexts[testFile]

	tsLang, _ := core.GetLanguage(core.LangJava)
	q, err := sitter.NewQuery(tsLang, java.JavaActionQuery)
	if err != nil {
		t.Fatalf("Failed to compile JavaActionQuery: %v", err)
	}
	defer q.Close()

	ctxResolver := resolver.NewNodeContextResolver(fCtx)
	segmenter := resolver.NewExpressionSegmenter(fCtx.SourceBytes)

	// 精准覆盖场景 7 条件表达式判断区、场景 10 方法传参内的纯引用求值链路
	expectations := []SegmentExpectation{
		{
			Name:        "场景7: 三元条件表达式判断区内的求值链路 (USE)",
			LineNum:     175,
			TargetText:  "length",
			ExpHeadType: resolver.HeadIdent,
			ExpHeadName: "container",
			ExpSegments: []resolver.ExpressionSegment{
				{Kind: resolver.SegmentField, Name: "inner1"},
				{Kind: resolver.SegmentField, Name: "data"},
				{Kind: resolver.SegmentMethod, Name: "length"},
			},
		},
		{
			Name:        "场景10: 方法外层传参包裹内的链式调用 (USE)",
			LineNum:     200,
			TargetText:  "processData",
			ExpHeadType: resolver.HeadIdent,
			ExpHeadName: "container",
			ExpSegments: []resolver.ExpressionSegment{
				{Kind: resolver.SegmentMethod, Name: "getInner1"},
				{Kind: resolver.SegmentMethod, Name: "processData"},
				{Kind: resolver.SegmentMethod, Name: "toUpperCase"},
			},
		},
	}

	runSegmentTestLoops(t, fCtx, q, ctxResolver, segmenter, model.Use, expectations)
}

// --- 🛠️ 抽象通用的双轨集成测试驱动引擎 ---
func runSegmentTestLoops(t *testing.T, fCtx *core.FileContext, q *sitter.Query, ctxResolver *resolver.NodeContextResolver, segmenter *resolver.ExpressionSegmenter, actType model.DependencyType, expectations []SegmentExpectation) {
	qc := sitter.NewQueryCursor()
	matches := qc.Matches(q, fCtx.RootNode, *fCtx.SourceBytes)

	capturedChains := make(map[string]*resolver.ExpressionChain)

	for {
		match := matches.Next()
		if match == nil {
			break
		}

		for _, cap := range match.Captures {
			// 过滤非当前动作类型的触发目标（可兼容 call_target, assign_target, use_target 等规则命名）
			lineNum := int(cap.Node.StartPosition().Row) + 1
			rawText := cap.Node.Utf8Text(*fCtx.SourceBytes)

			// 借助第一阶段已实现的 NodeContextResolver 定位出完整的 ExpressNode
			res := ctxResolver.ResolveContext(actType, &cap.Node)
			if res == nil || res.ExpressNode == nil {
				continue
			}

			// 移交第二阶段待测的 ExpressionSegmenter 转化为被平铺的解析链条
			chain := segmenter.Segment(res.ExpressNode)
			if chain == nil {
				continue
			}

			uniqueKey := fmt.Sprintf("%d:%s", lineNum, rawText)
			capturedChains[uniqueKey] = chain
		}
	}

	// 验证所有目标点位的拉平求值拓扑结构
	for _, exp := range expectations {
		t.Run(exp.Name, func(t *testing.T) {
			targetKey := fmt.Sprintf("%d:%s", exp.LineNum, exp.TargetText)
			actualChain, found := capturedChains[targetKey]
			if !found {
				// 容错降级：如果 TargetText 是长表达式的一部分，进行前缀或行号动态补正
				for key, chain := range capturedChains {
					if fmt.Sprintf("%d:", exp.LineNum) == key[:len(fmt.Sprintf("%d:", exp.LineNum))] {
						actualChain = chain
						found = true
						break
					}
				}
			}

			if !found {
				t.Fatalf("【断言失败】未能在行号 %d 处捕捉到动作为 %v 且关联文本为 %s 的表达式链", exp.LineNum, actType, exp.TargetText)
			}

			// 1. 验证链条起点 (Head) 的准确度
			if actualChain.Head.Type != exp.ExpHeadType {
				t.Errorf("Head 类型不匹配. 期望: %v, 实际: %v (文本: %s)", exp.ExpHeadType, actualChain.Head.Type, actualChain.Head.RawText)
			}
			if actualChain.Head.Name != exp.ExpHeadName {
				t.Errorf("Head 标识符不匹配. 期望: %s, 实际: %s", exp.ExpHeadName, actualChain.Head.Name)
			}

			// 2. 验证递进分段 (Segments) 的求值拓扑顺序与广度是否一致
			if len(actualChain.Segments) != len(exp.ExpSegments) {
				t.Fatalf("Segments 拓扑长度不匹配.\n期望级数: %d\n实际级数: %d\n完整解析链: %s", len(exp.ExpSegments), len(actualChain.Segments), actualChain.RawText)
			}

			for i, segExpect := range exp.ExpSegments {
				actualSeg := actualChain.Segments[i]
				if actualSeg.Kind != segExpect.Kind {
					t.Errorf("第 [%d] 步分段依赖性质 Kind 错位. 期望: %v, 实际: %v", i, segExpect.Kind, actualSeg.Kind)
				}
				if segExpect.Name != "" && actualSeg.Name != segExpect.Name {
					t.Errorf("第 [%d] 步分段映射标识符 Name 错误. 期望: %s, 实际: %s", i, segExpect.Name, actualSeg.Name)
				}
			}
		})
	}
}
