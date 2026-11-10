# C/C++ 语言实现边界

> 适用范围：`x/cpp/` · 与 [qn-format.md](../qn-format.md)（QN 契约）、[language-extension.md](../language-extension.md)（SPI 契约）互为补充
>
> 本文档回答「C/C++ 支持到什么程度」：元素/关系覆盖清单、声明-实现配对模型、条件编译策略、实现边界与已知限制。
> 行为契约由 `x/cpp/testdata/e2e/baseline.golden.jsonl`（867 行 = 319 元素 + 548 关系，LevelBalanced）锚定。

## 1. 总体状态

C 与 C++ **共用同一套实现**（`x/cpp/`），仅 Parser 层按扩展名选 grammar。全部 9 个 SPI（Parser / ParserFactory / Collector / Binder / Extractor / Linker / NoiseFilter / SymbolResolver / SymbolBuilder）已实现并经 `x/cpp/init.go` 同时注册为 `core.LangC` 与 `core.LangCPP`。

### 1.1 grammar 选型

`tree-sitter-cpp` 是 `tree-sitter-c` 的**严格超集**（C 独有规则为空集），因此：

| 扩展名 | grammar | 说明 |
|---|---|---|
| `.c` | tree-sitter-c | 纯 C 优先，AST 与 cpp grammar 对纯 C 代码逐节点一致 |
| `.h` | tree-sitter-cpp | 超集，兼容纯 C 头文件与 C++ 头文件 |
| `.cpp` / `.hpp` / `.cc` / `.cxx` | tree-sitter-cpp | C++ |
| 其它 | tree-sitter-cpp | 兜底 |

> CLI 侧 `-lang c` 扫描 `.c`/`.h`，`-lang cpp` 扫描 `.cpp`/`.hpp`/`.cc`/`.cxx`/`.h`（见 `cmd/analyzer/main.go` 的 `langExtensions`）。

## 2. 元素覆盖（CodeElement）

| 元素 | QN 格式 | 说明 |
|---|---|---|
| File | `相对路径`（正斜杠） | 头文件与源文件各自是独立 File 元素 |
| Namespace | `ns1` / `ns1::ns2` | 嵌套 namespace 逐级拆分，每级独立元素；匿名 namespace 记 `$anon` |
| Struct / Class | `ns::类型名` | `struct`/`union` 均产 STRUCT（union 记 `cpp.struct.is_union`）；类内嵌套类型 `Outer::Inner` |
| Enum / EnumConstant | `ns::枚举名` / `ns::枚举名::常量` | `enum class` 记 `cpp.enum.is_scoped=true` |
| Function（定义） | `[ns::]函数名(参数类型列表)` | 有函数体 |
| FunctionDecl（声明） | 同上 | 头文件原型 / 无函数体 declaration |
| Method（定义） | `ns::类名::方法名(参数类型列表)` | 类内定义或类外 `ns::Class::method` 定义 |
| MethodDecl（声明） | 同上 | 类内成员声明 |
| Field | `类QN::字段名` | 含 static / const 字段 |
| Type（typedef / using） | `[ns::]别名` | 底层类型记 `cpp.type.underlying` |
| Variable | `所在作用域QN::变量名` | 局部变量挂函数/块/Lambda；全局变量挂文件（`cpp.variable.is_global=true`） |
| Macro | `宏名` | `#define`；函数式宏记 `cpp.macro.is_function_like=true`，宏体原文入 `cpp.macro.body` |
| Lambda | `容器QN::lambda$N` | 形参/局部变量挂其下 |
| ScopeBlock | `容器QN::block$N` | 仅非函数体/Lambda 体的 `compound_statement`（if/for/switch/catch 体等） |

### 2.1 参数签名规则

参数类型列表是 QN 的一部分（区分重载），规则：

- 仅保留**类型**，不含参数名：`add(int a, int b)` → `add(int,int)`。
- 保留指针/引用/const 语义：`const char* s` → `const char*`；`int& x` → `int&`。
- 数组退化为 `[]`：`int arr[10]` → `int[]`。
- 函数指针参数记 `(*)`：`int (*cb)(int)` → `int(*)`。
- 可变参数记 `...`。
- `f(void)` 等价无参 → `f()`。
- 模板实参**从 QN 中剥离**（`Box<int>::run` → `Box::run`），模板参数原文记 `cpp.template.params`。

### 2.2 匿名 struct / union 的承载规则

匿名 struct/union **不产独立元素**，其字段挂到承载实体下：

