# 测试体系设计

> 适用版本：当前主干

## 1. 概述

测试体系由**三层单元测试**与**端到端黄金回归集**组成。testdata 下的源码样例与期望断言是黄金回归资产，随架构持续演进。

```
┌─────────────────────────────────────────────┐
│ 端到端测试（e2e）                              │
│   输入源码树 → 断言完整输出（JSONL 快照）       │
├─────────────────────────────────────────────┤
│ 三层单元测试                                  │
│   第1层 Extractor：关系提取断言矩阵            │
│   第2层 Resolver：符号解析断言（Java 三段式 / C++ 两段式） │
│   第3层 Collector：符号定义收集断言            │
├─────────────────────────────────────────────┤
│ testdata/  黄金回归集（源码样例 + 期望断言）    │
└─────────────────────────────────────────────┘
```

## 2. 三层测试详述

三层测试按语言分目录：`x/java/test/`（Java）、`x/cpp/test/`（C/C++）。两处各自提供 `helper.go`，暴露同一组流水线搭建辅助（`RunCollection` / `RunExtraction` / `RunPipeline`），语义一致。

### 第 1 层：Extractor 测试（extractor_test.go）

**断言模式**：解析源码注释中的期望块（`parseExpectationComments`）为 `expectedCase`，跑 Collect+Bind+Extract 后断言每条期望命中。无注释断言的 suite（EXTEND/IMPLEMENT）用显式期望表 `explicitExpectations`。

```go
type expectedCase struct {
    Line    int                  // 期望关系触发的源码行号（注释块起始行）
    Source  string               // 期望 Source QN
    Targets []string             // 期望 Target QN（可多个）
    RelType model.DependencyType // 期望关系类型（空表示跟随 suite）
    Mores   map[string]string    // 期望 Mores（key -> substring）
}
```

**suite 覆盖**（`extractorSuites`，相对 testdata/extractor）：

| 关系类型 | 测试文件 |
|---|---|
| Call | `call/CallRelationSuite.java` |
| Create | `create/CreateRelationSuite.java` |
| Assign | `assign/AssignRelationSuite.java` |
| Cast | `cast/CastRelationSuite.java` |
| Use | `use/UseRelationSuite.java` |
| Extend | `extend/ExtendRelationSuite.java` |
| Implement | `implement/ImplementRelationSuite.java` |
| Parameter | `parameter/ParameterRelationSuite.java` |
| Return | `return/ReturnRelationSuite.java` |
| Throw | `throw/ThrowRelationSuite.java` |
| Annotation | `annotation/AnnotationRelationSuite.java` |
| TypeArg | `type_arg/TypeArgRelationSuite.java` |
| Capture | `capture/CaptureRelationSuite.java` |

### 第 2 层：Resolver 测试（resolver_test.go）

**断言模式**：遍历 AST 定位动作节点（Call/Create/Cast/Throw/Assign/Use），对每个动作点调用 `ResolveAction`，断言解析结果非空且限定名合法。

```go
var actionKinds = []struct {
    kind   string
    relTyp model.DependencyType
}{
    {"method_invocation", model.Call},
    {"method_reference", model.Call},
    {"object_creation_expression", model.Create},
    {"cast_expression", model.Cast},
    {"throw_statement", model.Throw},
    {"assignment_expression", model.Assign},
}
```

**覆盖**：Call（method_invocation/method_reference）、Create、Cast、Throw、Assign 动作点的三段式解析。

### 第 3 层：Collector 测试（collector_test.go）

逐断言验证元素定义收集：

| 测试 | 验证内容 |
|---|---|
| `TestCollectorBase` | 基础类/方法/字段定义 |
| `TestCollectorFileMetrics` | 文件度量（LOC 等） |
| `TestCollectorEnum` | 枚举与枚举常量 |
| `TestCollectorRecord` | Record 组件字段与访问器 |
| `TestCollectorGeneric` | 泛型定义 |
| `TestCollectorClass` | 嵌套类 |
| `TestCollectorLombok` | Lombok 语法糖生成 |

### 2.4 C/C++ 三层测试（x/cpp/test/）

C/C++ 不使用 Java 的「源码注释期望块」模式，改用**显式断言函数**（`FindRel` + 谓词），逐条校验关系：

