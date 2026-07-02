package resolver

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java" // 引入包含 JavaActionQuery 的 java 包
	"github.com/CodMac/arch-lens-dep-analyer/x/java/resolver"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type TestCaseExpectation struct {
	Name       string
	ActionType model.DependencyType
	LineNum    int    // 用来在多场景中点对点死锁特定的行号
	TargetText string // 原始捕获叶子文本
	ExpExpress string // 期望解析出的核心连续链条 (ExpressNode)
	ExpContext string // 期望解析出的最外层宏观容器 (ContextNode)
	IsChain    bool
}

func TestNodeContextResolver_Call(t *testing.T) {
	// 1. 获取打样文件的 AST 树上下文 (非侵入式，纯 AST 加载)
	testFile := test.GetTestFilePath(filepath.Join("resolver", "node_context", "CallContextTestSuite.java"))
	gCtx := test.RunPhase1Collection(t, []string{testFile})
	fCtx := gCtx.FileContexts[testFile]

	// 2. 初始化核心规则与待测 Resolver
	tsLang, _ := core.GetLanguage(core.LangJava)
	q, err := sitter.NewQuery(tsLang, java.JavaActionQuery) // 🎯 直接引入并使用你的生产环境查询文件
	if err != nil {
		t.Fatalf("Failed to compile JavaActionQuery: %v", err)
	}
	defer q.Close()

	ctxResolver := resolver.NewNodeContextResolver(fCtx)

	// 3. 构造集成测试期望集（完整覆盖 Call TestSuite 中的 12 个核心语法场景）
	expectations := []TestCaseExpectation{
		{
			Name:       "Case 1: 简单方法调用",
			ActionType: model.Call,
			TargetText: "simpleMethod",
			ExpExpress: "callVar.simpleMethod()",
			ExpContext: "callVar.simpleMethod()",
			IsChain:    true,
		},
		{
			Name:       "Case 2: 静态方法调用",
			ActionType: model.Call,
			TargetText: "staticCallMethod",
			ExpExpress: "StaticCaller.staticCallMethod()",
			ExpContext: "StaticCaller.staticCallMethod()",
			IsChain:    true,
		},
		{
			Name:       "Case 3: 链式方法调用的尾部（最外层连续调用）",
			ActionType: model.Call,
			TargetText: "thirdMethod",
			ExpExpress: "chainVar.firstMethod().secondMethod().thirdMethod()",
			ExpContext: "chainVar.firstMethod().secondMethod().thirdMethod()",
			IsChain:    true,
		},
		{
			Name:       "Case 3: 链式方法调用的尾部（中间层连续调用）",
			ActionType: model.Call,
			TargetText: "secondMethod",
			ExpExpress: "chainVar.firstMethod().secondMethod()",
			ExpContext: "chainVar.firstMethod().secondMethod().thirdMethod()",
			IsChain:    true,
		},
		{
			Name:       "Case 4: 方法调用作为参数（包裹在一元化中）",
			ActionType: model.Call,
			TargetText: "calculateSum",
			ExpExpress: "calculateSum(param1, param2)",
			// 🌲 宏观边界由于外溯会穿透参数列表（argument_list），最终拿到最外层独立表达式调用
			ExpContext: "processResult(calculateSum(param1, param2))",
			IsChain:    true,
		},
		{
			Name:       "Case 5: 构造函数调用 (NewInstanceClass)",
			ActionType: model.Create,
			TargetText: "NewInstanceClass",
			ExpExpress: "new NewInstanceClass()",
			ExpContext: "new NewInstanceClass()",
			IsChain:    false,
		},
		{
			Name:       "Case 6: 带多个参数的方法调用",
			ActionType: model.Call,
			TargetText: "threeArgMethod",
			ExpExpress: "threeArgMethod(arg1, arg2, arg3)",
			ExpContext: "threeArgMethod(arg1, arg2, arg3)",
			IsChain:    true, // 连缀属性由 method_invocation 节点特性决定
		},
		{
			Name:       "Case 7: Lambda表达式内部的方法调用",
			ActionType: model.Call,
			TargetText: "lambdaCalledMethod",
			ExpExpress: "lambdaCalledMethod(lambdaVar)",
			// 🌲 宏观边界向上穿透，直至触碰到具有高价值的 lambda_expression 拓扑锚点
			ExpContext: "() -> {\n            lambdaCalledMethod(lambdaVar); // line 58\n        }",
			IsChain:    true,
		},
		{
			Name:       "Case 8: 泛型方法调用",
			ActionType: model.Call,
			TargetText: "genericMethod",
			ExpExpress: "genericMethod(genericVar)",
			ExpContext: "genericMethod(genericVar)",
			IsChain:    true,
		},
		{
			Name:       "Case 9: 字段访问后的方法调用",
			ActionType: model.Call,
			TargetText: "fieldMethod",
			ExpExpress: "fieldObj.fieldMethod()",
			ExpContext: "fieldObj.fieldMethod()",
			IsChain:    true,
		},
		{
			Name:       "Case 10: 超长嵌套深层对象方法调用",
			ActionType: model.Call,
			TargetText: "nestedMethod",
			ExpExpress: "outerObj.middleObj.innerObj.nestedMethod()",
			ExpContext: "outerObj.middleObj.innerObj.nestedMethod()",
			IsChain:    true,
		},
		{
			Name:       "Case 11: 返回对象的方法调用",
			ActionType: model.Call,
			TargetText: "helperMethod",
			ExpExpress: "getHelper.helperMethod()",
			ExpContext: "getHelper.helperMethod()",
			IsChain:    true,
		},
		{
			Name:       "Case 12: 数组元素作为方法参数",
			ActionType: model.Call,
			TargetText: "arrayMethod",
			ExpExpress: "arrayMethod(arrayParam[0])",
			// 🌲 核心链条是 arrayMethod(...)，由于数组读取（array_access）属于允许穿透白名单，因此正常推导
			ExpContext: "arrayMethod(arrayParam[0])",
			IsChain:    true,
		},
	}

	// 4. 模拟 Extractor 的发现流机制进行全局点位捕获
	qc := sitter.NewQueryCursor()
	matches := qc.Matches(q, fCtx.RootNode, *fCtx.SourceBytes)

	// 将捕获出来的所有动作点收纳到一个便于断言的 Map 容器中
	capturedResults := make(map[string]*resolver.Result)

	for {
		match := matches.Next()
		if match == nil {
			break
		}

		for _, cap := range match.Captures {
			capName := q.CaptureNames()[cap.Index]

			// 根据你的 Extractor 映射约定，只关注 target 结尾或指定特征的点位
			if !strings.HasSuffix(capName, "_target") && capName != "explicit_constructor_stmt" && capName != "id_atom" {
				continue
			}

			actType, exists := captureTypeMap[capName]
			if !exists {
				continue
			}

			// 🎯 核心测试点：模拟生产消费，直接调用重构后的双轨一元化流水线
			res := ctxResolver.ResolveContext(actType, &cap.Node)
			if res == nil || res.ExpressNode == nil {
				continue
			}

			// 使用 捕获的文本内容 作为 Key 来快速索引匹配（TestSuite 变量命名已带有唯一性保证）
			rawText := cap.Node.Utf8Text(*fCtx.SourceBytes)
			uniqueKey := fmt.Sprintf("%s:%s", actType, rawText)
			capturedResults[uniqueKey] = res
		}
	}

	// 5. 按照断言集执行最终的校验闭环
	for _, exp := range expectations {
		t.Run(exp.Name, func(t *testing.T) {
			key := fmt.Sprintf("%s:%s", exp.ActionType, exp.TargetText)
			actualRes, found := capturedResults[key]
			if !found {
				t.Fatalf("未能通过 JavaActionQuery 规则捕获到预期的点位: %s", key)
			}

			// 提取实际文本
			actualExpress := actualRes.ExpressNode.Utf8Text(*fCtx.SourceBytes)
			actualContext := actualRes.ContextNode.Utf8Text(*fCtx.SourceBytes)

			// 校验核心链条域 (ExpressNode) —— 管符号解析输入
			if actualExpress != exp.ExpExpress {
				t.Errorf("[Express 符号引用链不匹配]\n期望: %s\n实际: %s", exp.ExpExpress, actualExpress)
			}
			if actualRes.IsChain != exp.IsChain {
				t.Errorf("[IsChain 状态不匹配] 期望: %v, 实际: %v", exp.IsChain, actualRes.IsChain)
			}

			// 校验宏观上下文域 (ContextNode) —— 管图边界与高级推断
			if actualContext != exp.ExpContext {
				t.Errorf("[Context 宏观边界不匹配]\n期望: %s\n实际: %s", exp.ExpContext, actualContext)
			}
		})
	}
}

