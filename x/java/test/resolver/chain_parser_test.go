package java_test

import (
	"path/filepath"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/resolver"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type StepExpectation struct {
	Name    string
	IsCall  bool
	IsField bool
	IsNew   bool
}

type ChainExpectation struct {
	ReceiverType    resolver.ReceiverType
	ReceiverRawText string
	ExpectedSteps   []StepExpectation
}

func TestChainParser_ForCall(t *testing.T) {
	testFile := test.GetTestFilePath(filepath.Join("resolver", "ChainCallComplexCase.java"))
	files := []string{testFile}

	gCtx := test.RunPhase1Collection(t, files)

	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}
	test.PrintRelationsOnKinds(allRelations, []model.DependencyType{model.Call})

	fCtx := gCtx.FileContexts[files[0]]
	chainParser := resolver.NewChainParser(gCtx, fCtx)

	testCases := []struct {
		name         string
		relType      model.DependencyType
		relSourceQN  string
		relRawText   string
		expectations ChainExpectation
	}{
		{
			name:        "Scenario1: container.inner1.processData()",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.fieldAccessThenMethodCall()",
			relRawText:  "container.inner1.processData()",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.inner1.processData()",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "inner1", IsCall: false, IsField: true},
					{Name: "processData", IsCall: true, IsField: false},
				},
			},
		},
		{
			name:        "Scenario2: container.getInner1().processData()",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.continuousMethodCalls()",
			relRawText:  "container.getInner1().processData()",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.getInner1().processData()",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "getInner1", IsCall: true, IsField: false},
					{Name: "processData", IsCall: true, IsField: false},
				},
			},
		},
		{
			name:        "Scenario2: container.getInner1().getInner2().transform()",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.continuousMethodCalls()",
			relRawText:  "container.getInner1().getInner2().transform()",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.getInner1().getInner2().transform()",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "getInner1", IsCall: true, IsField: false},
					{Name: "getInner2", IsCall: true, IsField: false},
					{Name: "transform", IsCall: true, IsField: false},
				},
			},
		},
		{
			name:        "Scenario2: container.getInner1().getInner2().getInner3().format()",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.continuousMethodCalls()",
			relRawText:  "container.getInner1().getInner2().getInner3().format()",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.getInner1().getInner2().getInner3().format()",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "getInner1", IsCall: true, IsField: false},
					{Name: "getInner2", IsCall: true, IsField: false},
					{Name: "getInner3", IsCall: true, IsField: false},
					{Name: "format", IsCall: true, IsField: false},
				},
			},
		},
		{
			name:        "Scenario3: container.getInner1().processData()",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.mixedChainCalls()",
			relRawText:  "container.getInner1().processData()",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.getInner1().processData()",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "getInner1", IsCall: true, IsField: false},
					{Name: "processData", IsCall: true, IsField: false},
				},
			},
		},
		{
			name:        "Scenario3: container.setResult(\"new value\").getResult()",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.mixedChainCalls()",
			relRawText:  "container.setResult(\"new value\").getResult()",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.setResult(\"new value\").getResult()",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "setResult", IsCall: true, IsField: false},
					{Name: "getResult", IsCall: true, IsField: false},
				},
			},
		},
		{
			name:        "Scenario4: container.getInnerList().get(0).inner2.transform()",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.methodThenFieldAccessThenMethod()",
			relRawText:  "container.getInnerList()\n                               .get(0)\n                               .inner2\n                               .transform()",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.getInnerList()\n                               .get(0)\n                               .inner2\n                               .transform()",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "getInnerList", IsCall: true, IsField: false},
					{Name: "get", IsCall: true, IsField: false},
					{Name: "inner2", IsCall: false, IsField: true},
					{Name: "transform", IsCall: true, IsField: false},
				},
			},
		},
		{
			name:        "Scenario5: container.getInner1().processData().concat(...)",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.complexBuilderPattern()",
			relRawText:  "container.getInner1().processData()",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.getInner1().processData()",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "getInner1", IsCall: true, IsField: false},
					{Name: "processData", IsCall: true, IsField: false},
				},
			},
		},
		{
			name:        "Scenario5: container.getInner1().getInner2().transform()",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.complexBuilderPattern()",
			relRawText:  "container.getInner1().getInner2().transform()",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.getInner1().getInner2().transform()",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "getInner1", IsCall: true, IsField: false},
					{Name: "getInner2", IsCall: true, IsField: false},
					{Name: "transform", IsCall: true, IsField: false},
				},
			},
		},
		{
			name:        "Scenario6: String.valueOf(container.getInner1().processData().length())",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.mixedStaticAndInstance()",
			relRawText:  "container.getInner1().processData().length()",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.getInner1().processData().length()",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "getInner1", IsCall: true, IsField: false},
					{Name: "processData", IsCall: true, IsField: false},
					{Name: "length", IsCall: true, IsField: false},
				},
			},
		},
		{
			name:        "Scenario7: container.getInner1().processData().toUpperCase()",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.conditionalChainedCalls()",
			relRawText:  "container.getInner1().processData().toUpperCase()",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.getInner1().processData().toUpperCase()",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "getInner1", IsCall: true, IsField: false},
					{Name: "processData", IsCall: true, IsField: false},
					{Name: "toUpperCase", IsCall: true, IsField: false},
				},
			},
		},
		{
			name:        "Scenario7: container.getInner1().getInner2().transform().toLowerCase()",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.conditionalChainedCalls()",
			relRawText:  "container.getInner1().getInner2().transform().toLowerCase()",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.getInner1().getInner2().transform().toLowerCase()",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "getInner1", IsCall: true, IsField: false},
					{Name: "getInner2", IsCall: true, IsField: false},
					{Name: "transform", IsCall: true, IsField: false},
					{Name: "toLowerCase", IsCall: true, IsField: false},
				},
			},
		},
		{
			name:        "Scenario8: container.inner1.inner2.inner3.format().concat().concat()",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.deeplyNestedMixedAccess()",
			relRawText:  "container.inner1.inner2.inner3.format().concat(\"_\").concat(container.getInner1().getInner2().transform())",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.inner1.inner2.inner3.format().concat(\"_\").concat(container.getInner1().getInner2().transform())",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "inner1", IsCall: false, IsField: true},
					{Name: "inner2", IsCall: false, IsField: true},
					{Name: "inner3", IsCall: false, IsField: true},
					{Name: "format", IsCall: true, IsField: false},
					{Name: "concat", IsCall: true, IsField: false},
					{Name: "concat", IsCall: true, IsField: false},
				},
			},
		},
		{
			name:        "Scenario8: container.getInner1().getInner2().transform()",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.deeplyNestedMixedAccess()",
			relRawText:  "container.getInner1().getInner2().transform()",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.getInner1().getInner2().transform()",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "getInner1", IsCall: true, IsField: false},
					{Name: "getInner2", IsCall: true, IsField: false},
					{Name: "transform", IsCall: true, IsField: false},
				},
			},
		},
		{
			name:        "Scenario9: container.getInnerList().get(1).inner2.transform()",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.collectionChainedCalls()",
			relRawText:  "container.getInnerList().get(1).inner2.transform()",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.getInnerList().get(1).inner2.transform()",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "getInnerList", IsCall: true, IsField: false},
					{Name: "get", IsCall: true, IsField: false},
					{Name: "inner2", IsCall: false, IsField: true},
					{Name: "transform", IsCall: true, IsField: false},
				},
			},
		},
		{
			name:        "Scenario9: container.getInnerList().get(2).processData()",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.collectionChainedCalls()",
			relRawText:  "container.getInnerList().get(2).processData()",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.getInnerList().get(2).processData()",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "getInnerList", IsCall: true, IsField: false},
					{Name: "get", IsCall: true, IsField: false},
					{Name: "processData", IsCall: true, IsField: false},
				},
			},
		},
		{
			name:        "Scenario10: container.getInner1().processData().toUpperCase()",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.parameterChainedCalls()",
			relRawText:  "container.getInner1().processData().toUpperCase()",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.getInner1().processData().toUpperCase()",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "getInner1", IsCall: true, IsField: false},
					{Name: "processData", IsCall: true, IsField: false},
					{Name: "toUpperCase", IsCall: true, IsField: false},
				},
			},
		},
		{
			name:        "Scenario10: container.getInner1().getInner2().transform().trim()",
			relType:     model.Call,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.parameterChainedCalls()",
			relRawText:  "container.getInner1().getInner2().transform().trim()",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.getInner1().getInner2().transform().trim()",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "getInner1", IsCall: true, IsField: false},
					{Name: "getInner2", IsCall: true, IsField: false},
					{Name: "transform", IsCall: true, IsField: false},
					{Name: "trim", IsCall: true, IsField: false},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			receiverNode := findReceiverNode(allRelations, tc.relType, tc.relSourceQN, tc.relRawText)
			if receiverNode == nil {
				t.Fatalf("Receiver node not found for relation '%s'", tc.relRawText)
			}

			receiver := chainParser.ParseReceiverFromNode(receiverNode)
			if receiver == nil {
				t.Fatalf("Receiver is nil for %s", tc.name)
			}

			t.Logf("=== 测试: %s ===", tc.name)
			t.Logf("关系文本: %s", tc.relRawText)
			t.Logf("原始文本: %s", receiver.RawText)

			if receiver.Chained == nil {
				t.Fatalf("期望 Chained 上下文, 但实际为 nil")
			}

			t.Logf("根 Receiver 类型: %d", receiver.Type)

			if receiver.Type != tc.expectations.ReceiverType {
				t.Errorf("Receiver类型不匹配: 期望 %d, 实际 %d", tc.expectations.ReceiverType, receiver.Type)
			}

			if receiver.RawText != tc.expectations.ReceiverRawText {
				t.Errorf("Receiver原始文本不匹配: 期望 %s, 实际 %s", tc.expectations.ReceiverRawText, receiver.RawText)
			}

			steps := receiver.Chained.Steps
			expectedSteps := tc.expectations.ExpectedSteps

			if len(steps) != len(expectedSteps) {
				t.Errorf("步骤数量不匹配: 期望 %d, 实际 %d", len(expectedSteps), len(steps))
				t.Logf("期望的步骤: %v", expectedSteps)
				t.Logf("实际的步骤: %v", steps)
			}

			minSteps := len(expectedSteps)
			if len(steps) < minSteps {
				minSteps = len(steps)
			}

			for i := 0; i < minSteps; i++ {
				actual := steps[i]
				expected := expectedSteps[i]

				t.Logf("步骤 %d:", i)
				t.Logf("  期望: Name='%s', IsCall=%v, IsField=%v, IsNew=%v", expected.Name, expected.IsCall, expected.IsField, expected.IsNew)
				t.Logf("  实际: Name='%s', IsCall=%v, IsField=%v, IsNew=%v", actual.Name, actual.IsCall, actual.IsField, actual.IsNew)

				if actual.Name != expected.Name {
					t.Errorf("步骤 %d 名称不匹配: 期望 '%s', 实际 '%s'", i, expected.Name, actual.Name)
				}
				if actual.IsCall != expected.IsCall {
					t.Errorf("步骤 %d IsCall 不匹配: 期望 %v, 实际 %v", i, expected.IsCall, actual.IsCall)
				}
				if actual.IsField != expected.IsField {
					t.Errorf("步骤 %d IsField 不匹配: 期望 %v, 实际 %v", i, expected.IsField, actual.IsField)
				}
				if actual.IsNew != expected.IsNew {
					t.Errorf("步骤 %d IsNew 不匹配: 期望 %v, 实际 %v", i, expected.IsNew, actual.IsNew)
				}
			}
		})
	}
}

