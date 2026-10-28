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
| `Collector` | `CollectDefinitions(src ast.SourceFile) (*FileContext, error)` | 收集符号定义 |
| `Binder` | `BindSymbols(ctx *GlobalContext) error` | 符号绑定 |
| `Extractor` | `Extract(relPath string, ctx *GlobalContext) ([]*model.DependencyRelation, error)` | 提取关系 |
| `Linker` | `LinkHierarchy(ctx *GlobalContext) []*model.DependencyRelation` | 层级拓扑 |
| `NoiseFilter` | `SetLevel(FilterLevel)` + `IsNoise(rel) bool` | 降噪 |
| `SymbolResolver` | `BuildQualifiedName` + `RegisterPackage` + `IsPrimitive` + `ResolveType` + `ResolveFunc` + `ResolveVar` + `ResolveAction` | 符号解析 |
| `SymbolBuilder` | `BuildQualifiedName(parentQN, name) string` + `IsPrimitive(typeName) bool` | QN 构建 |

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
    })
}
```

**接入方式**：任意入口只需 `import _ "github.com/CodMac/arch-lens-dep-analyer/x/java"`，init() 自动完成注册。

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

**实现方**：`x/{lang}/ast/` 提供 `sitterNode` 适配器（包装 `*sitter.Node`），把具体节点映射到接口。

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

> 语言实现边界与能力要求，见 [languages/java.md](languages/java.md)；多语言推进优先级见 [roadmap.md](roadmap.md)。

## 7. 设计权衡

**为什么保留注册表 + init() 而非显式注册？**

| 方案 | 优点 | 缺点 |
|---|---|---|
| 全局注册表 + init() | 新语言只需 import blank；引擎零改动 | 隐式依赖；注册顺序不可控（靠 init 保证） |
| 显式注册 | 依赖可见 | 每加语言要改入口代码 |
| 编译期自注册 | 最灵活 | Go 无原生支持，复杂度高 |

选择「注册表 + init()」：Go 的 init() 保证所有依赖包先初始化，注册顺序稳定；`import _` 接入方式对使用者最友好。