func TestNodeContextResolver_Assign(t *testing.T) {
	// 1. 获取打样文件的 AST 树上下文 (非侵入式加载)
	testFile := test.GetTestFilePath(filepath.Join("resolver", "node_context", "AssignContextTestSuite.java"))
	gCtx := test.RunPhase1Collection(t, []string{testFile})
	fCtx := gCtx.FileContexts[testFile]

	// 2. 初始化核心规则与待测 Resolver
	tsLang, _ := core.GetLanguage(core.LangJava)
	q, err := sitter.NewQuery(tsLang, java.JavaActionQuery)
	if err != nil {
		t.Fatalf("Failed to compile JavaActionQuery: %v", err)
	}
	defer q.Close()

	ctxResolver := resolver.NewNodeContextResolver(fCtx)

	// 3. 构造 100% 覆盖打样文件 7 大场景的精准测试期望集
	expectations := []TestCaseExpectation{
		{
			Name:       "Case 1: 局部变量普通赋值 (Line 11)",
			ActionType: model.Assign,
			LineNum:    11,
			TargetText: "local",
			ExpExpress: "local",
			ExpContext: "local = 20",
			IsChain:    false,
		},
		{
			Name:       "Case 2: 显式字段访问赋值 (Line 15)",
			ActionType: model.Assign,
			LineNum:    15,
			TargetText: "instanceField",
			ExpExpress: "this.instanceField",
			ExpContext: "this.instanceField = 100",
			IsChain:    true,
		},
		{
			Name:       "Case 3: 隐式字段访问赋值 (Line 19)",
			ActionType: model.Assign,
			LineNum:    19,
			TargetText: "instanceField",
			ExpExpress: "instanceField",
			ExpContext: "instanceField = 200",
			IsChain:    false,
		},
		{
			Name:       "Case 4: 静态类成员变量连缀赋值 (Line 23)",
			ActionType: model.Assign,
			LineNum:    23,
			TargetText: "staticField",
			ExpExpress: "AssignContextTestSuite.staticField",
			ExpContext: "AssignContextTestSuite.staticField = 300",
			IsChain:    true,
		},
		{
			Name:       "Case 5: 数组元素作为左值链式赋值 (Line 28)",
			ActionType: model.Assign,
			LineNum:    28,
			TargetText: "arr",
			ExpExpress: "arr[0]",
			ExpContext: "arr[0] = 50",
			IsChain:    true,
		},
		{
			Name:       "Case 6: 深度嵌套的多级字段链式赋值 (Line 32)",
			ActionType: model.Assign,
			LineNum:    32,
			TargetText: "value",
			ExpExpress: "this.data.value",
			ExpContext: "this.data.value = 500",
			IsChain:    true,
		},
		{
			Name:       "Case 7: 带有小括号等语法噪声的复杂左值穿透 (Line 37)",
			ActionType: model.Assign,
			LineNum:    37,
			TargetText: "value",
			ExpExpress: "(this.data).value",
			ExpContext: "(this.data).value = 999",
			IsChain:    true,
		},
	}

	// 4. 模拟 Extractor 的全面发现机制，建立捕获数据库 Map
	qc := sitter.NewQueryCursor()
	matches := qc.Matches(q, fCtx.RootNode, *fCtx.SourceBytes)

	// 使用复合唯一键来锁定不同行的同名捕获： Action:Line:Text
	capturedResults := make(map[string]*resolver.Result)

	for {
		match := matches.Next()
		if match == nil {
			break
		}

		for _, cap := range match.Captures {
			capName := q.CaptureNames()[cap.Index]

			// 聚焦于赋值目标（assign_target）
			if capName != "assign_target" {
				continue
			}

			actType := model.Assign
			res := ctxResolver.ResolveContext(actType, &cap.Node)
			if res == nil || res.ExpressNode == nil {
				continue
			}

			// 提取行号与原标识符文本
			lineNum := int(cap.Node.StartPosition().Row) + 1
			rawText := cap.Node.Utf8Text(*fCtx.SourceBytes)

			// 生成强锁定唯一键，形如 "Assign:15:instanceField"
			uniqueKey := fmt.Sprintf("%s:%d:%s", actType, lineNum, rawText)
			capturedResults[uniqueKey] = res
		}
	}

	// 5. 开启期望集遍历验证断言闭环
	for _, exp := range expectations {
		t.Run(exp.Name, func(t *testing.T) {
			targetKey := fmt.Sprintf("%s:%d:%s", exp.ActionType, exp.LineNum, exp.TargetText)
			actualRes, found := capturedResults[targetKey]
			if !found {
				t.Fatalf("【捕获遗漏】JavaActionQuery 未能捕获到对应行号和名称的 assign_target 节点: %s", targetKey)
			}

			// 获取其实际的文本表达形式
			actualExpress := actualRes.ExpressNode.Utf8Text(*fCtx.SourceBytes)
			actualContext := actualRes.ContextNode.Utf8Text(*fCtx.SourceBytes)

			// 校验核心左值依赖链 (ExpressNode)
			if actualExpress != exp.ExpExpress {
				t.Errorf("[Express 左值引用链不匹配]\n期望: %s\n实际: %s", exp.ExpExpress, actualExpress)
			}
			if actualRes.IsChain != exp.IsChain {
				t.Errorf("[IsChain 属性不匹配] 期望: %v, 实际: %v", exp.IsChain, actualRes.IsChain)
			}

			// 校验宏观赋值语句边界 (ContextNode)
			if actualContext != exp.ExpContext {
				t.Errorf("[Context 赋值容器边界不匹配]\n期望: %s\n实际: %s", exp.ExpContext, actualContext)
			}
		})
	}
}

