# 分层架构设计

> 适用版本：当前主干

## 1. 分层总览

架构分为六个层次，依赖方向严格自上而下，禁止反向依赖：

```
+---------------------------------------------------------------+
| cmd/analyzer     CLI 入口（薄层）                               |
|   只做参数解析、文件扫描、调用 engine，不承载业务逻辑          |
+---------------------------------------------------------------+
| engine/          流水线编排                                     |
|   五阶段 Stage 抽象 + 并发调度 + 上下文生命周期管理            |
+---------------------------------------------------------------+
| core/            核心 SPI（语言无关）                           |
|   语言注册表 + 各阶段接口 + AST 抽象接口                       |
|   ├─ ast/        AST 薄封装（Node/SourceFile 接口）            |
|   └─ 各阶段 SPI 接口定义                                      |
+---------------------------------------------------------------+
| model/           数据模型（纯数据结构，零依赖）                 |
|   CodeElement / DependencyRelation / Mores / Location / Kind   |
+---------------------------------------------------------------+
| x/{lang}/        语言实现（Java / C / C++ 已落地，Go 预留）     |
|   collector/binder/extractor/resolver/desugar/... 通过 SPI 注册 |
+---------------------------------------------------------------+
| output/          导出器（jsonl）                               |
+---------------------------------------------------------------+
```

## 2. 依赖方向规则

```
cmd → engine → core → model
                    ↕        （core 依赖 model；model 零依赖）
          x/{lang} → core + model（语言实现 core 的 SPI 接口）
          output → model
```

- `core` **绝不 import** 任何 tree-sitter 包或 `x/` 下任何语言实现。
- `model` 是纯数据结构包，不 import 除标准库外的任何包。
- `x/{lang}` 是 core SPI 的唯一实现方，通过注册表接入。
- `engine` 只依赖 `core` 与 `model`，通过注册表获取语言实现，不直接依赖 `x/`。
- 解析器不设独立目录，各语言在 `x/{lang}/` 内自持 `*ParserFactory`（如 `x/java/java_parser.go`），经 `core.ParserFactory` SPI 暴露给 engine。

## 3. 各层职责详述

### 3.1 cmd/analyzer（CLI 入口）

**职责**：
- 解析命令行参数（语言、路径、过滤等级、并发数、输出目录）。
- 扫描源码文件（正则匹配）。
- 组装 engine 配置并调用一次完整的流水线。
- 将结果交给 output 层导出。

**不承载**：任何符号解析、关系提取、数据模型构造逻辑。

### 3.2 engine/（流水线编排）

**职责**：
- 定义 `Stage` 接口（Name / Execute / Concurrent）。
- 将五阶段注册为 Stage 并按序执行。
- 管理并发调度（阶段内文件级并行）。
- 管理 EngineContext 生命周期（创建、冻结、释放）。
- 统一清理 Relation 上的私有临时字段（见 data-model.md 临时字段章节）。

**设计要点**：
- Stage 抽象使新增阶段或替换实现成为纯增量操作。
- 阶段间数据传递通过 EngineContext，不共享可变全局状态。

### 3.3 core/（核心 SPI）

**职责**：
- `language.go`：`Language` 类型 + `Register`/`Get` 注册表。
- 各阶段 SPI：`Collector` / `Binder` / `Extractor` / `Linker` / `NoiseFilter`。
- 符号 SPI：`SymbolResolver` / `SymbolBuilder`。
- `ast/`：AST 薄封装接口（`Node` / `SourceFile`）。
- `parser_factory.go`：`ParserFactory` / `Parser` 接口。
- `file_context.go` / `global_context.go`：**上下文接口 + Base 通用实现**（见 §3.3.1）。

**关键约束**：
- 所有接口签名只使用 `core`/`model` 定义的类型，**绝不出现** `*sitter.Node` 等 AST 库类型。
- 接口是契约，语言实现负责把 AST 适配为 core 语义。

#### 3.3.1 上下文分层（2026-11-06 接口化 → 2026-11-09 语言实现下沉）

`FileContext` / `GlobalContext` 是 **core 契约**；通用索引/锁/去重由 `core.BaseFileContext` / `core.BaseGlobalContext` 提供；**语言专属语义由 `x/{lang}/context` 承载**，core 不含任何语言硬编码。

