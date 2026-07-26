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
	mainTestFile := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "create", "CreateCase1.java"))
	dependencyFile1 := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "create", "case1", "Outer.java"))
	dependencyFile2 := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "create", "case1", "Builder.java"))
	dependencyFile3 := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "create", "case1", "Product.java"))
	dependencyFile4 := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "create", "case1", "User.java"))

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
		// 该语法当前暂不支持
		//{
		//	Name:             "场景 1.2: 成员内部类创建",
		//	LineNum:          22,
		//	TargetText:       "outer.new MemberInner()",
		//	ExpQualifiedName: "com.test.Outer.MemberInner",
		//	ExpKind:          model.Class,
		//	ExpIsExternal:    false,
		//},
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
	mainTestFile := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "cast", "CastCase1.java"))
	depUser := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "cast", "case1", "User.java"))
	depSub := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "cast", "case1", "Sub.java"))
	depDummy := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "cast", "case1", "Dummy.java"))

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
			ExpQualifiedName: "com.test.case1.User",
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
			ExpQualifiedName: "com.test.case1.User",
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
			ExpQualifiedName: "com.test.case1.User",
			ExpKind:          model.Class,
			ExpIsExternal:    false,
		},

		// 维度 4: 强转后的链式流转
		{
			Name:             "场景 4.1: 强转后调用方法 (Cast 依赖捕获点)",
			LineNum:          44,
			TargetText:       "((Sub) obj)",
			ExpQualifiedName: "com.test.case1.Sub",
			ExpKind:          model.Class,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 4.2: 强转后访问属性字段 (Cast 依赖捕获点)",
			LineNum:          47,
			TargetText:       "((Dummy) obj)",
			ExpQualifiedName: "com.test.case1.Dummy",
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

func TestSymbolResolver_Throw(t *testing.T) {
	// 1. 获取测试文件路径
	mainTestFile := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "throw", "ThrowCase1.java"))
	depCustomExc := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "throw", "case1", "CustomException.java"))
	depExcFactory := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "throw", "case1", "ExceptionFactory.java"))

	// 2. 将主测试文件和跨包依赖文件送入 Phase 1 Collection
	gCtx := test.RunPhase1Collection(t, []string{mainTestFile, depCustomExc, depExcFactory})

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

	// 4. 定义涵盖所有语法维度的 Throw 预期断言矩阵
	throwExpectations := []TestExpectation{
		// 维度 1: 基础实例化抛出
		{
			Name:             "场景 1.1: 同包/已导入的自定义异常抛出",
			LineNum:          15,
			TargetText:       "throw new CustomException(\"object is null\")",
			ExpQualifiedName: "com.test.case1.CustomException",
			ExpKind:          model.Class,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 1.2: 标准库异常抛出",
			LineNum:          20,
			TargetText:       "throw new IllegalArgumentException(\"invalid flag\")",
			ExpQualifiedName: "java.lang.IllegalArgumentException",
			ExpKind:          model.Class,
			ExpIsExternal:    true,
		},
		{
			Name:             "场景 1.3: 未导入/未知异常保底抛出",
			LineNum:          25,
			TargetText:       "throw new UnknownException()",
			ExpQualifiedName: "UnknownException",
			ExpKind:          model.Class,
			ExpIsExternal:    true,
		},

		// 维度 2: 变量与参数直接抛出
		{
			Name:             "场景 2.1: Catch 块形参直接重新抛出",
			LineNum:          35,
			TargetText:       "throw e",
			ExpQualifiedName: "java.io.IOException",
			ExpKind:          model.Class,
			ExpIsExternal:    true,
		},
		{
			Name:             "场景 2.2: 局部变量抛出",
			LineNum:          40,
			TargetText:       "throw ex",
			ExpQualifiedName: "java.lang.RuntimeException",
			ExpKind:          model.Class,
			ExpIsExternal:    true,
		},

		// 维度 3: 工厂方法与复杂表达式抛出
		{
			Name:             "场景 3.1: 静态工厂方法创建并抛出（按返回值类型推导）",
			LineNum:          48,
			TargetText:       "throw ExceptionFactory.createException()",
			ExpQualifiedName: "com.test.case1.CustomException",
			ExpKind:          model.Class,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 3.2: 带 Cast 强转的异常抛出",
			LineNum:          53,
			TargetText:       "throw (RuntimeException) obj",
			ExpQualifiedName: "java.lang.RuntimeException",
			ExpKind:          model.Class,
			ExpIsExternal:    true,
		},

		// 维度 4: 三元运算符抛出
		{
			Name:             "场景 4.1: 三元表达式分支抛出",
			LineNum:          60,
			TargetText:       "throw flag ? new CustomException() : new IllegalArgumentException()",
			ExpQualifiedName: "com.test.case1.CustomException",
			ExpKind:          model.Class,
			ExpIsExternal:    false,
		},

		// 维度 5: Lambda 与 Optional 抛出
		{
			Name:             "场景 5.1: Lambda 作用域内部抛出",
			LineNum:          68,
			TargetText:       "throw new IllegalStateException(\"lambda error\")",
			ExpQualifiedName: "java.lang.IllegalStateException",
			ExpKind:          model.Class,
			ExpIsExternal:    true,
		},
		{
			Name:             "场景 5.2: Optional orElseThrow 方法引用抛出",
			LineNum:          73,
			TargetText:       "CustomException::new",
			ExpQualifiedName: "com.test.case1.CustomException",
			ExpKind:          model.Class,
			ExpIsExternal:    false,
		},
	}

	// 5. 运行断言，针对 model.Throw 动作捕获点进行比对
	runSymbolAsserts(t, gCtx, fCtx, captures, symbolResolver, throwExpectations, model.Throw)
}