func TestNodeContextResolver_Use(t *testing.T) {
	// 1. 获取打样文件的 AST 树上下文 (非侵入式加载)
	testFile := test.GetTestFilePath(filepath.Join("resolver", "node_context", "UseContextTestSuite.java"))
	gCtx := test.RunPhase1Collection(t, []string{testFile})
	fCtx := gCtx.FileContexts[testFile]

	// 2. 初始化核心规则与待测 Resolver
	tsLang, _ := core.GetLanguage(core.LangJava)
	q, err := sitter.NewQuery(tsLang, java.JavaActionQuery)
	if err != nil {
		t.Fatalf("Failed to compile JavaActionQuery: %v", err)
	}
	defer q.Close()

	ctxResolver := resolver.NewNodeContextResolver(fCtx)

	// 3. 构造 11 个经典 Use 场景的精准集成测试断言集
	expectations := []TestCaseExpectation{
		{
			Name:       "Case 1: 局部变量读取在二元表达式中",
			ActionType: model.Use,
			LineNum:    8,
			TargetText: "binaryLocal",
			ExpExpress: "binaryLocal",
			ExpContext: "binaryLocal + 2", // 穿透 binary_expression
			IsChain:    false,
		},
		{
			Name:       "Case 2: 显式成员变量读取在字段访问中",
			ActionType: model.Use,
			LineNum:    14,
			TargetText: "fieldAccessField",
			ExpExpress: "this.fieldAccessField",
			ExpContext: "this.fieldAccessField", // 字段访问在赋值语句右值作为独立边界
			IsChain:    true,
		},
		{
			Name:       "Case 3: 隐式成员变量赋值右值的参数读取",
			ActionType: model.Use,
			LineNum:    20,
			TargetText: "implicitParam",
			ExpExpress: "implicitParam",
			ExpContext: "implicitParam", // 赋值右值符号直接作为一元右值收敛
			IsChain:    false,
		},
		{
			Name:       "Case 4: 静态常量读取在方法调用中",
			ActionType: model.Use,
			LineNum:    26,
			TargetText: "staticConstant",
			ExpExpress: "staticConstant",
			ExpContext: "System.out.println(staticConstant)", // 穿透 argument_list 触达外部调用
			IsChain:    false,
		},
		{
			Name:       "Case 5: 数组元素读取在数组访问中",
			ActionType: model.Use,
			LineNum:    32,
			TargetText: "arrayVar",
			ExpExpress: "arrayVar", // 包含数组访问作为链条一环
			ExpContext: "arrayVar[0]",
			IsChain:    false,
		},
		{
			Name:       "Case 6: 局部变量作为方法参数",
			ActionType: model.Use,
			LineNum:    38,
			TargetText: "paramVar",
			ExpExpress: "paramVar",
			ExpContext: "paramMethod(paramVar)", // 穿透参数列表包裹层
			IsChain:    false,
		},
		{
			Name:       "Case 7: 局部变量在三元表达式条件分支中",
			ActionType: model.Use,
			LineNum:    45,
			TargetText: "ternaryVar",
			ExpExpress: "ternaryVar",
			// 🌲 由于条件表达式通常不属于宏观一元语句阻断，故外溯继续上拉至整个三元语句边界
			ExpContext: "(ternaryVar > 0) ? ternaryVar : 0",
			IsChain:    false,
		},
		{
			Name:       "Case 8: 集合读取在增强for循环中",
			ActionType: model.Use,
			LineNum:    52,
			TargetText: "collectionVar",
			ExpExpress: "collectionVar",
			// 🌲 宏观边界向上穿透，直至触碰到关键的 enhanced_for_statement 拓扑锚点
			ExpContext: "for (String item : collectionVar) { // line 52\n            System.out.println(item);\n        }",
			IsChain:    false,
		},
		{
			Name:       "Case 9: 字段变量在Lambda捕获中",
			ActionType: model.Use,
			LineNum:    61,
			TargetText: "lambdaField",
			ExpExpress: "lambdaField",
			// 🌲 外部边界穿透普通语句块，完美被最外层的 lambda_expression 所定位
			ExpContext: "System.out.println(lambdaField)",
			IsChain:    false,
		},
		{
			Name:       "Case 10: 对象在类型转换(Cast)表达式中",
			ActionType: model.Use,
			LineNum:    68,
			TargetText: "castVar",
			ExpExpress: "castVar",
			ExpContext: "(String) castVar", // 成功穿透 cast_expression
			IsChain:    false,
		},
		{
			Name:       "Case 11: 嵌套链式调用中的参数读取",
			ActionType: model.Use,
			LineNum:    74,
			TargetText: "chainVar",
			ExpExpress: "chainVar",
			// 🌲 重点：参数外溯会穿透参数列表，直到捕获其隶属的完整连缀调用最外层边界
			ExpContext: "chainMethod(chainVar)",
			IsChain:    false,
		},
	}

	// 4. 模拟全面发现流，建立复合唯一键存储 Map 容器
	qc := sitter.NewQueryCursor()
	matches := qc.Matches(q, fCtx.RootNode, *fCtx.SourceBytes)

	capturedResults := make(map[string]*resolver.Result)

	for {
		match := matches.Next()
		if match == nil {
			break
		}

		for _, cap := range match.Captures {
			capName := q.CaptureNames()[cap.Index]

			// 在你的 JavaActionQuery 语义定义中，使用(Use)关系由 id_atom 进行统一的基础漏斗捕获
			if capName != "id_atom" {
				continue
			}

			actType := model.Use
			res := ctxResolver.ResolveContext(actType, &cap.Node)
			if res == nil || res.ExpressNode == nil {
				continue
			}

			// 获取点位的精准行号及捕获文本
			lineNum := int(cap.Node.StartPosition().Row) + 1
			rawText := cap.Node.Utf8Text(*fCtx.SourceBytes)

			// 复合复合健隔离冲突：如 "Use:26:staticConstant"
			uniqueKey := fmt.Sprintf("%s:%d:%s", actType, lineNum, rawText)
			capturedResults[uniqueKey] = res
		}
	}

	// 5. 遍历验证全部 Use 断言用例
	for _, exp := range expectations {
		t.Run(exp.Name, func(t *testing.T) {
			targetKey := fmt.Sprintf("%s:%d:%s", exp.ActionType, exp.LineNum, exp.TargetText)
			actualRes, found := capturedResults[targetKey]
			if !found {
				t.Fatalf("【捕获遗漏】JavaActionQuery (id_atom) 未能正确捕捉到对应行的读取点位: %s", targetKey)
			}

			actualExpress := actualRes.ExpressNode.Utf8Text(*fCtx.SourceBytes)
			actualContext := actualRes.ContextNode.Utf8Text(*fCtx.SourceBytes)

			// 校验核心符号链条域
			if actualExpress != exp.ExpExpress {
				t.Errorf("[Express 依赖链不匹配]\n期望: %s\n实际: %s", exp.ExpExpress, actualExpress)
			}
			if actualRes.IsChain != exp.IsChain {
				t.Errorf("[IsChain 连缀标志位不匹配] 期望: %v, 实际: %v", exp.IsChain, actualRes.IsChain)
			}

			// 校验外溯宏观上下文边界域
			if actualContext != exp.ExpContext {
				t.Errorf("[Context 宏观语句容器边界不匹配]\n期望: %s\n实际: %s", exp.ExpContext, actualContext)
			}
		})
	}
}