| 形态 | 承载实体 | 示例 |
|---|---|---|
| `typedef struct { int x; } Point;` | typedef 名 | `Point::x` |
| `struct { int a; } obj;` | 变量名 | `obj::a` |
| `struct Outer { struct { int a; } pair; };` | 外层字段名 | `Outer::pair::a` |

### 2.3 声明 / 定义双元素

`FunctionDecl`/`MethodDecl` 与 `Function`/`Method` **成对共存**，QN 相同、Kind 不同：

- `BaseGlobalContext` 的全量索引键为 `QN + "|" + Kind`，两者不互相吞并。
- `FindByQualifiedName` **定义优先**：先注册声明、后注册定义时定义覆盖声明；纯声明（无定义）仍可命中。

## 3. 关系覆盖（DependencyRelation）

### 3.1 结构关系

| 关系 | 说明 |
|---|---|
| Import | 文件级 `#include`。`"foo.h"`（项目内）按路径后缀匹配 File 元素；`<stdio.h>`（系统）产外部 File 符号。`cpp.rel.include.is_system` 标记 |
| Contain | 层级拓扑：File → 顶层元素、Namespace → 子 Namespace、Namespace → File、类/结构体/枚举 → 成员、函数 → 局部变量/块/Lambda |
| Extend | C++ 继承（`base_class_clause`），基类名按当前命名空间补全后解析 |
| Parameter | 函数/方法形参类型（`cpp.rel.raw_text`） |
| Return | 函数/方法返回类型（`void` 不产） |
| TypeArg | 变量/字段类型 + 模板实参（`Foo<Bar>` 的 `Bar` 各产一条） |
| ImplLink | 声明-实现链接（定义 → 声明），见 §4 |

### 3.2 动作关系（带 Location）

| 关系 | 触发点 | 说明 |
|---|---|---|
| Call | `call_expression` | 自由函数、`obj.method()`、`ptr->method()`、`Class::staticMethod()`、`ns::func()`、Lambda 变量调用 |
| Create | `new_expression` | C++ `new T(...)`；同时产构造器 CALL |
| Cast | `cast_expression` / `static_cast` 等 | C 风格 `(T)x` 与 `static_cast<T>(x)`；`(void)x` 抑制形态不产 |
| Assign | `assignment_expression` / `init_declarator` / `field_initializer` | 含变量初始化与构造器成员初始化列表 `: a(x)` |
| Use | `identifier` / `field_expression` / `qualified_identifier` | 变量/字段/枚举常量读取 |
| Throw | `throw_statement` | 异常类型（`throw new T` / `throw T(...)` / `throw var`） |
| Capture | 由 Lambda 的 USE/ASSIGN 推导 | 目标为 Lambda 作用域外的局部变量 |

### 3.3 Source 定位

动作关系的 Source 取**包含该节点的最内层可执行元素**（区间最小者）：

- Function / Method / Lambda / ScopeBlock / Field 均可作 Source。
- 局部 Variable **不作 Source**（其初始化表达式归属所在函数）；仅全局 Variable 可作 Source（初始化表达式归属变量自身）。

## 4. 声明-实现链接（ImplLink）

### 4.1 配对模型：签名配对，不靠文件名

真实工程存在多种头↔源形态，故**不按同名 basename 推断**，而是按函数签名配对：

| 形态 | 处理 |
|---|---|
| 1 头 ↔ 1 源（`util.h` ↔ `util.cpp`） | 签名配对自然命中 |
| 1 头 ↔ 多源（`big.h` 声明 5 个函数，分散在多个源文件定义） | 签名配对跨文件命中 |
| `static` 函数（无声明） | **不参与配对** |
| 只声明无定义（`extern int extFunc()`） | 声明照常产出，**不建 ImplLink** |

### 4.2 签名格式

```
<完整 QN>(<参数类型列表>)
```

例：`add(int,int)(int,int)`、`app::Box::twice(int)(int)`。

- 完整 QN 含 namespace / 类前缀，可区分重载与同名。
- 参数类型取 `cpp.function.parameters` 的类型部分（不含参数名）。
- 签名写入关系 Mores `cpp.rel.impl_link.signature`。

### 4.3 方向

`ImplLink: Source = 定义（Function/Method）→ Target = 声明（FunctionDecl/MethodDecl）`。

## 5. 条件编译策略

tree-sitter **不做宏展开**，条件编译各分支在 AST 中全部保留。引擎的策略是「全分支保留 + 分支标注」：