```go
// core/file_context.go —— 只保留语言无关契约
type FileContext interface {
    FilePath() string;  SetFilePath(string)
    Root() ast.Node;  SetRoot(ast.Node)
    SourceBytes() *[]byte; SetSourceBytes(*[]byte)
    Definitions() []*DefinitionEntry; AddDefinition(*DefinitionEntry)
    Imports() map[string][]*ImportInfo; AddImport(string, *ImportInfo)
    FindByShortName(string) ([]*DefinitionEntry, bool)
    FindByQualifiedName(string) (*DefinitionEntry, bool)
    FindByKind(model.ElementKind) ([]*DefinitionEntry, bool)
}

// 可选能力接口：单包语义的语言（Java）实现
type PackageAwareFileContext interface {
    FileContext
    PackageName() string; SetPackageName(string)
}

// 通用导入信息：语言专属属性走 Attrs（core 不解释语义）
type ImportInfo struct {
    RawImportPath string
    Alias         string
    Kind          model.ElementKind
    Location      *model.Location
    Attrs         map[string]any   // "java.import.is_wildcard" / "cpp.include.is_system"
}

// core/global_context.go
type GlobalContext interface {
    RegisterFileContext(FileContext)   // 语言实现可覆盖，先注册包/ns 再委托 Base
    FileContexts() map[string]FileContext
    FileContext(relPath string) (FileContext, bool)
    Definitions() []*DefinitionEntry
    AddDefinition(*DefinitionEntry)
    FindByQualifiedName(qn string) (*DefinitionEntry, bool)
    FindOverloadCandidates(noParamsQN string) ([]*DefinitionEntry, bool)
    BuildQualifiedName(parentQN, name string) string
    Resolver() SymbolResolver
    Builder() SymbolBuilder
}

// 语言策略注入：core 不再硬编码 kind 集合
type GlobalContextRules struct {
    OverloadKinds map[model.ElementKind]struct{}
}
```

**分层职责**

| 层 | 位置 | 内容 |
|---|---|---|
| 契约 | `core/*_context.go` | `FileContext` / `GlobalContext` 接口、`GlobalContextRules`、`ImportInfo` |
| 通用底座 | `core.BaseFileContext` / `core.BaseGlobalContext` | 索引/锁/去重/声明-定义替换 |
| 语言实现 | `x/java/context`、`x/cpp/context` | 内嵌 Base + 覆盖 `RegisterFileContext`；Java 实现 `PackageAwareFileContext`；C++ 实现 `NamespaceAware` |
| 编排 | `engine` | 只面向 core 接口；经 `LanguageImpl.NewGlobalContext` 取得语言实现 |

**语言差异收敛点**

| 差异 | Java | C/C++ |
|---|---|---|
| 上下文可选能力 | `PackageAwareFileContext`（单包） | `cppctx.NamespaceAware`（多 namespace） |
| 包/命名空间注册 | `RegisterFileContext` 内按 `.` 拆 PACKAGE | NAMESPACE 由 Collector 建元素，无需注册 |
| 重载索引 kind | `{Method}` | `{Method, Function, MethodDecl, FunctionDecl}` |
| 导入属性 | `Attrs["java.import.is_wildcard"]` | `Attrs["cpp.include.is_system"]` |

**通用机制（core 内，语言无关）**

- **声明/定义共存（G2）**：`BaseGlobalContext` 全量索引键为 `QN + "|" + Kind`，使 `add(int,int)|FUNCTION_DECL` 与 `add(int,int)|FUNCTION` 共存；主索引 `primaryIndex` 保持「定义优先」，但纯声明（无定义）仍可命中。
- **声明 → 定义原地替换（G2 补强）**：`DefinitionEntry.IsDeclaration` 标记声明条目；同一 `QN + Kind` 下（如 `struct Fwd;` 后接 `struct Fwd {};`、`extern int x;` 后接 `int x;`），后到的定义原地替换先到的声明，FileContext 与 GlobalContext 的索引同步更新，避免前向声明遮蔽真实定义。该判定只看 `IsDeclaration` 字段，与语言 kind 无关。
- **导入属性泛化（G5）**：`ImportInfo` 只保留通用字段，语言专属属性写入 `Attrs`（key 带语言前缀），core 不解释其语义。

> 工厂函数 `core.NewFileContext` / `core.NewGlobalContext` 已删除：新语言**必须**提供 `x/{lang}/context` 并在 `LanguageImpl.NewGlobalContext` 中注册，避免误用通用实现。

### 3.4 model/（数据模型）

**职责**：
- `element.go`：`CodeElement` / `ElementKind` / `Extra` / `Location`。
- `relation.go`：`DependencyRelation` / `DependencyType`。
- `mores.go`：通用 Mores key 常量。