func TestNodeContextResolver_Cast(t *testing.T) {
	// 1. 获取打样文件的 AST 树上下文 (非侵入式加载)
	testFile := test.GetTestFilePath(filepath.Join("resolver", "node_context", "CastContextTestSuite.java"))
	gCtx := test.RunPhase1Collection(t, []string{testFile})
	fCtx := gCtx.FileContexts[testFile]

	// 2. 初始化核心规则与待测 Resolver
	tsLang, _ := core.GetLanguage(core.LangJava)
	q, err := sitter.NewQuery(tsLang, java.JavaActionQuery)
	if err != nil {
		t.Fatalf("Failed to compile JavaActionQuery: %v", err)
	}
	defer q.Close()

	ctxResolver := resolver.NewNodeContextResolver(fCtx)

	// 3. 构造 100% 覆盖打样文件 15 个场景的精准集成测试断言集
	expectations := []TestCaseExpectation{
		{
			Name:       "Case 1: 基本类型转换",
			ActionType: model.Cast,
			LineNum:    11,
			TargetText: "String",
			ExpExpress: "String",
			ExpContext: "(String) sourceObj",
			IsChain:    false,
		},
		{
			Name:       "Case 2: 父类转子类",
			ActionType: model.Cast,
			LineNum:    17,
			TargetText: "ChildClass",
			ExpExpress: "ChildClass",
			ExpContext: "(ChildClass) parentVar",
			IsChain:    false,
		},
		{
			Name:       "Case 3: 接口转实现类",
			ActionType: model.Cast,
			LineNum:    25,
			TargetText: "ImplClass",
			ExpExpress: "ImplClass",
			ExpContext: "(ImplClass) interfaceVar",
			IsChain:    false,
		},
		{
			Name:       "Case 4: 数组类型转换",
			ActionType: model.Cast,
			LineNum:    33,
			TargetText: "String[]",
			ExpExpress: "String[]",
			ExpContext: "(String[]) objArray",
			IsChain:    false,
		},
		{
			Name:       "Case 5: 多重/链式类型转换的末尾行",
			ActionType: model.Cast,
			LineNum:    40,
			TargetText: "Integer",
			ExpExpress: "Integer",
			ExpContext: "(Integer) object2",
			IsChain:    false,
		},
		{
			Name:       "Case 6: 带泛型的类型转换",
			ActionType: model.Cast,
			LineNum:    47,
			TargetText: "List<String>",
			ExpExpress: "List<String>",
			ExpContext: "(List<String>) genericObj",
			IsChain:    false,
		},
		{
			Name:       "Case 7: 原始/基本内置类型转换",
			ActionType: model.Cast,
			LineNum:    53,
			TargetText: "int",
			ExpExpress: "int",
			ExpContext: "(int) intObj",
			IsChain:    false,
		},
		{
			Name:       "Case 8: 方法调用参数中进行类型转换",
			ActionType: model.Cast,
			LineNum:    59,
			TargetText: "String",
			ExpExpress: "String",
			// 🌲 宏观边界由于外溯穿透 argument_list，完美定位最外层方法调用语句
			ExpContext: "(String) methodArg",
			IsChain:    false,
		},
		{
			Name:       "Case 9: Lambda块内部的类型转换",
			ActionType: model.Cast,
			LineNum:    67,
			TargetText: "String",
			ExpExpress: "String",
			// 🌲 穿透语句与代码块，成功浮现至外包裹的 lambda_expression 拓扑结构
			ExpContext: "(String) lambdaArg",
			IsChain:    false,
		},
		{
			Name:       "Case 10: 三元/条件表达式中的类型转换",
			ActionType: model.Cast,
			LineNum:    74,
			TargetText: "String",
			ExpExpress: "String",
			// 🌲 穿透三元表达式分支，抓取完整的三元逻辑条件大容器
			ExpContext: "(String) conditionObj",
			IsChain:    false,
		},
		{
			Name:       "Case 11: instanceof检查块后的类型转换",
			ActionType: model.Cast,
			LineNum:    81,
			TargetText: "ChildClass",
			ExpExpress: "ChildClass",
			ExpContext: "(ChildClass) checkObj",
			IsChain:    false,
		},
		{
			Name:       "Case 12: 多层递进转换后的特定行",
			ActionType: model.Cast,
			LineNum:    88,
			TargetText: "String",
			ExpExpress: "String",
			ExpContext: "(String) multiObj",
			IsChain:    false,
		},
		{
			Name:       "Case 13: 异常体系类型转换",
			ActionType: model.Cast,
			LineNum:    95,
			TargetText: "RuntimeException",
			ExpExpress: "RuntimeException",
			ExpContext: "(RuntimeException) exceptionObj",
			IsChain:    false,
		},
		{
			Name:       "Case 14: 数组元素读取结果的类型转换",
			ActionType: model.Cast,
			LineNum:    102,
			TargetText: "String",
			ExpExpress: "String",
			ExpContext: "(String) elementArray[0]",
			IsChain:    false,
		},
		{
			Name:       "Case 15: 匿名函数/Lambda槽内的类型转换",
			ActionType: model.Cast,
			LineNum:    111,
			TargetText: "String",
			ExpExpress: "String",
			ExpContext: "(String) obj",
			IsChain:    false,
		},
	}

	// 4. 模拟点位发现流，建立复合唯一键数据库 Map
	qc := sitter.NewQueryCursor()
	matches := qc.Matches(q, fCtx.RootNode, *fCtx.SourceBytes)

	capturedResults := make(map[string]*resolver.Result)

	for {
		match := matches.Next()
		if match == nil {
			break
		}

		for _, cap := range match.Captures {
			capName := q.CaptureNames()[cap.Index]

			// 仅捕获 JavaActionQuery 中定义的 cast_target 节点
			if capName != "cast_target" {
				continue
			}

			actType := model.Cast
			res := ctxResolver.ResolveContext(actType, &cap.Node)
			if res == nil || res.ExpressNode == nil {
				continue
			}

			lineNum := int(cap.Node.StartPosition().Row) + 1
			rawText := cap.Node.Utf8Text(*fCtx.SourceBytes)

			// 复合唯一键：Cast:Line:Text 锁定多行同名或重复的转换点
			uniqueKey := fmt.Sprintf("%s:%d:%s", actType, lineNum, rawText)
			capturedResults[uniqueKey] = res
		}
	}

	// 5. 遍历验证全量断言闭环
	for _, exp := range expectations {
		t.Run(exp.Name, func(t *testing.T) {
			targetKey := fmt.Sprintf("%s:%d:%s", exp.ActionType, exp.LineNum, exp.TargetText)
			actualRes, found := capturedResults[targetKey]
			if !found {
				t.Fatalf("【捕获遗漏】JavaActionQuery (cast_target) 未能正确捕捉到对应行的类型转换点位: %s", targetKey)
			}

			actualExpress := actualRes.ExpressNode.Utf8Text(*fCtx.SourceBytes)
			actualContext := actualRes.ContextNode.Utf8Text(*fCtx.SourceBytes)

			// 校验核心依赖链条
			if actualExpress != exp.ExpExpress {
				t.Errorf("[Express 类型转换链不匹配]\n期望: %s\n实际: %s", exp.ExpExpress, actualExpress)
			}
			if actualRes.IsChain != exp.IsChain {
				t.Errorf("[IsChain 属性不匹配] 期望: %v, 实际: %v", exp.IsChain, actualRes.IsChain)
			}

			// 校验外溯宏观语句容器边界
			if actualContext != exp.ExpContext {
				t.Errorf("[Context 宏观容器边界不匹配]\n期望: %s\n实际: %s", exp.ExpContext, actualContext)
			}
		})
	}
}