- 收集时维护条件栈，元素注册时写入 `cpp.condition`（如 `#ifdef FOO` / `#ifndef BAZ` / `#if defined(X)` / `#elif defined(BAR)` / `#else`）。
- 嵌套条件取**最内层**标签。
- 不做前置预处理阶段，因此同一符号在不同分支下的多个定义会同时出现（QN 可能相同，按 Kind 共存，重名时 `$N` 消歧）。

## 6. QN 与解析规则

### 6.1 分隔符

- QN 分隔符为 **`::`**（`SymbolBuilder.BuildQualifiedName`，`x/cpp/symbol_builder`）。
- 文件元素 QN 为相对路径（正斜杠），与 Java 一致。

### 6.2 两段式解析

`SymbolResolver` 采用**两段式**架构（Java 为「NodeContextResolver → ExpressionSegmenter → ChainResolver」三段式）：

**段 1 归一化**：`normalize(node, src)` 把任意动作 AST 节点折叠为统一的 `expr` 描述，吸收形态差异。

```go
type expr struct {
    form     exprForm // formIdent / formMember / formScoped / formCall / formNew / formCast / formText
    node     coreast.Node
    recv     *expr    // 递归接收者
    name     string
    typeText string   // new T / (T)x / static_cast<T> 的类型文本
    text     string   // 纯文本接收者（公开 SPI 传入 receiver 字符串时）
    argc     int
}
```

覆盖的 AST 形态：`identifier`（formIdent）、`field_expression`（formMember，`argument` 即 receiver）、`qualified_identifier`（formScoped）、`call_expression`（formCall）、`new_expression`（formNew）、`cast_expression`/`static_cast`（formCast）、`template_function`、`pointer_expression`（`*p` 解引用）、`parenthesized_expression`、`subscript_expression`（`arr[i]`）。

**段 2 查找**：`resolveExpr(gc, fc, e, want)` 只依赖 `expr`，按期望类型分派：

| want | 入口 | 行为 |
|---|---|---|
| `wantValue` | `resolveValue` | 变量/字段/枚举常量 |
| `wantFunc` | `resolveFunc` | 函数/方法（含重载挑选） |
| `wantType` | `typeOf` | 类型符号；链式场景递归 `typeOf(e.recv)` 传播接收者类型 |

查找规则：

1. 短名沿词法作用域链上溯：`ns::Class::method` → `ns::Class` → `ns` → 全局。
2. 类型解析先查精确 QN，再查文件内短名，再查全局短名；相对限定名（`core::Base`）按文件内 namespace 补全。
3. 成员查找沿类型继承链（`cpp.class.superclass`）递归。

`ResolveAction` 缩为「normalize → 按 relType 选 want」的薄分派；`ResolveVar`/`ResolveFunc` 公开 SPI 也构造 `expr` 后走同一查找路径，保证入口归一。

**为什么不需要三段式**：C/C++ 的 AST 已把 receiver 与成员名分开（`call_expression.function` 为 `field_expression`，其 `argument` 即 receiver），段 1 的归一化直接吸收该结构，无需 Java 的「向上爬升取上下文 + 拉平成链」两步；且无 `super`/`Xxx.class`/方法引用/泛型实例化等语法。

### 6.2.1 链式接收者类型传播

链式调用（`f()->g()`、`a.b.c`）在段 2 的 `typeOf` 中递归传播接收者类型，否则会退化为全局短名匹配（同名方法误命中）：

| 接收者形态 | 解析方式 |
|---|---|
| `f()->g()` | 解析 `f` 得到函数元素，取其 `cpp.function.return_type_with_qn`（退回 `return_type`）作为 receiver 类型 |
| `a.b.c` | 递归 `typeOf(e.recv)`，取字段的 `cpp.field.type_with_qn`（退回 `raw_type`） |
| `makeL1(h)->g()` | 同上，自由函数返回值作接收者 |
| `this->f()->g()` | `this` → 所属类，后续段按调用/字段继续传播 |
| `*p` / `(expr)` / `arr[i]` | 归一化阶段剥去解引用/括号/下标后递归 |
| `new T()` | 直接取 `T` |

解析顺序：段 1 已把接收者折叠为 `expr` 子树，段 2 的 `typeOf` 直接递归该子树；仅当接收者以纯文本形式（`formText`，公开 SPI 传入 `receiver` 字符串）出现时才走文本匹配（`this` / 类型名 / 简单变量）。裸标识符作接收者时**变量优先于同名类型**（`obj.method()` 语义）。

已知边界：接收者为外部类型的方法时，返回类型无法解析（无源码符号），链在此处断开并退回短名兜底。

### 6.3 typedef / using 别名链

