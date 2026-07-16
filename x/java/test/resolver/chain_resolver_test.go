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

type TestExpectation struct {
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
	createExpectations := []TestExpectation{
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
	runSymbolAsserts(t, gCtx, fCtx, captures, symbolResolver, createExpectations, model.Create)
}

func TestSymbolResolver_Cast(t *testing.T) {
	// 1. 获取测试文件路径（包含主测试文件与跨包依赖文件）
	mainTestFile := test.GetTestFilePath(filepath.Join("resolver", "chain_resolve", "cast", "CastCase1.java"))
	depUser := test.GetTestFilePath(filepath.Join("resolver", "chain_resolve", "cast", "dependency", "User.java"))
	depSub := test.GetTestFilePath(filepath.Join("resolver", "chain_resolve", "cast", "dependency", "Sub.java"))
	depDummy := test.GetTestFilePath(filepath.Join("resolver", "chain_resolve", "cast", "dependency", "Dummy.java"))

	// 2. 将主测试文件和依赖文件送入 Phase 1 Collection，建立全局符号索引
	gCtx := test.RunPhase1Collection(t, []string{mainTestFile, depUser, depSub, depDummy})

	// 提取分析上下文
	fCtx, exists := gCtx.FileContexts[mainTestFile]
	if !exists {
		t.Fatalf("无法在 GlobalContext 中找到主测试文件上下文: %s", mainTestFile)
	}

	// 3. 初始化 Extractor 与 SymbolResolver
	captures, err := java.NewJavaExtractor().GetCaptures(fCtx)
	if err != nil {
		t.Fatal(err)
	}

	symbolResolver := java.NewSymbolResolver()

	// 4. 定义包含 4 大维度的 Cast 预期断言矩阵
	castExpectations := []TestExpectation{
		// 维度 1: 基础与泛型强转
		{
			Name:             "场景 1.1: 同包/已导入的本地类强转",
			LineNum:          14,
			TargetText:       "(User) obj",
			ExpQualifiedName: "com.test.dependency.User",
			ExpKind:          model.Class,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 1.2: 带泛型的集合类强转（需擦除泛型）",
			LineNum:          17,
			TargetText:       "(List<String>) obj",
			ExpQualifiedName: "java.util.List",
			ExpKind:          model.Class,
			ExpIsExternal:    true,
		},
		{
			Name:             "场景 1.3: 全限定名强转（不作冗余解析）",
			LineNum:          20,
			TargetText:       "(java.util.Map) obj",
			ExpQualifiedName: "java.util.Map",
			ExpKind:          model.Class,
			ExpIsExternal:    true,
		},

		// 维度 2: 多重强转与嵌套
		{
			Name:             "场景 2.1: 多重强转 (应当捕获最外层的 Runnable)",
			LineNum:          26,
			TargetText:       "(Runnable)(Object) obj",
			ExpQualifiedName: "java.lang.Runnable",
			ExpKind:          model.Class,
			ExpIsExternal:    true,
		},
		{
			Name:             "场景 2.2: 带括号嵌套强转",
			LineNum:          29,
			TargetText:       "((User) obj)",
			ExpQualifiedName: "com.test.dependency.User",
			ExpKind:          model.Class,
			ExpIsExternal:    false,
		},

		// 维度 3: 类型检查 (Instanceof)
		{
			Name:             "场景 3.1: 传统 instanceof 检查",
			LineNum:          35,
			TargetText:       "obj instanceof String",
			ExpQualifiedName: "java.lang.String",
			ExpKind:          model.Class,
			ExpIsExternal:    true,
		},
		{
			Name:             "场景 3.2: 模式匹配 instanceof",
			LineNum:          38,
			TargetText:       "obj instanceof User u",
			ExpQualifiedName: "com.test.dependency.User",
			ExpKind:          model.Class,
			ExpIsExternal:    false,
		},

		// 维度 4: 强转后的链式流转
		{
			Name:             "场景 4.1: 强转后调用方法 (Cast 依赖捕获点)",
			LineNum:          44,
			TargetText:       "((Sub) obj)",
			ExpQualifiedName: "com.test.dependency.Sub",
			ExpKind:          model.Class,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 4.2: 强转后访问属性字段 (Cast 依赖捕获点)",
			LineNum:          47,
			TargetText:       "((Dummy) obj)",
			ExpQualifiedName: "com.test.dependency.Dummy",
			ExpKind:          model.Class,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 4.3: 外部类强转后流转保底 (Cast 依赖捕获点)",
			LineNum:          50,
			TargetText:       "((ArrayList) obj)",
			ExpQualifiedName: "java.util.ArrayList",
			ExpKind:          model.Class,
			ExpIsExternal:    true,
		},
	}

	// 5. 运行断言，针对 model.Cast 动作捕获点进行比对
	runSymbolAsserts(t, gCtx, fCtx, captures, symbolResolver, castExpectations, model.Cast)
}

// 辅助断言运行器保持通用
func runSymbolAsserts(
	t *testing.T,
	gCtx *core.GlobalContext,
	fCtx *core.FileContext,
	captures []*java.CaptureTarget,
	symbolResolver *java.SymbolResolver,
	expectations []TestExpectation,
	relType model.DependencyType,
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

		if captureTypeMap[cap.CapName] != relType {
			continue
		}

		element := symbolResolver.ResolveAction(gCtx, fCtx, cap.Node, relType)
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
			t.Fatalf("【%s 失败】未能在行号 %d 处捕捉到匹配文本 [%s] 的 %s 动作", exp.Name, exp.LineNum, exp.TargetText, relType)
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