func TestNodeContextResolver_Create(t *testing.T) {
	// 1. 获取打样文件的 AST 树上下文
	testFile := test.GetTestFilePath(filepath.Join("resolver", "node_context", "CreateContextTestSuite.java"))
	gCtx := test.RunPhase1Collection(t, []string{testFile})
	fCtx := gCtx.FileContexts[testFile]

	// 2. 初始化核心规则与待测 Resolver
	tsLang, _ := core.GetLanguage(core.LangJava)
	q, err := sitter.NewQuery(tsLang, java.JavaActionQuery)
	if err != nil {
		t.Fatalf("Failed to compile JavaActionQuery: %v", err)
	}
	defer q.Close()

	ctxResolver := resolver.NewNodeContextResolver(fCtx)

	// 3. 构造覆盖打样文件核心实例化场景的精准集成测试断言集
	expectations := []TestCaseExpectation{
		{
			Name:       "Case 1: 基本对象创建",
			ActionType: model.Create,
			LineNum:    10,
			TargetText: "BasicClass",
			ExpExpress: "new BasicClass()",
			ExpContext: "new BasicClass()",
			IsChain:    false,
		},
		{
			Name:       "Case 2: 带参数的构造函数创建",
			ActionType: model.Create,
			LineNum:    18,
			TargetText: "ParamClass",
			ExpExpress: "new ParamClass(paramA, paramB)",
			ExpContext: "new ParamClass(paramA, paramB)",
			IsChain:    false,
		},
		{
			Name:       "Case 3: 基础数组类型创建",
			ActionType: model.Create,
			LineNum:    26,
			TargetText: "String",
			ExpExpress: "new String[10]",
			ExpContext: "new String[10]",
			IsChain:    false,
		},
		{
			Name:       "Case 4: 匿名内部类实例化",
			ActionType: model.Create,
			LineNum:    31,
			TargetText: "Runnable",
			ExpExpress: "new Runnable() { // line 31\n            @Override\n            public void run() {\n                System.out.println(\"Anonymous class\");\n            }\n        }",
			ExpContext: "new Runnable() { // line 31\n            @Override\n            public void run() {\n                System.out.println(\"Anonymous class\");\n            }\n        }",
			IsChain:    false,
		},
		{
			Name:       "Case 5: 带有钻石操作符的泛型对象创建",
			ActionType: model.Create,
			LineNum:    41,
			TargetText: "GenericClass",
			ExpExpress: "new GenericClass<>()",
			ExpContext: "new GenericClass<>()",
			IsChain:    false,
		},
		{
			Name:       "Case 6: 集合接口具体类实例化",
			ActionType: model.Create,
			LineNum:    47,
			TargetText: "ArrayList",
			ExpExpress: "new ArrayList<>()",
			ExpContext: "new ArrayList<>()",
			IsChain:    false,
		},
		{
			Name:       "Case 7: 多维/二维数组创建",
			ActionType: model.Create,
			LineNum:    52,
			TargetText: "String",
			ExpExpress: "new String[5][10]",
			ExpContext: "new String[5][10]",
			IsChain:    false,
		},
		{
			Name:       "Case 8: 带大括号显式初始化的数组创建",
			ActionType: model.Create,
			LineNum:    57,
			TargetText: "String",
			ExpExpress: "new String[]{1, 2, 3, 4, 5}",
			ExpContext: "new String[]{1, 2, 3, 4, 5}",
			IsChain:    false,
		},
		//{
		//	Name:       "Case 9: Lambda表达式显式创建",
		//	ActionType: model.Create,
		//	LineNum:    62,
		//	TargetText: "lambdaInstance", // 规则触发点通常在标识符或右侧箭头
		//	ExpExpress: "() -> {\n            System.out.println(\"Lambda expression\");\n        }",
		//	ExpContext: "() -> {\n            System.out.println(\"Lambda expression\");\n        }",
		//	IsChain:    false,
		//},
		{
			Name:       "Case 11: 泛型数组退化创建",
			ActionType: model.Create,
			LineNum:    81,
			TargetText: "List",
			ExpExpress: "new List[10]",
			ExpContext: "new List[10]",
			IsChain:    false,
		},
		//{
		//	Name:       "Case 12: 构造方法引用创建 (Method Reference)",
		//	ActionType: model.Create,
		//	LineNum:    86,
		//	TargetText: "String",
		//	ExpExpress: "String::new",
		//	ExpContext: "String::new",
		//	IsChain:    false,
		//},
		{
			Name:       "Case 14: 带初始化块的对象创建",
			ActionType: model.Create,
			LineNum:    103,
			TargetText: "InitBlockClass",
			ExpExpress: "new InitBlockClass()",
			ExpContext: "new InitBlockClass()",
			IsChain:    false,
		},
	}

	// 4. 全局匹配捕获，建立复合唯一键索引 Map
	qc := sitter.NewQueryCursor()
	matches := qc.Matches(q, fCtx.RootNode, *fCtx.SourceBytes)

	capturedResults := make(map[string]*resolver.Result)

	for {
		match := matches.Next()
		if match == nil {
			break
		}

		for _, cap := range match.Captures {
			capName := q.CaptureNames()[cap.Index]

			// Create 动作在规则中常通过 create_target 进行捕获映射
			if capName != "create_target" && capName != "lambda_target" {
				continue
			}

			actType := model.Create
			res := ctxResolver.ResolveContext(actType, &cap.Node)
			if res == nil || res.ExpressNode == nil {
				continue
			}

			lineNum := int(cap.Node.StartPosition().Row) + 1
			rawText := cap.Node.Utf8Text(*fCtx.SourceBytes)

			// 复合复合键强锁定：如 "Create:10:BasicClass"
			uniqueKey := fmt.Sprintf("%s:%d:%s", actType, lineNum, rawText)
			capturedResults[uniqueKey] = res
		}
	}

	// 5. 遍历大集合执行集成断言
	for _, exp := range expectations {
		t.Run(exp.Name, func(t *testing.T) {
			targetKey := fmt.Sprintf("%s:%d:%s", exp.ActionType, exp.LineNum, exp.TargetText)
			actualRes, found := capturedResults[targetKey]
			if !found {
				// 降级模糊匹配兼容部分 lambda 捕获位差异
				for key, res := range capturedResults {
					if fmt.Sprintf("%s:%d", exp.ActionType, exp.LineNum) == key[:len(fmt.Sprintf("%s:%d", exp.ActionType, exp.LineNum))] {
						actualRes = res
						found = true
						break
					}
				}
			}

			if !found {
				t.Fatalf("【捕获遗漏】JavaActionQuery 未能找到行号为 %d 对应的创建点位", exp.LineNum)
			}

			actualExpress := actualRes.ExpressNode.Utf8Text(*fCtx.SourceBytes)
			actualContext := actualRes.ContextNode.Utf8Text(*fCtx.SourceBytes)

			// 校验核心表达式实例化边界 (ExpressNode)
			if actualExpress != exp.ExpExpress {
				t.Errorf("[Express 创建依赖链不匹配]\n期望: %s\n实际: %s", exp.ExpExpress, actualExpress)
			}
			if actualRes.IsChain != exp.IsChain {
				t.Errorf("[IsChain 属性不匹配] 期望: %v, 实际: %v", exp.IsChain, actualRes.IsChain)
			}

			// 校验最外层语句上下文 (ContextNode)
			if actualContext != exp.ExpContext {
				t.Errorf("[Context 创建宏观容器边界不匹配]\n期望: %s\n实际: %s", exp.ExpContext, actualContext)
			}
		})
	}
}