| 测试 | 覆盖 |
|---|---|
| `TestCollectorC` | C 函数/声明/结构体/枚举/typedef/宏/全局与局部变量/匿名 struct 承载（基础样例 `collector/base`） |
| `TestCollectorCBase` | 基准元素：文件/函数+原型/全局变量/结构体/枚举/typedef/宏（`collector/c/base`） |
| `TestCollectorCFunc` | 函数族：`(void)`/`()`/可变参数/static/inline/递归/函数指针返回/数组指针返回 |
| `TestCollectorCDeclDef` | 同文件声明+定义共享 QN、前向声明覆盖、extern 声明+定义、头文件原型 |
| `TestCollectorCStruct` | 具名/嵌套/位域/匿名 union 成员/匿名 struct 多声明器/union |
| `TestCollectorCEnum` | 具名/显式值/匿名常量挂文件级/前向声明/枚举类型变量 |
| `TestCollectorCTypedef` | 基础/指针/数组/函数指针别名、多声明器、链式、匿名 struct 承载 |
| `TestCollectorCMacro` | 对象宏/函数式宏/空宏/多行宏/条件分支同名宏 |
| `TestCollectorCVariable` | 全局/static/const/局部/块作用域/多声明器/混合声明器 |
| `TestCollectorCParam` | 具名/无名指针/多级指针/数组/函数指针/const/struct+enum 关键字 |
| `TestCollectorCConditional` | `#ifdef`/`#ifndef`/`#if`/`#elif`/`#else`/嵌套分支标注（`cpp.condition`） |
| `TestCollectorCEdge` | 类型引用负例/块内重名/重载式命名/无名指针参数/嵌套结构体 |
| `TestCollectorCpp` | namespace/class/继承/模板/纯虚/访问级别/enum class/类外定义（基础样例 `collector/base`） |
| `TestCollectorCppBase` | 基准元素：16 种 ElementKind 冒烟（`collector/cpp/base`） |
| `TestCollectorCppNamespace` | 嵌套 namespace / C++17 `a::b` / inline / 匿名 `$anon` / alias / using |
| `TestCollectorCppDeclDef` | 同文件声明+定义、头文件原型、前向声明覆盖、类外定义、extern 声明+定义、只声明不定义 |
| `TestCollectorCppClass` | 访问级别、嵌套 class/struct/enum、friend、static/const/mutable 成员、final、抽象类 |
| `TestCollectorCppCtorDtor` | 默认/带参/explicit/拷贝/移动/双参构造、析构、`= default`/`= delete`、委托构造、成员初始化列表 |
| `TestCollectorCppOperator` | 算术/比较/赋值/下标/调用/自增自减/转换运算符/内存分配/逻辑位运算、成员与非成员 |
| `TestCollectorCppTemplate` | 类/函数模板、可变参数、默认模板参数、全特化/偏特化（QN 剥离实参）、别名模板、concept |
| `TestCollectorCppInherit` | 单/多继承、virtual 继承、菱形、private/protected 继承、override/final |
| `TestCollectorCppEnum` | 普通 enum / enum class / 指定底层类型 / 前向声明 / 匿名 enum / 显式值 |
| `TestCollectorCppTypedefUsing` | typedef 基础/指针/引用/数组/函数指针/多声明器/链式、using、别名模板、匿名 struct 承载 |
| `TestCollectorCppVariable` | 全局/static/const/constexpr/thread_local/extern/alignas、auto、引用/右值引用、指针、数组、结构化绑定、多声明器 |
| `TestCollectorCppFunction` | 自由函数、可变参数、noexcept、constexpr、尾置返回、默认实参、重载组、static/inline、extern "C" |
| `TestCollectorCppParam` | 值/指针/多级指针/左值引用/右值引用/const/数组/函数指针/默认值/可变参数/无名参数 |
| `TestCollectorCppLambda` | 无捕获/值捕获/引用捕获/全捕获、generic、mutable、嵌套、作字段/实参/返回值 |
| `TestCollectorCppScope` | 裸 block / if / else / for / range-for / while / do / switch case / try-catch / 嵌套 block / 块内重名 / 局部类型 |
| `TestCollectorCppMacro` | 对象宏/函数式宏/可变参数宏/多行宏/空宏/字符串化/连接/条件分支同名宏/#undef |
| `TestCollectorCppConditional` | `#if`/`#ifdef`/`#ifndef`/`#elif`/`#else`/嵌套取最内层/`#pragma` |
| `TestCollectorCppEdge` | 重名消歧、匿名 struct/union 承载、类型引用负例、前向声明、空 struct/函数、无名参数 |
| `TestCollectorConditional` | 条件编译分支标注（`collector/cond` 基础样例） |
| `TestExtractorStructural` | Import/Contain/Extend/Parameter/Return/TypeArg |
| `TestExtractorActions` | Call/Assign/Use/Cast |
| `TestExtractorImplLink` | 跨文件签名配对、static 不配对、类外定义配对 |
| `TestExtractorContainNamespace` | namespace 层级拓扑 |
| `TestResolverCall` | 自由函数/成员/指针成员/静态/命名空间限定调用 |
| `TestResolverCreate` | `new` 表达式 + 构造器 CALL |
| `TestResolverUse` / `TestResolverAssign` | 变量/字段/枚举常量读写 |
| `TestResolverCast` | C 风格强转 + `static_cast` |
| `TestResolverFieldInitializer` | 构造器成员初始化列表 `: a(x)` |
| `TestResolverCapture` | Lambda 捕获 |
| `TestResolverChain` | 链式接收者类型传播：`f()->g()` / `a.b.c` / `this->f()->g()` / 自由函数返回值 / 同名方法区分 |

## 3. 测试数据组织（黄金回归集）

