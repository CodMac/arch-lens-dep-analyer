# 语言扩展 SPI 设计

> 适用版本：当前主干

## 1. 概述

引擎采用**全局注册表 + init() 自动注册**的语言扩展模型。core 定义契约（SPI 接口），各语言实现契约并通过 `init()` 注册，引擎通过注册表按 `Language` 分派。

```
新语言 x/go
  ├─ 实现全部 SPI（collector/binder/extractor/...）
  └─ init() 注册 → import _ ".../x/go" 即接入
```

## 2. 核心注册表

```go
// core/language.go
type Language string

const (
    LangJava Language = "java"  // 已实现
    LangGo   Language = "go"    // 已声明，尚未实现（见 roadmap.md）
    LangC    Language = "c"     // 已实现（x/cpp 共用实现）
    LangCPP  Language = "cpp"   // 已实现（x/cpp 共用实现）
)

// 全局注册表（各语言 init() 调用）
func Register(impl *LanguageImpl)  // 注册语言实现（重复注册覆盖）
func Get(lang Language) (*LanguageImpl, error) // 获取（未注册则报错）

// LanguageImpl 聚合一个语言的全部 SPI 实现
type LanguageImpl struct {
    Language       Language
    Parser         ParserFactory     // 创建语言解析器
    Collector      Collector
    Binder         Binder
    Extractor      Extractor
    Linker         Linker
    NoiseFilter    NoiseFilter
    SymbolResolver SymbolResolver
    SymbolBuilder  SymbolBuilder
}
```

**设计要点**：各语言聚合为单一 `LanguageImpl`，一次注册全部能力，结构清晰。

## 3. SPI 接口清单

| 接口 | 方法 | 职责 |
|---|---|---|
| `ParserFactory` | `NewParser() Parser` | 创建语言解析器 |
| `Parser` | `ParseFile(path) (ast.SourceFile, error)` + `Close()` | 解析源文件为 AST |
| `Collector` | `CollectDefinitions(src ast.SourceFile) (FileContext, error)` | 收集符号定义 |
| `Binder` | `BindSymbols(ctx GlobalContext) error` | 符号绑定 |
| `Extractor` | `Extract(relPath string, ctx GlobalContext) ([]*model.DependencyRelation, error)` | 提取关系 |
| `Linker` | `LinkHierarchy(ctx GlobalContext) []*model.DependencyRelation` | 层级拓扑 |
| `NoiseFilter` | `SetLevel(FilterLevel)` + `IsNoise(rel) bool` | 降噪 |
| `SymbolResolver` | `BuildQualifiedName` + `IsPrimitive` + `ResolveType` + `ResolveFunc` + `ResolveVar` + `ResolveAction` | 符号解析 |
| `SymbolBuilder` | `BuildQualifiedName(parentQN, name) string` + `IsPrimitive(typeName) bool` | QN 构建 |

> **上下文分层（2026-11-06 接口化 / 2026-11-09 语言实现下沉）**：`FileContext` / `GlobalContext` 均为**接口**，SPI 签名只使用接口类型。通用索引/锁由 `core.BaseFileContext` / `core.BaseGlobalContext` 提供；**语言专属语义（包/命名空间注册、重载索引 kind、命名空间列表）由 `x/{lang}/context` 实现**，core 不含语言硬编码。详见 [architecture.md](architecture.md) §3.3.1。

### 3.1 新语言必须提供的上下文

```go
// x/{lang}/context/file_context.go
type FileContext struct{ *core.BaseFileContext }
func NewFileContext(...) *FileContext
// 单包语义（Java）额外实现 core.PackageAwareFileContext；
// 多命名空间语义（C++）额外实现本包的 NamespaceAware。

// x/{lang}/context/global_context.go
type GlobalContext struct{ *core.BaseGlobalContext }
func NewGlobalContext(r core.SymbolResolver, b core.SymbolBuilder) core.GlobalContext
func (gc *GlobalContext) RegisterFileContext(fc core.FileContext)   // 先注册包/ns 再委托 Base
```

- 语言专属能力用**可选能力接口**表达（`core.PackageAwareFileContext` / `cppctx.NamespaceAware`），core 接口不含语言概念。
- 重载索引 kind 通过 `core.GlobalContextRules{OverloadKinds: ...}` 注入。
- 语言专属导入属性写入 `ImportInfo.Attrs`（key 带语言前缀）。
- 在 `LanguageImpl.NewGlobalContext` 中挂载工厂；引擎经该字段创建上下文。