func TestChainParser_ForUseAndAssign(t *testing.T) {
	testFile := test.GetTestFilePath(filepath.Join("resolver", "ChainCallComplexCase.java"))
	files := []string{testFile}

	gCtx := test.RunPhase1Collection(t, files)

	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}
	test.PrintRelations(allRelations)

	fCtx := gCtx.FileContexts[files[0]]
	chainParser := resolver.NewChainParser(gCtx, fCtx)

	// 调试：输出关键关系的ContextNode信息
	t.Log("=== 调试输出：ContextNode信息 ===")
	foundUseRelation := false
	foundAssignRelation := false
	for _, rel := range allRelations {
		if (rel.Type == model.Use || rel.Type == model.Assign) && rel.Mores != nil {
			if rawText, ok := rel.Mores["java.rel.raw_text"].(string); ok {
				// 查找包含 container.inner1 的关系
				if rel.Type == model.Use && !foundUseRelation && (rawText == "container.inner1" || rawText == "container.innerList") {
					foundUseRelation = true
					t.Logf("找到USE关系: raw_text: %s", rawText)

					if ctxNode, ok := rel.Mores["tmp_ctx_node"].(interface{}); ok {
						if node, ok := ctxNode.(*sitter.Node); ok {
							t.Logf("  tmp_ctx_node kind: %s", node.Kind())
							t.Logf("  tmp_ctx_node text: %s", node.Utf8Text(*fCtx.SourceBytes))

							// 输出父节点链
							parent := node.Parent()
							level := 1
							for parent != nil && level <= 5 {
								t.Logf("  parent[%d] kind: %s, text: %s", level, parent.Kind(),
									parent.Utf8Text(*fCtx.SourceBytes))
								parent = parent.Parent()
								level++
							}
						}
					}
				}

				if rel.Type == model.Assign && !foundAssignRelation && (rawText == "container.inner1" || rawText == "container.innerList") {
					foundAssignRelation = true
					t.Logf("找到ASSIGN关系: raw_text: %s", rawText)

					if ctxNode, ok := rel.Mores["tmp_ctx_node"].(interface{}); ok {
						if node, ok := ctxNode.(*sitter.Node); ok {
							t.Logf("  tmp_ctx_node kind: %s", node.Kind())
							t.Logf("  tmp_ctx_node text: %s", node.Utf8Text(*fCtx.SourceBytes))

							// 输出父节点链
							parent := node.Parent()
							level := 1
							for parent != nil && level <= 5 {
								t.Logf("  parent[%d] kind: %s, text: %s", level, parent.Kind(),
									parent.Utf8Text(*fCtx.SourceBytes))
								parent = parent.Parent()
								level++
							}
						}
					}
				}

				if foundUseRelation && foundAssignRelation {
					break
				}
			}
		}
	}
	t.Log("=== 调试输出结束 ===")

	testCases := []struct {
		name         string
		relType      model.DependencyType
		relSourceQN  string
		relRawText   string
		expectations ChainExpectation
	}{
		{
			name:        "Scenario1: container.inner1 - USE (field access)",
			relType:     model.Use,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.fieldAccessThenMethodCall()",
			relRawText:  "container.inner1",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.inner1",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "inner1", IsCall: false, IsField: true},
				},
			},
		},
		{
			name:        "Scenario1: container.inner1 - ASSIGN",
			relType:     model.Assign,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.fieldAccessThenMethodCall()",
			relRawText:  "container.inner1",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.inner1",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "inner1", IsCall: false, IsField: true},
				},
			},
		},
		{
			name:        "Scenario3: container.inner1 - USE (field access)",
			relType:     model.Use,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.mixedChainCalls()",
			relRawText:  "container.inner1",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.inner1",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "inner1", IsCall: false, IsField: true},
				},
			},
		},
		{
			name:        "Scenario4: container.innerList - USE (field access)",
			relType:     model.Use,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.methodThenFieldAccessThenMethod()",
			relRawText:  "container.innerList",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.innerList",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "innerList", IsCall: false, IsField: true},
				},
			},
		},
		{
			name:        "Scenario4: container.innerList - ASSIGN",
			relType:     model.Assign,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.methodThenFieldAccessThenMethod()",
			relRawText:  "container.innerList",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.innerList",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "innerList", IsCall: false, IsField: true},
				},
			},
		},
		{
			name:        "Scenario5: container.inner1 - USE (field access)",
			relType:     model.Use,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.complexBuilderPattern()",
			relRawText:  "container.inner1",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.inner1",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "inner1", IsCall: false, IsField: true},
				},
			},
		},
		{
			name:        "Scenario6: container.inner1 - USE (field access)",
			relType:     model.Use,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.mixedStaticAndInstance()",
			relRawText:  "container.inner1",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.inner1",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "inner1", IsCall: false, IsField: true},
				},
			},
		},
		{
			name:        "Scenario7: container.inner1 - USE (field access)",
			relType:     model.Use,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.conditionalChainedCalls()",
			relRawText:  "container.inner1",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.inner1",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "inner1", IsCall: false, IsField: true},
				},
			},
		},
		{
			name:        "Scenario8: container.inner1 - USE (simple field access)",
			relType:     model.Use,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.deeplyNestedMixedAccess()",
			relRawText:  "container.inner1",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.inner1",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "inner1", IsCall: false, IsField: true},
				},
			},
		},
		{
			name:        "Scenario9: container.innerList - USE (field access)",
			relType:     model.Use,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.collectionChainedCalls()",
			relRawText:  "container.innerList",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.innerList",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "innerList", IsCall: false, IsField: true},
				},
			},
		},
		{
			name:        "Scenario9: container.innerList - ASSIGN",
			relType:     model.Assign,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.collectionChainedCalls()",
			relRawText:  "container.innerList",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.innerList",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "innerList", IsCall: false, IsField: true},
				},
			},
		},
		{
			name:        "Scenario10: container.inner1 - USE (field access)",
			relType:     model.Use,
			relSourceQN: "com.example.resolver.chain.ChainCallComplexCase.parameterChainedCalls()",
			relRawText:  "container.inner1",
			expectations: ChainExpectation{
				ReceiverType:    resolver.ReceiverChained,
				ReceiverRawText: "container.inner1",
				ExpectedSteps: []StepExpectation{
					{Name: "container", IsCall: false, IsField: false},
					{Name: "inner1", IsCall: false, IsField: true},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			receiverNode := findReceiverNode(allRelations, tc.relType, tc.relSourceQN, tc.relRawText)
			if receiverNode == nil {
				t.Fatalf("Receiver node not found for relation '%s'", tc.relRawText)
			}

			receiver := chainParser.ParseReceiverFromNode(receiverNode)
			if receiver == nil {
				t.Fatalf("Receiver is nil for %s", tc.name)
			}

			t.Logf("=== 测试: %s ===", tc.name)
			t.Logf("关系类型: %s", tc.relType)
			t.Logf("关系文本: %s", tc.relRawText)
			t.Logf("原始文本: %s", receiver.RawText)

			if receiver.Chained == nil {
				t.Fatalf("期望 Chained 上下文, 但实际为 nil")
			}

			t.Logf("根 Receiver 类型: %d", receiver.Type)

			if receiver.Type != tc.expectations.ReceiverType {
				t.Errorf("Receiver类型不匹配: 期望 %d, 实际 %d", tc.expectations.ReceiverType, receiver.Type)
			}

			if receiver.RawText != tc.expectations.ReceiverRawText {
				t.Errorf("Receiver原始文本不匹配: 期望 %s, 实际 %s", tc.expectations.ReceiverRawText, receiver.RawText)
			}

			steps := receiver.Chained.Steps
			expectedSteps := tc.expectations.ExpectedSteps

			if len(steps) != len(expectedSteps) {
				t.Errorf("步骤数量不匹配: 期望 %d, 实际 %d", len(expectedSteps), len(steps))
				t.Logf("期望的步骤: %v", expectedSteps)
				t.Logf("实际的步骤: %v", steps)
			}

			minSteps := len(expectedSteps)
			if len(steps) < minSteps {
				minSteps = len(steps)
			}

			for i := 0; i < minSteps; i++ {
				actual := steps[i]
				expected := expectedSteps[i]

				t.Logf("步骤 %d:", i)
				t.Logf("  期望: Name='%s', IsCall=%v, IsField=%v, IsNew=%v", expected.Name, expected.IsCall, expected.IsField, expected.IsNew)
				t.Logf("  实际: Name='%s', IsCall=%v, IsField=%v, IsNew=%v", actual.Name, actual.IsCall, actual.IsField, actual.IsNew)

				if actual.Name != expected.Name {
					t.Errorf("步骤 %d 名称不匹配: 期望 '%s', 实际 '%s'", i, expected.Name, actual.Name)
				}
				if actual.IsCall != expected.IsCall {
					t.Errorf("步骤 %d IsCall 不匹配: 期望 %v, 实际 %v", i, expected.IsCall, actual.IsCall)
				}
				if actual.IsField != expected.IsField {
					t.Errorf("步骤 %d IsField 不匹配: 期望 %v, 实际 %v", i, expected.IsField, actual.IsField)
				}
				if actual.IsNew != expected.IsNew {
					t.Errorf("步骤 %d IsNew 不匹配: 期望 %v, 实际 %v", i, expected.IsNew, actual.IsNew)
				}
			}
		})
	}
}

func findReceiverNode(rels []*model.DependencyRelation, relType model.DependencyType, relSourceQN, relRawText string) *sitter.Node {
	for _, rel := range rels {
		if rel.Type == relType && rel.Source.QualifiedName == relSourceQN && rel.Mores[constants.RelRawText] == relRawText {
			return rel.Mores[constants.TmpCtxNode].(*sitter.Node)
		}
	}

	return nil
}