func TestSymbolResolver_Call(t *testing.T) {
	// 1. 获取测试文件路径
	mainTestFile := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "call", "CallCase1.java"))
	depParent := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "call", "case1", "ParentService.java"))
	depUserService := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "call", "case1", "UserService.java"))
	depUser := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "call", "case1", "User.java"))
	depOrder := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "call", "case1", "Order.java"))

	// 2. 一同送入 Phase 1 Collection 建立符号全局索引
	gCtx := test.RunPhase1Collection(t, []string{
		mainTestFile, depParent, depUserService, depUser, depOrder,
	})

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

	// 4. 定义包含 5 大维度的 Call 预期断言矩阵
	callExpectations := []TestExpectation{
		// 维度 1: 基础与跨类调用
		{
			Name:             "场景 1.1: 同类隐式方法调用",
			LineNum:          18,
			TargetText:       "doInternal()",
			ExpQualifiedName: "com.test.CallCase1.doInternal()",
			ExpKind:          model.Method,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 1.2: 实例属性方法调用",
			LineNum:          21,
			TargetText:       "userService.saveUser(currentUser)",
			ExpQualifiedName: "com.test.case1.UserService.saveUser(User)",
			ExpKind:          model.Method,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 1.3: 静态方法调用",
			LineNum:          24,
			TargetText:       "StringUtils.isEmpty(\"test\")",
			ExpQualifiedName: "com.test.CallCase1$StringUtils.isEmpty(String)",
			ExpKind:          model.Method,
			ExpIsExternal:    false,
		},

		// 维度 2: 继承链与多态调用
		{
			Name:             "场景 2.1: this 关键字调用",
			LineNum:          30,
			TargetText:       "this.doInternal()",
			ExpQualifiedName: "com.test.CallCase1.doInternal()",
			ExpKind:          model.Method,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 2.2: super 父类方法调用",
			LineNum:          33,
			TargetText:       "super.parentExecute()",
			ExpQualifiedName: "com.test.case1.ParentService.parentExecute()",
			ExpKind:          model.Method,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 2.3: 接口/多态方法调用",
			LineNum:          37,
			TargetText:       "listener.onEvent()",
			ExpQualifiedName: "com.test.CallCase1$EventListener.onEvent()",
			ExpKind:          model.Method,
			ExpIsExternal:    false,
		},

		// 维度 3: 方法重载决议
		{
			Name:             "场景 3.1: 基本类型重载精准匹配 (int)",
			LineNum:          43,
			TargetText:       "process(100)",
			ExpQualifiedName: "com.test.CallCase1.process(int)",
			ExpKind:          model.Method,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 3.2: 多参数对象重载匹配",
			LineNum:          46,
			TargetText:       "process(\"admin\", currentUser)",
			ExpQualifiedName: "com.test.CallCase1.process(String,User)",
			ExpKind:          model.Method,
			ExpIsExternal:    false,
		},

		// 维度 4: 链式连续调用
		{
			Name:             "场景 4.1: Builder 链式流转调用 (build)",
			LineNum:          52,
			TargetText:       "new UserBuilder().setName(\"Alice\").build()",
			ExpQualifiedName: "com.test.CallCase1$UserBuilder.build()",
			ExpKind:          model.Method,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 4.2: 跨类深度链式调用 (getAmount)",
			LineNum:          55,
			TargetText:       "currentUser.getOrders().get(0).getAmount()",
			ExpQualifiedName: "com.test.case1.Order.getAmount()",
			ExpKind:          model.Method,
			ExpIsExternal:    false,
		},

		// 维度 5: 高阶函数与方法引用
		{
			Name:             "场景 5.1: 方法引用 (::)",
			LineNum:          61,
			TargetText:       "User::getName",
			ExpQualifiedName: "com.test.case1.User.getName()",
			ExpKind:          model.Method,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 5.2: 未导入外部工具类方法调用 (保底)",
			LineNum:          64,
			TargetText:       "UnimportedTool.runTask()",
			ExpQualifiedName: "UnimportedTool.runTask()",
			ExpKind:          model.Method,
			ExpIsExternal:    true,
		},
	}

	// 5. 运行断言，针对 model.Call 依赖动作捕获点进行比对
	runSymbolAsserts(t, gCtx, fCtx, captures, symbolResolver, callExpectations, model.Call)
}