## 4. 注册流程（以 Java 为例）

```go
// x/java/init.go
func init() {
    resolver := symbol_resolver.New()
    core.Register(&core.LanguageImpl{
        Language:       core.LangJava,
        Parser:         &JavaParserFactory{},
        Collector:      collector.New(resolver),
        Binder:         binder.New(),
        Extractor:      extractor.New(),
        Linker:         linker.New(),
        NoiseFilter:    noise_filter.New(core.LevelBalanced),
        SymbolResolver: resolver,
        SymbolBuilder:  symbol_builder.New(),

        NewGlobalContext: javactx.NewGlobalContext,
    })
}
```

**接入方式**：任意入口只需 `import _ "github.com/CodMac/arch-lens-dep-analyer/x/java"`，init() 自动完成注册。

### 4.1 一次注册多种语言（C/C++ 模式）

`x/cpp/init.go` 用同一套 SPI 实例注册两个 `Language`：

```go
func init() {
    resolver := symbol_resolver.New()
    builder := symbol_builder.New()
    impl := func(lang core.Language) *core.LanguageImpl {
        return &core.LanguageImpl{
            Language: lang, Parser: &CppParserFactory{},
            Collector: collector.New(resolver), Binder: binder.New(resolver),
            Extractor: extractor.New(resolver), Linker: linker.New(),
            NoiseFilter: noise_filter.New(core.LevelBalanced),
            SymbolResolver: resolver, SymbolBuilder: builder,
            NewGlobalContext: cppctx.NewGlobalContext,
        }
    }
    core.Register(impl(core.LangC))
    core.Register(impl(core.LangCPP))
}
```

适用前提：两种语言的 AST 与语义足够接近（C 是 C++ 子集），差异仅在 Parser 层（按扩展名选 grammar）。若语义差异大（如 Go 的隐式接口实现），应各自独立实现。

## 5. AST 抽象（core/ast）

引擎用**薄封装接口**隔离 AST 库：

```go
// core/ast/node.go
type Node interface {
    Kind() string
    Utf8Text(src []byte) string
    ChildByFieldName(name string) Node
    ChildByFieldNameRecursive(name string) Node
    NamedChild(i int) Node
    NamedChildCount() int
    // ...导航能力
}

type SourceFile interface {
    Root() Node
    Content() []byte
    Path() string
}
```

**实现方**：`x/{lang}/ast/` 提供 `Node` 适配器（包装 `*sitter.Node`），把具体节点映射到接口（`x/java/ast/node.go`、`x/cpp/ast/node.go`）。

> 注意：tree-sitter 节点是含 C 指针的 struct，**不能用 `==` 比较**；`Equal` 必须比较节点 `Id()`（见 `x/cpp/ast/node.go`）。

**好处**：
- core/engine/model 完全不感知 tree-sitter。
- 换 AST 库（如 go-tree-sitter 升级、或未来换解析器）只需改适配器。
- 测试可用 mock Node 构造任意 AST 形状。

## 6. 新增语言步骤（Go 为例）

1. **创建目录**：`x/go/`。
2. **AST 适配**：`x/go/ast/` 实现 core/ast.Node（包装 tree-sitter-go）。
3. **实现 SPI**：collector / binder / extractor / linker / noise_filter / symbol_resolver / symbol_builder。
4. **注册**：`x/go/init.go` 调 `core.Register`。
5. **测试**：建 testdata/go/，沿用三层测试模式。

> 语言实现边界与能力要求，见 [languages/java.md](languages/java.md) 与 [languages/cpp.md](languages/cpp.md)；多语言推进优先级见 [roadmap.md](roadmap.md)。

## 7. 设计权衡

**为什么保留注册表 + init() 而非显式注册？**

| 方案 | 优点 | 缺点 |
|---|---|---|
| 全局注册表 + init() | 新语言只需 import blank；引擎零改动 | 隐式依赖；注册顺序不可控（靠 init 保证） |
| 显式注册 | 依赖可见 | 每加语言要改入口代码 |
| 编译期自注册 | 最灵活 | Go 无原生支持，复杂度高 |

选择「注册表 + init()」：Go 的 init() 保证所有依赖包先初始化，注册顺序稳定；`import _` 接入方式对使用者最友好。