- Collector 记录 `cpp.type.underlying`（底层类型文本）。
- Binder 构建全局别名表并在类型绑定时**递归展开**（最多 8 层防环），再解析为 QN。
- **匿名承载 typedef 不展开**：`typedef struct {...} Point;` 的底层文本是匿名结构体（`struct { int x; int y; }`），不是可解析的类型名；此时别名本身即目标类型，Binder 直接绑定到 `Point`。否则变量的 `cpp.variable.type_with_qn` 会退化为匿名结构体文本，`p.x` 的接收者类型解析失败后退回全局短名扫描，命中同名参数/变量（结果随并行收集顺序变化而不确定）。

### 6.4 外部符号兜底

未命中源码符号库时：

- 基本类型（`int`/`double`/`size_t`/… 及 `auto`）→ 产 `PRIMITIVE` 元素。
- 模板类型变量（单大写字母 `T`/`E`/`S`/`K`/`V`）→ 不产外部元素。
- 其余 → 外部符号，`PackageName` 按 `::` 拆分取前缀，无 `::` 归 `<external>`。
  不复用 `model.NewExternalElement`（其按点号拆分，C++ QN 会拆错），使用 `x/cpp/helper.NewExternalElement`。

## 7. 已知边界与限制

| # | 边界 | 说明 |
|---|---|---|
| 1 | **运算符重载** | `a + b` 不建模为 `operator+` 调用（已确认的已知限制）。`operator=`/`operator==` 等**成员声明**仍作为 Method/MethodDecl 元素收集 |
| 2 | **宏不展开** | `#define` 只收集宏名+宏体，不参与符号解析；宏调用的语义无法追踪 |
| 3 | **条件编译** | 全分支保留，不做宏环境求解；同一符号的多分支定义会并存 |
| 4 | **模板实例化** | 不建模具体实例化（`Box<int>` 与 `Box<double>` 共用 `Box` 元素）；QN 中剥离模板实参 |
| 5 | **模板特化** | 显式特化（`template<> class X<int>`）与偏特化不单独建模，与主模板同 QN 时按重名 `$N` 处理 |
| 6 | **`= default` / `= delete`** | 无函数体，Kind 保持定义语义（Method），额外记 `cpp.is_declaration=true` |
| 7 | **函数指针变量** | `int (*fp)(int)` 识别为变量（非函数声明）；作为参数类型记 `(*)` |
| 8 | **预处理指令位置** | `#include` 出现在任意位置（含条件分支内）均被收集为文件级 Import，不区分包含位置 |
| 9 | **多继承** | `base_class_clause` 的多个基类各产一条 EXTEND；不区分 `virtual` 继承语义 |
| 10 | **匿名 namespace** | 记名 `$anon`，不与外部链接语义区分 |
| 11 | **`extern "C"`** | `linkage_specification` 内的声明照常收集，不做 C/C++ 链接名区分 |
| 12 | **头文件重复包含** | 无 include guard 求解；同一头文件被多个源 include 时各产一条 Import |
| 13 | **外部类型链式断开** | 链式接收者为外部类型（如 `std::vector<T>::begin()`）时，其返回类型无源码符号可查，链在此处断开并退回短名兜底 |

## 8. 与 SPI 契约的对应

| SPI | C/C++ 实现 | 说明 |
|---|---|---|
| Parser / ParserFactory | `x/cpp/cpp_parser.go` | 按扩展名选 grammar（`languageForPath`） |
| Collector | `x/cpp/collector/` | 五步收集 + 条件栈 + 匿名 struct 承载 + 元数据增强 |
| Binder | `x/cpp/binder/` | raw 类型 → QN，typedef 别名链展开 |
| Extractor | `x/cpp/extractor/` | 结构 + 动作 + Capture 派生 |
| Linker | `x/cpp/linker/` | Contain 拓扑 + ImplLink 签名配对 |
| NoiseFilter | `x/cpp/noise_filter/` | 复用 `core.DefaultNoiseFilter` 三态；ImplLink 恒保留 |
| SymbolResolver | `x/cpp/symbol_resolver/` | 两段式：段 1 `normalize` 归一化为 `expr` + 段 2 `resolveExpr` 作用域链/继承链查找 + 链式接收者类型传播 |
| SymbolBuilder | `x/cpp/symbol_builder/` | QN 用 `::` 拼接 |
| FileContext | `x/cpp/context/` | 内嵌 `core.BaseFileContext` + 实现 `cppctx.NamespaceAware`（`Namespaces()`） |
| GlobalContext | `x/cpp/context/` | 内嵌 `core.BaseGlobalContext`；NAMESPACE 由 Collector 建元素；重载索引收 Function 族 |

