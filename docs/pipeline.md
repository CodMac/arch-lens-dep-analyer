# 五阶段流水线设计

> 适用版本：当前主干

## 1. 概述

引擎将「文件 → 关系」的处理流程抽象为**五阶段流水线**，每个阶段是一个 `Stage`，由 `engine` 编排器按序执行。阶段之间通过 `EngineContext` 传递数据。

```
+-----------+  +--------+  +----------+  +--------+  +-------------+
| Collector |→ | Binder |→ | Extractor |→ | Linker |→ | NoiseFilter |
+-----------+  +--------+  +----------+  +--------+  +-------------+
  并行收集       符号绑定       并行提取       层级拓扑       三级降噪
```

## 2. Stage 抽象

```go
// engine/stage.go
type Stage interface {
    Name() string
    // Concurrent 返回是否允许阶段内文件级并行
    Concurrent() bool
    // Execute 执行阶段逻辑
    Execute(ctx *EngineContext) error
}
```

**设计要点**：
- 新增阶段 = 实现 `Stage` 接口 + 注册到 Pipeline，纯增量。
- 替换语言实现 = 通过注册表替换 SPI，Pipeline 不变。
- 阶段可配置开关（如跳过 NoiseFilter）。

## 3. EngineContext（阶段间上下文）

```go
// engine/context.go
type EngineContext struct {
    Config      engine.Config      // Lang / RootPath / FilePaths / Concurrency / FilterLevel
    Impl        *core.LanguageImpl // 从注册表获取的语言实现（缓存）
    GC          core.GlobalContext // 全局符号上下文（接口；Base 实现内含 RWMutex）
    Relations   []*model.DependencyRelation // 已提取的关系集合
}
```

> 上下文分层（2026-11-06 接口化 / 2026-11-09 语言实现下沉）：`GC` 与各 SPI 签名中的上下文均为**接口**，通用索引/锁机制由 `core.BaseGlobalContext` / `core.BaseFileContext` 提供；语言专属实现位于 `x/{lang}/context`，engine 经 `LanguageImpl.NewGlobalContext` 创建。详见 [architecture.md](architecture.md) §3.3.1。

**生命周期**：
1. **创建**：engine 初始化时构造（`GC = impl.NewGlobalContext(impl.SymbolResolver, impl.SymbolBuilder)`，由语言上下文实现创建）。
2. **使用**：各阶段读写。
3. **结束**：流水线完成后统一清理临时字段、释放资源。

## 4. 五个阶段详述

### 阶段 1：Collector（收集符号定义）

**目标**：将每个源文件解析为 `FileContext`，内含该文件的全部代码元素定义。

**SPI**：`core.Collector`
```go
type Collector interface {
    CollectDefinitions(src ast.SourceFile) (FileContext, error)
}
```

**并发**：文件级并行。每个文件独立解析、独立产出 FileContext。

**Java 实现步骤**（x/java/collector/java_collector.go）：
1. 初始化 File 元素。
2. 处理顶层声明（package/import/class/interface/enum）。
3. 收集基本定义（嵌套类、方法、字段、变量）。
4. 细化变量作用域（ScopeBlock、Lambda 作用域）。
5. 元数据增强（enricher）。
6. 语法糖合成（desugar：Lombok/Record/Enum）。

### 阶段 2：Binder（符号绑定）

**目标**：将元素上的 raw 类型字符串（如 `List<User>`）解析为 QualifiedName（如 `java.util.List` / `com.demo.User`），建立全局符号索引。

**SPI**：`core.Binder`
```go
type Binder interface {
    BindSymbols(ctx GlobalContext) error
}
```

**并发**：串行（依赖全局索引一致性）。

**Java 实现**：遍历所有 DefinitionEntry，将 `MethodReturnType` / `MethodParameters` / `MethodThrowsTypes` / `VariableRawType` 解析为 `*WithQN` 版本，写入 Extra.Mores。

### 阶段 3：Extractor（提取关系）

**目标**：从每个文件中提取全部依赖关系，是引擎的核心与最复杂阶段。

**SPI**：`core.Extractor`
```go
type Extractor interface {
    Extract(relPath string, ctx GlobalContext) ([]*model.DependencyRelation, error)
}
```

**并发**：文件级并行（与 Collector 相同调度器）。

