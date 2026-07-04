package resolver_test

import (
	"fmt"
	"path/filepath"
	"strings"
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
	segmenter := resolver.NewExpressionSegmenter(fCtx)

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
	segmenter := resolver.NewExpressionSegmenter(fCtx)

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
			LineNum:     131,
			TargetText:  "value",
			ExpHeadType: resolver.HeadIdent,
			ExpHeadName: "container",
			ExpSegments: []resolver.ExpressionSegment{
				{Kind: resolver.SegmentField, Name: "innerList"},
				{Kind: resolver.SegmentMethod, Name: "get"},
				{Kind: resolver.SegmentField, Name: "inner2"},
				{Kind: resolver.SegmentField, Name: "value"},
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
	segmenter := resolver.NewExpressionSegmenter(fCtx)

	// 精准覆盖场景 7 条件表达式判断区、场景 9 集合操作中的链式调用
	expectations := []SegmentExpectation{
		{
			Name:        "场景7: 三元条件表达式判断区内的求值链路 (USE)",
			LineNum:     175,
			TargetText:  "data",
			ExpHeadType: resolver.HeadIdent,
			ExpHeadName: "container",
			ExpSegments: []resolver.ExpressionSegment{
				{Kind: resolver.SegmentField, Name: "inner1"},
				{Kind: resolver.SegmentField, Name: "data"},
				{Kind: resolver.SegmentMethod, Name: "length"}, // 补全真实物理长链
			},
		},
		{
			Name:        "场景9: 集合操作中的链式调用 (USE)",
			LineNum:     212,
			TargetText:  "inner2",
			ExpHeadType: resolver.HeadIdent,
			ExpHeadName: "container",
			ExpSegments: []resolver.ExpressionSegment{
				{Kind: resolver.SegmentMethod, Name: "getInnerList"},
				{Kind: resolver.SegmentMethod, Name: "get"},
				{Kind: resolver.SegmentField, Name: "inner2"},
				{Kind: resolver.SegmentMethod, Name: "transform"}, // 补全真实物理长链
			},
		},
	}

	runSegmentTestLoops(t, fCtx, q, ctxResolver, segmenter, model.Use, expectations)
}

// --- 🛠️ 抽象通用的双轨集成测试驱动引擎（带 USE 节点噪音过滤） ---
func runSegmentTestLoops(t *testing.T, fCtx *core.FileContext, q *sitter.Query, ctxResolver *resolver.NodeContextResolver, segmenter *resolver.ExpressionSegmenter, actType model.DependencyType, expectations []SegmentExpectation) {
	qc := sitter.NewQueryCursor()
	matches := qc.Matches(q, fCtx.RootNode, *fCtx.SourceBytes)

	// 使用行号作为第一层隔离，Value 为该行所有合法的拉平表达式链
	// 允许一行内存在多个不同的捕获链
	capturedChains := make(map[string]*resolver.ExpressionChain)

	// 用于防止同一个完整表达式被其内部的子 identifier 重复注入而覆盖正确结果
	// Key: 完整表达式的唯一区间范围 "start-end"
	seenExpressions := make(map[string]bool)

	// 建立快捷索引，只处理我们在 expectations 里声明的行，大幅过滤不相关的行
	targetLines := make(map[int]bool)
	for _, exp := range expectations {
		targetLines[exp.LineNum] = true
	}

	for {
		match := matches.Next()
		if match == nil {
			break
		}

		for _, cap := range match.Captures {
			lineNum := int(cap.Node.StartPosition().Row) + 1

			// 过滤 1: 如果当前行不是我们断言关心的行，直接跳过，过滤大量无关的 USE 捕获
			if !targetLines[lineNum] {
				continue
			}

			// 过滤 2: 验证捕获动作类型
			capName := q.CaptureNames()[cap.Index]
			if captureTypeMap[capName] != actType {
				continue
			}

			// 借助 NodeContextResolver 定位出完整的 ExpressNode
			res := ctxResolver.ResolveContext(actType, &cap.Node)
			if res == nil || res.ExpressNode == nil {
				continue
			}

			// 过滤 3: 去重同一个表达式触发的多次 identifier 捕获。
			// 比如 container.inner1.data 会触发 3 次捕获，但它们往上找的 ExpressNode 是同一个。
			// 我们只处理完整长表达式的第一次解析（Tree-sitter 深度优先遍历通常先遇到大表达式或特定边缘）
			exprKey := fmt.Sprintf("%d-%d", res.ExpressNode.StartByte(), res.ExpressNode.EndByte())

			// 将 ExpressNode 转化为被平铺的解析链条
			chain := segmenter.Segment(res.ExpressNode)
			if chain == nil {
				continue
			}

			rawText := cap.Node.Utf8Text(*fCtx.SourceBytes)

			// 过滤 4: 精准对齐。只保留当前捕获词刚好是链条最后一段名称，或者是链条 Head 名称的有效节点
			// 这模拟了 Extractor 的有效引用过滤
			isTargetComponent := false
			if chain.Head.Name == rawText {
				isTargetComponent = true
			}
			for _, seg := range chain.Segments {
				if seg.Name == rawText {
					isTargetComponent = true
					break
				}
			}

			if !isTargetComponent {
				continue
			}

			if seenExpressions[exprKey] {
				// 如果这个长表达式已经录入过了，后续子 identifier 的重复触发直接略过
				continue
			}
			seenExpressions[exprKey] = true

			// 拼装精准 Key
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
				// 容错降级：如果 TargetText 是该行某个长表达式的子段，动态补正
				for key, chain := range capturedChains {
					if strings.HasPrefix(key, fmt.Sprintf("%d:", exp.LineNum)) {
						// 检查这个链里是否包含期望的末端目标
						for _, seg := range chain.Segments {
							if seg.Name == exp.TargetText {
								actualChain = chain
								found = true
								break
							}
						}
					}
					if found {
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