**声明/定义成对 kind（2026-11-06）**：`FunctionDecl`（`FUNCTION_DECL`）与 `MethodDecl`（`METHOD_DECL`）用于 C/C++ 等声明/实现分离语言，与 `Function`/`Method` 成对，QN 相同、Kind 不同。

**关键约束**：
- 零依赖，可被任何层 import。
- Mores 是 `map[string]interface{}`，语言无关 key 定义于 model 层，语言特化 key 定义于各语言 constants 包，避免散落魔法字符串。

### 3.5 x/{lang}/（语言实现）

**职责**：
- 实现 core 定义的全部 SPI 接口。
- 提供语言特有的解析逻辑：collector、binder、extractor、resolver、enricher、desugar。
- 通过 `init()` 自动注册到 core 注册表。

**结构**（以 Java 为例）：
```
x/java/
├── init.go                  # init() 注册全部 SPI
├── java_parser.go           # JavaParserFactory（core.ParserFactory 实现）
├── ast/                     # core/ast 接口适配（包装 tree-sitter 节点）
├── collector/               # Collector 实现（java_collector.go）
├── binder/                  # Binder 实现（java_binder.go）
├── extractor/               # Extractor 实现（java_extractor.go）
├── linker/                  # Linker 实现（java_linker.go）
├── noise_filter/            # NoiseFilter 实现（java_noise_filter.go）
├── symbol_resolver/         # SymbolResolver 实现（java_symbol_resolver.go）
├── symbol_builder/          # SymbolBuilder 实现（java_symbol_builder.go）
├── resolver/                # 三段式解析（node_context / segmenter / chain / member）
├── enricher/                # 元素/关系元数据增强（ele/ 与 rel/ 子包）
├── desugar/                 # 语法糖（Lombok/Record/Enum）
├── query/                   # tree-sitter query 定义（java_query.go）
├── constants/               # 语言特有常量（含 Mores key）
└── helper/                  # 通用解析辅助
```

**结构**（C/C++，共用一套实现）：
```
x/cpp/
├── init.go                  # init() 同时注册 LangC 与 LangCPP
├── cpp_parser.go            # CppParserFactory（按扩展名选 c / cpp grammar）
├── ast/                     # core/ast 接口适配
├── collector/               # 五步收集 + 条件编译栈 + 匿名 struct 承载
├── binder/                  # raw 类型 → QN（typedef 别名链展开）
├── extractor/               # 结构 + 动作 + Capture 派生
├── linker/                  # Contain 拓扑 + ImplLink 签名配对
├── noise_filter/            # 复用 DefaultNoiseFilter，ImplLink 恒保留
├── symbol_resolver/         # 两段式：normalize 归一化为 expr + 作用域链/继承链查找
├── symbol_builder/          # QN 用 :: 拼接
├── constants/               # cpp.* Mores key
└── helper/                  # :: 切分、类型文本归一化、外部符号构造
```

> C/C++ 无独立的 `resolver/` 三段式包：C/C++ 的 AST 已把 receiver 与成员名分开，`symbol_resolver` 内采用两段式（段 1 `normalize` 归一化为统一 `expr`，段 2 `resolveExpr` 查找）。

### 3.6 output/（导出器）

- **output/**：将结果导出为 JSONL（`element.jsonl` 与 `relation_{type}.jsonl`）。只依赖 `model`。

## 4. 模块清单

```
arch-lens-dep-analyer/
├── cmd/analyzer/main.go
├── engine/
│   ├── pipeline.go       # Pipeline 编排器 + Stage 接口
│   ├── stage.go          # 五个内置 Stage
│   ├── context.go        # EngineContext
│   └── parallel.go       # 并发调度器（Scheduler）
├── core/
│   ├── ast/              # Node / SourceFile 接口
│   ├── language.go       # 注册表
│   ├── parser_factory.go # Parser / ParserFactory 接口
│   ├── collector.go
│   ├── binder.go
│   ├── extractor.go
│   ├── linker.go
│   ├── noise_filter.go
│   ├── symbol_resolver.go
│   ├── symbol_builder.go
│   ├── global_context.go # GlobalContext 接口 + BaseGlobalContext（统一 RWMutex）
│   └── file_context.go   # FileContext 接口 + BaseFileContext
├── model/
│   ├── element.go
│   ├── relation.go
│   └── mores.go
├── x/java/
│   ├── ...（见 §3.5）
├── x/cpp/
│   ├── ...（见 §3.5）
├── output/
│   ├── exporter.go
│   └── jsonl.go
├── docs/
├── x/java/testdata/      # Java 黄金回归集
└── x/cpp/testdata/       # C/C++ 黄金回归集
```