**Java 实现**（x/java/extractor/java_extractor.go）：
1. `extractHierarchy`：Import / Contain。
2. `extractStructural`：Extend / Implement / Annotation / Parameter / Return / Throw。
3. `discoverActionRelations`：核心动作提取（详见 action-resolution.md）——
   - 执行 JavaActionQuery 捕获动作点
   - `mapAction` 分流
   - `ResolveContext` → `Segment` → `ResolveChain`（三段式）
   - 组装 Relation
   - enricher 填 Mores、派生 Capture/TypeArg

**C/C++ 实现**（x/cpp/extractor/cpp_extractor.go）：
1. `extractHierarchy`：Import（`#include`）/ Contain。
2. `extractStructural`：Extend / Parameter / Return / TypeArg。
3. `extractActions`：AST 遍历直接按节点形态分派（`call_expression` / `new_expression` / `cast_expression` / `assignment_expression` / `init_declarator` / `field_initializer` / `throw_statement` / `identifier` / `field_expression` / `qualified_identifier`），不经 tree-sitter query。
4. `deriveCaptures`：从 Lambda 的 USE/ASSIGN 推导 Capture。

### 阶段 4：Linker（层级拓扑）

**目标**：补齐层级关系（Contain / Extend / Implement），构建全局符号树。

**C/C++ 额外职责**：`ImplLink` 声明-实现链接（函数级签名配对，定义 → 声明）。详见 [languages/cpp.md](languages/cpp.md) §4。

**SPI**：`core.Linker`
```go
type Linker interface {
    LinkHierarchy(ctx GlobalContext) []*model.DependencyRelation
}
```

**并发**：串行（依赖全局拓扑）。

### 阶段 5：NoiseFilter（降噪）

**目标**：按过滤等级移除噪音关系，得到干净关系集。

**SPI**：`core.NoiseFilter`
```go
type NoiseFilter interface {
    SetLevel(level core.FilterLevel)
    IsNoise(rel model.DependencyRelation) bool
}
```

**三级过滤**（core.FilterLevel）：

| 等级 | 语义 |
|---|---|
| `LevelRaw` | 直通，不过滤 |
| `LevelBalanced` | 保留源码/语法糖/Contain/ImplLink；外部符号的 Extend/Implement/Annotation/Parameter/Return/TypeArg/Call/Create/Cast/Use/Assign 为噪音 |
| `LevelPure` | 仅保留源码符号相关关系 |

> `ImplLink`（C/C++ 声明-实现链接）与 `Contain` 同语义，**恒保留**，不受等级影响。

## 5. 并发调度器

engine 内置并发调度器（engine/parallel.go）：

```go
// engine/parallel.go
type Scheduler struct {
    concurrency int
}

func (s *Scheduler) Run(paths []string, task func(path string) error) error {
    // channel + WaitGroup + errOnce 模式
    // 第一个错误即终止，通过 errOnce 保证只记录一次
}
```

**要点**：
- 每个 worker 独立持有语言资源（parser），避免并发争用。
- 错误只报告第一个（errOnce）。
- 阶段可复用同一调度器实例。

## 6. 临时字段清理

流水线结束时（Extractor 之后、导出之前），引擎统一清空所有 Relation 的私有临时字段：

```go
func clearTempFields(rels []*model.DependencyRelation) {
    for _, rel := range rels {
        rel.ClearTemp() // model 提供的方法：tmpNode/tmpExpr/tmpCtx 置 nil
    }
}
```

设计动机：私有临时字段是语言实现内部子阶段（extract→enricher）传递 AST 节点的通道，不属于最终输出。统一在流水线末尾置 nil，既保证 Mores 纯净，也杜绝节点泄漏。

## 7. 阶段执行顺序与数据流

| 阶段 | 输入 | 输出 | 并发 |
|---|---|---|---|
| Collector | 源文件路径列表 | FileContext 集合（注册进 GC） | 并行 |
| Binder | GC | GC（符号索引完整） | 串行 |
| Extractor | GC + 文件路径 | Relation 集合 | 并行 |
| Linker | GC | 层级 Relation 追加 | 串行 |
| NoiseFilter | 全部 Relation | 过滤后 Relation | 串行 |

## 8. 扩展点

- **新语言**：实现全部 SPI → `init()` 注册 → 流水线自动适配。
- **新阶段**：实现 Stage → 注册到 Pipeline。
- **新关系类型**：model 加常量 + 对应 enricher。
- **新过滤规则**：实现 NoiseFilter 的 IsNoise。