func TestSymbolResolver_Use(t *testing.T) {
	// 1. 获取测试文件路径
	mainTestFile := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "use", "UseCase1.java"))
	depParent := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "use", "case1", "ParentClass.java"))
	depUser := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "use", "case1", "User.java"))
	depRole := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "use", "case1", "Role.java"))

	// 2. 送入 Phase 1 Collection 建立符号全局索引
	gCtx := test.RunPhase1Collection(t, []string{
		mainTestFile, depParent, depUser, depRole,
	})

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

	// 4. 定义包含 4 大维度的 Use (值读取) 预期断言矩阵
	useExpectations := []TestExpectation{
		// 维度 1: 变量与形参读取
		{
			Name:             "场景 1.1: 局部变量读取 (x)",
			LineNum:          17,
			TargetText:       "x",
			ExpQualifiedName: "com.test.UseCase1.execute(String,Object).x",
			ExpKind:          model.Variable,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 1.2: 方法形参读取 (param)",
			LineNum:          20,
			TargetText:       "param",
			ExpQualifiedName: "com.test.UseCase1.execute(String,Object).param",
			ExpKind:          model.Variable,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 1.3: Catch 异常变量读取 (e)",
			LineNum:          26,
			TargetText:       "e",
			ExpQualifiedName: "com.test.UseCase1.execute(String,Object).block$1.e", // 匹配作用域修正后的 QN
			ExpKind:          model.Variable,
			ExpIsExternal:    false,
		},

		// 维度 2: 成员字段与继承属性读取
		{
			Name:             "场景 2.1: 当前类实例字段读取 (count)",
			LineNum:          33,
			TargetText:       "this.count",
			ExpQualifiedName: "com.test.UseCase1.count",
			ExpKind:          model.Field,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 2.2: 父类继承字段读取 (parentField)",
			LineNum:          36,
			TargetText:       "super.parentField",
			ExpQualifiedName: "com.test.case1.ParentClass.parentField",
			ExpKind:          model.Field,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 2.3: 对象属性直接读取 (user.name)",
			LineNum:          40,
			TargetText:       "user.name",
			ExpQualifiedName: "com.test.case1.User.name",
			ExpKind:          model.Field,
			ExpIsExternal:    false,
		},

		// 维度 3: 常量与枚举读取
		{
			Name:             "场景 3.1: 类静态常量读取 (DEFAULT_NAME)",
			LineNum:          46,
			TargetText:       "Constants.DEFAULT_NAME",
			ExpQualifiedName: "com.test.UseCase1$Constants.DEFAULT_NAME",
			ExpKind:          model.Field,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 3.2: 枚举项读取 (ADMIN)",
			LineNum:          49,
			TargetText:       "Role.ADMIN",
			ExpQualifiedName: "com.test.case1.Role.ADMIN",
			ExpKind:          model.EnumConstant, // 提取为枚举常量
			ExpIsExternal:    false,
		},

		// 维度 4: 高级语法读取与未导入保底
		{
			Name:             "场景 4.1: 模式匹配变量读取 (s)",
			LineNum:          56,
			TargetText:       "s",
			ExpQualifiedName: "com.test.UseCase1.execute(String,Object).block$2.s",
			ExpKind:          model.Variable,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 4.2: 未导入外部类静态常量读取 (保底)",
			LineNum:          60,
			TargetText:       "UnimportedConfig.TIMEOUT",
			ExpQualifiedName: "UnimportedConfig.TIMEOUT",
			ExpKind:          model.Field,
			ExpIsExternal:    true,
		},
	}

	// 5. 运行断言，针对 model.Use 依赖动作捕获点进行比对
	runSymbolAsserts(t, gCtx, fCtx, captures, symbolResolver, useExpectations, model.Use)
}