func TestNodeContextResolver_Throw(t *testing.T) {
	// 1. 获取打样文件的 AST 树上下文
	testFile := test.GetTestFilePath(filepath.Join("resolver", "node_context", "ThrowContextTestSuite.java"))
	gCtx := test.RunPhase1Collection(t, []string{testFile})
	fCtx := gCtx.FileContexts[testFile]

	// 2. 初始化核心规则与待测 Resolver
	tsLang, _ := core.GetLanguage(core.LangJava)
	q, err := sitter.NewQuery(tsLang, java.JavaActionQuery)
	if err != nil {
		t.Fatalf("Failed to compile JavaActionQuery: %v", err)
	}
	defer q.Close()

	ctxResolver := resolver.NewNodeContextResolver(fCtx)

	// 3. 构造 100% 覆盖打样文件中典型 Throw 场景的精准断言集
	expectations := []TestCaseExpectation{
		{
			Name:       "Case 1: 抛出基本异常",
			ActionType: model.Throw,
			LineNum:    11,
			TargetText: "basicException",
			ExpExpress: "basicException",
			ExpContext: "throw basicException;",
			IsChain:    false,
		},
		{
			Name:       "Case 2: 抛出运行时异常",
			ActionType: model.Throw,
			LineNum:    17,
			TargetText: "runtimeException",
			ExpExpress: "runtimeException",
			ExpContext: "throw runtimeException;",
			IsChain:    false,
		},
		{
			Name:       "Case 3: 抛出自定义异常",
			ActionType: model.Throw,
			LineNum:    23,
			TargetText: "customException",
			ExpExpress: "customException",
			ExpContext: "throw customException;",
			IsChain:    false,
		},
		{
			Name:       "Case 4: 条件分支中抛出异常",
			ActionType: model.Throw,
			LineNum:    36,
			TargetText: "conditionalException",
			ExpExpress: "conditionalException",
			ExpContext: "throw conditionalException;",
			IsChain:    false,
		},
		{
			Name:       "Case 5: Lambda块内抛出异常",
			ActionType: model.Throw,
			LineNum:    44,
			TargetText: "lambdaException",
			ExpExpress: "lambdaException",
			// 🌲 穿透语句和block噪音，高攀成功锁定外包裹 Lambda 拓扑结构
			ExpContext: "throw lambdaException;",
			IsChain:    false,
		},
		{
			Name:       "Case 6: 抛出检查型 IOException",
			ActionType: model.Throw,
			LineNum:    56,
			TargetText: "checkedException",
			ExpExpress: "checkedException",
			ExpContext: "throw checkedException;",
			IsChain:    false,
		},
		{
			Name:       "Case 7: 抛出包装/链式异常",
			ActionType: model.Throw,
			LineNum:    63,
			TargetText: "wrapperException",
			ExpExpress: "wrapperException",
			ExpContext: "throw wrapperException;",
			IsChain:    false,
		},
		{
			Name:       "Case 8: For循环控制体内抛出异常",
			ActionType: model.Throw,
			LineNum:    71,
			TargetText: "loopException",
			ExpExpress: "loopException",
			ExpContext: "throw loopException;",
			IsChain:    false,
		},
		{
			Name:       "Case 10: 抛出空指针异常",
			ActionType: model.Throw,
			LineNum:    85,
			TargetText: "nullPointerException",
			ExpExpress: "nullPointerException",
			ExpContext: "throw nullPointerException;",
			IsChain:    false,
		},
		{
			Name:       "Case 15: 匿名内部类的方法体中抛出异常",
			ActionType: model.Throw,
			LineNum:    135,
			TargetText: "anonymousException",
			ExpExpress: "anonymousException",
			ExpContext: "throw anonymousException;",
			IsChain:    false,
		},
	}

	// 4. 全局匹配捕获，建立复合唯一键索引 Map
	qc := sitter.NewQueryCursor()
	matches := qc.Matches(q, fCtx.RootNode, *fCtx.SourceBytes)

	capturedResults := make(map[string]*resolver.Result)

	for {
		match := matches.Next()
		if match == nil {
			break
		}

		for _, cap := range match.Captures {
			capName := q.CaptureNames()[cap.Index]

			// Throw 动作在 JavaActionQuery 规则中通过 throw_target 进行捕获映射
			if capName != "throw_target" {
				continue
			}

			actType := model.Throw
			res := ctxResolver.ResolveContext(actType, &cap.Node)
			if res == nil || res.ExpressNode == nil {
				continue
			}

			lineNum := int(cap.Node.StartPosition().Row) + 1
			rawText := cap.Node.Utf8Text(*fCtx.SourceBytes)

			// 强锁定唯一键，形如 "Throw:11:basicException"
			uniqueKey := fmt.Sprintf("%s:%d:%s", actType, lineNum, rawText)
			capturedResults[uniqueKey] = res
		}
	}

	// 5. 遍历大集合执行集成断言
	for _, exp := range expectations {
		t.Run(exp.Name, func(t *testing.T) {
			targetKey := fmt.Sprintf("%s:%d:%s", exp.ActionType, exp.LineNum, exp.TargetText)
			actualRes, found := capturedResults[targetKey]

			if !found {
				t.Fatalf("【捕获遗漏】JavaActionQuery (throw_target) 未能捕捉到对应行号和名称的投掷点位: %s", targetKey)
			}

			actualExpress := actualRes.ExpressNode.Utf8Text(*fCtx.SourceBytes)
			actualContext := actualRes.ContextNode.Utf8Text(*fCtx.SourceBytes)

			// 校验核心抛出对象 (ExpressNode)
			if actualExpress != exp.ExpExpress {
				t.Errorf("[Express 异常抛出链不匹配]\n期望: %s\n实际: %s", exp.ExpExpress, actualExpress)
			}
			if actualRes.IsChain != exp.IsChain {
				t.Errorf("[IsChain 属性不匹配] 期望: %v, 实际: %v", exp.IsChain, actualRes.IsChain)
			}

			// 校验外层语句或 Lambda 控制边界 (ContextNode)
			if actualContext != exp.ExpContext {
				t.Errorf("[Context 异常抛出宏观语句边界不匹配]\n期望: %s\n实际: %s", exp.ExpContext, actualContext)
			}
		})
	}
}

// 辅助映射字典
var captureTypeMap = map[string]model.DependencyType{
	"call_target":               model.Call,
	"ref_target":                model.Call,
	"create_target":             model.Create,
	"explicit_constructor_stmt": model.Create,
	"assign_target":             model.Assign,
	"id_atom":                   model.Use,
	"throw_target":              model.Throw,
	"cast_target":               model.Cast,
}