```
x/java/testdata/
├── collector/       # Collector 测试样例（base/enum/class/generic/metrics/sugar/lombok/...）
├── extractor/       # Extractor 测试样例（call/assign/cast/create/throw/use/...）
└── e2e/             # 端到端黄金基线（baseline.golden.jsonl）

x/cpp/testdata/
├── collector/base/  # C/C++ 基础样例（basic.c / basic.cpp / basic.h）
├── collector/cond/  # 条件编译样例（cond.c）
├── collector/c/     # C 全场景元素提取样例（11 组：base/func/decl_def/struct/
│                    #   enum/typedef/macro/variable/param/cond/edge）
├── collector/cpp/   # C++ 全场景元素提取样例（18 组：base/namespace/decl_def/class/
│                    #   ctor_dtor/operator/template/inherit/enum/typedef_using/
│                    #   variable/function/param/lambda/scope/macro/cond/edge）
├── resolver/        # 解析样例（calls.cpp / chain.cpp）
└── e2e/             # 端到端黄金基线（baseline.golden.jsonl）
```

**迁移原则**：
- testdata 样例文件与期望断言作为黄金回归集完整迁移。
- 新架构测试代码重写（因为接口签名变化），但断言语义保持。

## 4. 公共测试辅助（helper.go）

```go
// x/java/test/helper.go、x/cpp/test/helper.go（同名同义）
func RunPipeline(t *testing.T, root string, paths []string, level core.FilterLevel) *engine.EngineContext // 完整五阶段
func RunExtraction(t *testing.T, root string, paths []string) (core.GlobalContext, []*model.DependencyRelation) // Collect+Bind+Extract
func RunCollection(t *testing.T, root string, paths []string) core.GlobalContext // Collect

func FindDef(t *testing.T, gc core.GlobalContext, qn string) *model.CodeElement
func FindRel(t *testing.T, rels []*model.DependencyRelation, pred func(*model.DependencyRelation) bool) *model.DependencyRelation
func HasRel(rels []*model.DependencyRelation, pred func(*model.DependencyRelation) bool) bool
func RelFilter(rels []*model.DependencyRelation, typ model.DependencyType) []*model.DependencyRelation
func PrintCodeElements(t *testing.T, gc core.GlobalContext)
func PrintRelations(t *testing.T, rels []*model.DependencyRelation)
```

> 上下文接口化后，辅助函数签名使用 `core.GlobalContext` 接口（非 `*core.GlobalContext` 指针）。
> 测试内的 `runCollect` 需复刻引擎 `normalizeFilePath` 的相对化逻辑（`SetFilePath` + File 元素 QN/Path 改写），否则 Import/Contain 的路径断言会与 CLI 行为不一致。

## 5. 端到端测试（e2e）

**目标**：输入完整的 testdata 源码树，断言引擎完整输出（JSONL 确定性快照）。

```go
// x/java/test/e2e_test.go、x/cpp/test/e2e_test.go
func TestE2EGolden(t *testing.T) {
    // 1. 收集全部 testdata 下的源文件（Java: *.java；C/C++: *.c/*.h/*.cpp/*.hpp/*.cc/*.cxx）
    // 2. 跑完整五阶段流水线（LevelBalanced）
    // 3. 元素按 QualifiedName（C/C++ 追加 Kind）排序、关系按 (Type,SourceQN,TargetQN,Location) 排序，生成确定性快照
    // 4. 与 baseline.golden.jsonl 比对（-update 标志可刷新）
}
```

**快照确定性策略**：原始输出受 map 迭代顺序影响，因此生成快照前统一排序——元素按 QualifiedName 排序，关系按 `(Type, SourceQN, TargetQN, Location)` 排序，Mores 由 `encoding/json` 按键排序序列化，从而同一输入必然产出同一快照。

**golden 文件**：`x/java/testdata/e2e/baseline.golden.jsonl`、`x/cpp/testdata/e2e/baseline.golden.jsonl`，用 `-update` 标志刷新。

**C/C++ 快照的额外确定性规则**：元素去重键为 `QN + "|" + Kind`（声明/定义同 QN 共存），排序键在 QN 之后追加 `Kind`，保证 `FUNCTION_DECL` 与 `FUNCTION` 的输出顺序稳定。

## 6. 测试命令

```bash
# 全部测试
go test ./...

# 仅 Java 层
go test ./x/java/...
go test ./x/java/test/...

# 仅 C/C++ 层
go test ./x/cpp/...
go test ./x/cpp/test/...

# 更新 e2e 黄金快照
go test ./x/java/test -run TestE2EGolden -update
go test ./x/cpp/test  -run TestE2EGolden -update

# 覆盖率
# go test ./... -coverprofile=cover.out && go tool cover -html=cover.out
```

## 7. 验收标准

1. 黄金回归集（testdata 全部断言）通过。
2. 端到端 CLI 跑通（`go run ./cmd/analyzer` 输出 JSONL）。
3. 文档完备（docs/ 全部章节）。
