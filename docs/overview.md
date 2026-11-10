# Arch-Lens Dependency Analyzer 设计文档总览

> 文档语言：中文
> 适用版本：当前主干

## 1. 项目定位

Arch-Lens 是一个**基于 tree-sitter 的多语言静态架构依赖分析引擎**。它读取源码文件，解析出代码元素（类、方法、字段、变量等）及其之间的依赖关系（调用、创建、使用、赋值、继承、实现等），最终以 JSONL 形式输出，用于架构可视化、依赖审计与演化追踪。

```
输入: 源码文件树
  ↓ 解析
AST
  ↓ 收集
代码元素 (CodeElement)
  ↓ 绑定
类型符号 (QualifiedName 索引)
  ↓ 提取
依赖关系 (DependencyRelation)
  ↓ 链接 + 降噪
层级拓扑 / 纯净关系集
  ↓ 导出
element.jsonl / relation_{type}.jsonl
```

## 2. 目标

1. **分层清晰**：CLI 入口、引擎编排、核心 SPI、语言实现、数据模型、输出六个层次职责分明。
2. **功能解耦**：语言无关的 `core` 层不依赖任何 AST 库；语言实现通过 SPI 接入。
3. **文档完备**：本目录下多篇设计文档覆盖架构、数据模型、流水线、动作解析、测试、语言扩展。
4. **测试完善**：三层测试（collector / extractor / resolver）+ 端到端黄金回归集。

## 3. 设计原则

| 原则 | 说明 |
|---|---|
| **分层单向依赖** | 依赖方向严格自上而下（cmd → engine → core → model），禁止反向依赖 |
| **AST 隔离** | 核心层不感知具体 AST 库，通过薄封装接口访问节点 |
| **契约优先** | core 定义 SPI 契约，语言实现负责适配；接口签名只用 core/model 类型 |
| **Mores 纯净** | 语言特化元数据入 Mores；临时传递数据走 Relation 私有临时字段，不污染 Mores |
| **并发受控** | 共享上下文统一 RWMutex 保护，消除嵌套锁风险 |

## 4. 分层架构一览

```
cmd/analyzer        CLI 入口（薄层：解析参数、调用引擎）
engine/             流水线编排（五阶段 Stage 抽象 + 并发调度）
core/               核心 SPI（语言注册表 + 各阶段接口 + AST 抽象）
  ast/              AST 薄封装接口（Node/SourceFile）
model/              数据模型（CodeElement / DependencyRelation / Mores）
x/{lang}/           语言实现（collector/binder/extractor/resolver/...）
output/             导出器（jsonl）
docs/               设计文档（本目录）
```

> 注：解析器不设独立 `parser/` 目录，各语言在 `x/{lang}/` 内自持 `*ParserFactory`（如 `x/java/java_parser.go`），经 `core.ParserFactory` SPI 暴露给 engine。

## 5. 文档索引

| 文档 | 内容 |
|---|---|
| [overview.md](overview.md) | 本总览：定位、目标、原则、分层 |
| [architecture.md](architecture.md) | 分层架构详述：各层职责、依赖方向、模块清单 |
| [data-model.md](data-model.md) | 数据模型：CodeElement / DependencyRelation / Mores / Location |
| [pipeline.md](pipeline.md) | 五阶段流水线：Collector → Binder → Extractor → Linker → NoiseFilter |
| [action-resolution.md](action-resolution.md) | 三段式动作解析：Context → Segment → Chain（Java） |
| [language-extension.md](language-extension.md) | 语言扩展 SPI：注册表、接入新语言的步骤 |
| [languages/java.md](languages/java.md) | Java 语言实现边界：能力清单、语法糖支持、已知限制 |
| [languages/cpp.md](languages/cpp.md) | C/C++ 语言实现边界：元素/关系覆盖、声明-实现配对、条件编译、已知限制 |
| [roadmap.md](roadmap.md) | 语言支持路标：现状矩阵、分阶段规划、范围边界 |
| [testing.md](testing.md) | 测试体系：三层单测 + 黄金回归集（Java / C/C++） |
| [qn-format.md](qn-format.md) | QN 格式规范：各元素 QN 形态、消歧规则、边界情况（含 C/C++ `::` 章节） |

## 6. 核心概念速览

- **CodeElement**：源码中的可识别实体（类/方法/字段/变量/...），带 QualifiedName、Location、元数据。
- **DependencyRelation**：两个 CodeElement 之间的一条依赖（CALL/CREATE/USE/ASSIGN/EXTEND/...）。
- **Mores**：语言特化的元数据字典，key 为字符串常量，按 `ele_` / `rel_` 分组集中定义。
- **五阶段流水线**：Collector（并行收集）→ Binder（符号绑定）→ Extractor（并行提取）→ Linker（层级拓扑）→ NoiseFilter（三级降噪）。
- **三段式动作解析**（Java）：NodeContextResolver（上下文提取）→ ExpressionSegmenter（分段）→ ChainResolver（符号推导）。C/C++ 因无链式调用语法糖，改用「节点形态分类 + 作用域链/继承链查找」。
- **上下文接口**：`FileContext` / `GlobalContext` 为接口，`Base*` 实现提供通用索引与锁；语言实现内嵌复用。
- **声明/定义双元素**：`FUNCTION_DECL`/`METHOD_DECL` 与 `FUNCTION`/`METHOD` 同 QN 共存，`ImplLink` 连接（C/C++）。

## 7. 变更记录

| 日期 | 变更 |
|---|---|
| 2026-11-09 | C/C++ 语言落地：新增 [languages/cpp.md](languages/cpp.md)；同步架构/流水线/语言扩展/数据模型/QN/测试/路标文档 |
| 2026-11-09 | 上下文分层：语言专属 `FileContext`/`GlobalContext` 下沉 `x/{lang}/context`；包注册迁入语言 GlobalContext；重载 kind 改为策略注入；`PackageName`/`Namespaces` 与导入属性改为可选接口/Attrs |
| 2026-11-06 | 上下文接口化：`FileContext`/`GlobalContext` 改为接口 + Base 实现；新增 `FunctionDecl`/`MethodDecl`；`qnIndex` 键加 `|Kind` |
| 2026-10-20 | 文档体系完善：新增 Java 实现边界与语言路标 |
| 2026-10-16 | 建立 docs/，完成架构、数据模型、流水线等设计文档 |