## 9. 测试现状

- 三层测试 + e2e 黄金回归集，全部位于 `x/cpp/test/`。
- e2e 基线：`x/cpp/testdata/e2e/baseline.golden.jsonl`（319 元素 + 548 关系，LevelBalanced）。
- 元素覆盖（基线实测）：FILE 20 / NAMESPACE 3 / CLASS 4 / STRUCT 21 / ENUM 6 / ENUM_CONSTANT 14 / TYPE 14 / MACRO 14 / FIELD 48 / FUNCTION 33 / FUNCTION_DECL 8 / METHOD 13 / METHOD_DECL 6 / VARIABLE 109 / SCOPE_BLOCK 5 / LAMBDA 1。
- 关系覆盖（基线实测）：CONTAIN 341 / USE 89 / ASSIGN 34 / TYPE_ARG 28 / CALL 27 / IMPORT 9 / RETURN 7 / IMPL_LINK 4 / PARAMETER 4 / CAST 2 / CAPTURE 1 / CREATE 1 / EXTEND 1。
- 基线（LevelBalanced）中 PARAMETER/RETURN 大幅减少，是因为当前样例的参数/返回类型多为基本类型（`int`/`void`），被外部符号降噪过滤；CONTAIN/USE/ASSIGN 等指向源码符号的关系不受影响。
- 测试用例：
  | 测试 | 覆盖 |
  |---|---|
  | `TestCollectorC` | C 函数/声明/结构体/枚举/typedef/宏/全局与局部变量/匿名 struct 承载 |
  | `TestCollectorCBase` | 基准元素：文件/函数+原型/全局变量/结构体/枚举/typedef/宏 |
  | `TestCollectorCFunc` | 函数族：`(void)`/`()`/可变参数/static/inline/递归/函数指针返回/数组指针返回 |
  | `TestCollectorCDeclDef` | 同文件声明+定义共享 QN、前向声明覆盖、extern 声明+定义、头文件原型 |
  | `TestCollectorCStruct` | 具名/嵌套/位域/匿名 union 成员/匿名 struct 多声明器/union |
  | `TestCollectorCEnum` | 具名/显式值/匿名常量挂文件级/前向声明/枚举类型变量 |
  | `TestCollectorCTypedef` | 基础/指针/数组/函数指针别名、多声明器、链式、匿名 struct 承载 |
  | `TestCollectorCMacro` | 对象宏/函数式宏/空宏/多行宏/条件分支同名宏 |
  | `TestCollectorCVariable` | 全局/static/const/局部/块作用域/多声明器/混合声明器 |
  | `TestCollectorCParam` | 具名/无名指针/多级指针/数组/函数指针/const/struct+enum 关键字 |
  | `TestCollectorCConditional` | `#ifdef`/`#ifndef`/`#if`/`#elif`/`#else`/嵌套分支标注 |
  | `TestCollectorCEdge` | 类型引用负例/块内重名/重载式命名/无名指针参数/嵌套结构体 |
  | `TestCollectorCpp` | namespace/class/继承/模板/纯虚/访问级别/enum class/类外定义 |
  | `TestCollectorConditional` | 条件编译分支标注 |
  | `TestExtractorStructural` | Import/Contain/Extend/Parameter/Return/TypeArg |
  | `TestExtractorActions` | Call/Assign/Use/Cast |
  | `TestExtractorImplLink` | 跨文件签名配对、static 不配对、类外定义配对 |
  | `TestExtractorContainNamespace` | namespace 层级拓扑 |
  | `TestResolverCall` | 自由函数/成员/指针成员/静态/命名空间限定调用 |
  | `TestResolverCreate` | `new` 表达式 + 构造器 CALL |
  | `TestResolverUse` / `TestResolverAssign` | 变量/字段/枚举常量读写 |
  | `TestResolverCast` | C 风格强转 + `static_cast` |
  | `TestResolverFieldInitializer` | 构造器成员初始化列表 |
  | `TestResolverCapture` | Lambda 捕获 |
  | `TestResolverChain` | 链式接收者类型传播：`f()->g()` / `a.b.c` / `this->f()->g()` / 自由函数返回值 / 同名方法区分 |

## 10. 测试命令

```bash
go test ./x/cpp/...                                  # 全部 C/C++ 测试
go test ./x/cpp/test -run TestE2EGolden -update      # 刷新黄金基线
go run ./cmd/analyzer -lang cpp -path <dir> -out-dir <out>
go run ./cmd/analyzer -lang c   -path <dir> -out-dir <out>
```