func TestSymbolResolver_Assign(t *testing.T) {
	// 1. 获取测试文件路径
	mainTestFile := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "assign", "AssignCase1.java"))
	depParent := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "assign", "case1", "ParentClass.java"))
	depUser := test.GetTestFilePath(filepath.Join("resolver", "symbol_resolve", "assign", "case1", "User.java"))

	// 2. 送入 Phase 1 Collection 建立符号全局索引
	gCtx := test.RunPhase1Collection(t, []string{
		mainTestFile, depParent, depUser,
	})

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

	// 4. 定义包含 4 大维度的 Assign (写入) 预期断言矩阵
	assignExpectations := []TestExpectation{
		// 维度 1: 变量声明初始化与重新赋值
		{
			Name:             "场景 1.1: 局部变量声明初始化 (x)",
			LineNum:          15,
			TargetText:       "int x = 10",
			ExpQualifiedName: "com.test.AssignCase1.execute(User).x",
			ExpKind:          model.Variable,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 1.2: 已有局部变量重新赋值 (x)",
			LineNum:          18,
			TargetText:       "x = 20",
			ExpQualifiedName: "com.test.AssignCase1.execute(User).x",
			ExpKind:          model.Variable,
			ExpIsExternal:    false,
		},

		// 维度 2: 成员字段与继承属性写入
		{
			Name:             "场景 2.1: 当前类实例字段写入 (count)",
			LineNum:          24,
			TargetText:       "this.count = 100",
			ExpQualifiedName: "com.test.AssignCase1.count",
			ExpKind:          model.Field,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 2.2: 父类继承字段写入 (parentField)",
			LineNum:          27,
			TargetText:       "super.parentField = \"new_value\"",
			ExpQualifiedName: "com.test.case1.ParentClass.parentField",
			ExpKind:          model.Field,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 2.3: 跨类对象属性写入 (user.name)",
			LineNum:          30,
			TargetText:       "user.name = \"Bob\"",
			ExpQualifiedName: "com.test.case1.User.name",
			ExpKind:          model.Field,
			ExpIsExternal:    false,
		},

		// 维度 3: 复合赋值与自增自减
		{
			Name:             "场景 3.1: 复合运算符赋值 (total += 50)",
			LineNum:          36,
			TargetText:       "this.total += 50",
			ExpQualifiedName: "com.test.AssignCase1.total",
			ExpKind:          model.Field,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 3.2: 自增运算符 (count++)",
			LineNum:          39,
			TargetText:       "this.count++",
			ExpQualifiedName: "com.test.AssignCase1.count",
			ExpKind:          model.Field,
			ExpIsExternal:    false,
		},

		// 维度 4: 类静态常量/字段与未导入保底
		{
			Name:             "场景 4.1: 静态成员字段写入 (Config.DEBUG_MODE)",
			LineNum:          45,
			TargetText:       "Config.DEBUG_MODE = true",
			ExpQualifiedName: "com.test.AssignCase1$Config.DEBUG_MODE",
			ExpKind:          model.Field,
			ExpIsExternal:    false,
		},
		{
			Name:             "场景 4.2: 未导入外部类静态字段写入 (保底)",
			LineNum:          48,
			TargetText:       "UnimportedConfig.FLAG = false",
			ExpQualifiedName: "UnimportedConfig.FLAG",
			ExpKind:          model.Field,
			ExpIsExternal:    true,
		},
	}

	// 5. 运行断言，针对 model.Assign 依赖动作捕获点进行比对
	runSymbolAsserts(t, gCtx, fCtx, captures, symbolResolver, assignExpectations, model.Assign)
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
