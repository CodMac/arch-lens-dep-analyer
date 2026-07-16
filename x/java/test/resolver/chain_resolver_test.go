package resolver_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

// CreateExpectation 定义 Create 场景下最终符号解析的预期
type CreateExpectation struct {
	Name             string            // 场景描述
	LineNum          int               // 触发依赖关系的行号
	TargetText       string            // 期望被 NodeContext 捕获的表达式文本
	ExpQualifiedName string            // 期望解析出的最终符号全限定名 (QN)
	ExpKind          model.ElementKind // 期望的符号类型 (Kind)
	ExpIsExternal    bool              // 期望是否是外部符号 (IsFormExternal)
}

func TestSymbolResolver_Create(t *testing.T) {
	// 1. 获取测试文件路径
	mainTestFile := test.GetTestFilePath(filepath.Join("resolver", "chain_resolve", "create", "CreateCase1.java"))
	dependencyFile1 := test.GetTestFilePath(filepath.Join("resolver", "chain_resolve", "create", "case1", "Outer.java"))
	dependencyFile2 := test.GetTestFilePath(filepath.Join("resolver", "chain_resolve", "create", "case1", "Builder.java"))
	dependencyFile3 := test.GetTestFilePath(filepath.Join("resolver", "chain_resolve", "create", "case1", "Product.java"))
	dependencyFile4 := test.GetTestFilePath(filepath.Join("resolver", "chain_resolve", "create", "case1", "User.java"))

	// 2. 关键：将主测试文件和跨包依赖文件一同送入 Phase 1 Collection，使其在 GlobalContext 中建立符号索引
	gCtx := test.RunPhase1Collection(t, []string{mainTestFile, dependencyFile1, dependencyFile2, dependencyFile3, dependencyFile4})

	// 提取主测试文件的 FileContext 作为分析上下文
	fCtx, exists := gCtx.FileContexts[mainTestFile]
	if !exists {
		t.Fatalf("无法在 GlobalContext 中找到主测试文件上下文: %s", mainTestFile)
	}

	// 3. 初始化 Java 提取器与一元化/符号解析器
	captures, err := java.NewJavaExtractor().GetCaptures(fCtx)
	if err != nil {
		t.Fatal(err)
	}

	symbolResolver := java.NewSymbolResolver()

	// 4. 定义包含跨包验证在内的完整 Create 预期矩阵
	createExpectations := []CreateExpectation{
		{
			Name:             "场景 1.1: 静态内部类创建",
			LineNum:          18,
			TargetText:       "new Outer.StaticInner()",
			ExpQualifiedName: "com.test.Outer.StaticInner",
			ExpKind:          model.Class,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 1.2: 成员内部类创建",
			LineNum:          22,
			TargetText:       "outer.new MemberInner()",
			ExpQualifiedName: "com.test.Outer.MemberInner",
			ExpKind:          model.Class,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 2.1: 导入外部类创建（带泛型擦除）",
			LineNum:          25,
			TargetText:       "new ArrayList<>()",
			ExpQualifiedName: "java.util.ArrayList",
			ExpKind:          model.Class,
			ExpIsExternal:    true,
		},
		{
			Name:             "场景 2.2: 基础外部类创建",
			LineNum:          28,
			TargetText:       "new String(\"hello\")",
			ExpQualifiedName: "String",
			ExpKind:          model.Class,
			ExpIsExternal:    true,
		},
		{
			Name:             "场景 2.3: 野指针/未导入创建",
			LineNum:          31,
			TargetText:       "new SomeUnknownClass()",
			ExpQualifiedName: "SomeUnknownClass",
			ExpKind:          model.Class,
			ExpIsExternal:    true,
		},
		{
			Name:             "场景 3.1: 实例化后立即发起链式调用 (Create 动作捕获点)",
			LineNum:          34,
			TargetText:       "new Builder()",
			ExpQualifiedName: "com.test.Builder",
			ExpKind:          model.Class,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 4.1: 跨包公开类创建",
			LineNum:          37,
			TargetText:       "new Product()",
			ExpQualifiedName: "com.factory.Product",
			ExpKind:          model.Class,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 5.1: 接口匿名实现类创建",
			LineNum:          40,
			TargetText:       "new Runnable() {\n            @Override public void run() {}\n        }",
			ExpQualifiedName: "Runnable",
			ExpKind:          model.Class,
			ExpIsExternal:    true,
		},
		{
			Name:             "场景 7.1: 数组类型创建 (Array Creation)",
			LineNum:          45,
			TargetText:       "new User[5]",
			ExpQualifiedName: "com.test.User",
			ExpKind:          model.Class,
			ExpIsExternal:    false,
		},
	}

	// 5. 运行断言
	runSymbolCreateAsserts(t, gCtx, fCtx, captures, symbolResolver, createExpectations)
}

// 辅助断言运行器保持通用
func runSymbolCreateAsserts(
	t *testing.T,
	gCtx *core.GlobalContext,
	fCtx *core.FileContext,
	captures []*java.CaptureTarget,
	symbolResolver *java.SymbolResolver,
	expectations []CreateExpectation,
) {
	targetLines := make(map[int]bool)
	for _, exp := range expectations {
		targetLines[exp.LineNum] = true
	}

	resolvedElements := make(map[string]*model.CodeElement)

	for _, cap := range captures {
		lineNum := int(cap.Node.StartPosition().Row) + 1

		if !targetLines[lineNum] {
			continue
		}

		if cap.CapName != "create_target" {
			continue
		}

		element := symbolResolver.ResolveAction(gCtx, fCtx, cap.Node, model.Create)
		if element == nil {
			continue
		}

		exprText := cap.Node.Utf8Text(*fCtx.SourceBytes)
		uniqueKey := fmt.Sprintf("%d:%s", lineNum, normalizeSpaces(exprText))
		resolvedElements[uniqueKey] = element
	}

	for _, exp := range expectations {
		targetKey := fmt.Sprintf("%d:%s", exp.LineNum, normalizeSpaces(exp.TargetText))
		actualElement, found := resolvedElements[targetKey]

		if found {
			fmt.Printf("line: %d -> (%s): %s\n", exp.LineNum, actualElement.Kind, actualElement.QualifiedName)
		} else {
			t.Fatalf("【%s 失败】未能在行号 %d 处捕捉到匹配文本 [%s] 的 Create 动作", exp.Name, exp.LineNum, exp.TargetText)
		}

		if actualElement.QualifiedName != exp.ExpQualifiedName {
			t.Errorf("【%s 失败】QualifiedName 错位.\n 期望: %s\n 实际: %s", exp.Name, exp.ExpQualifiedName, actualElement.QualifiedName)
		}

		if actualElement.Kind != exp.ExpKind {
			t.Errorf("【%s 失败】符号 Kind 不匹配.\n 期望: %s\n 实际: %s", exp.Name, exp.ExpKind, actualElement.Kind)
		}

		if actualElement.IsFormExternal != exp.ExpIsExternal {
			t.Errorf("【%s 失败】IsFormExternal 标志判定错误.\n 期望: %t\n 实际: %t", exp.Name, exp.ExpIsExternal, actualElement.IsFormExternal)
		}
	}
}

func normalizeSpaces(s string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}
