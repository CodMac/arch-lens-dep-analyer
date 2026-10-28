# 语言支持路标

> 适用范围：全引擎 · 与 [languages/java.md](languages/java.md)（实现边界）、[language-extension.md](language-extension.md)（SPI 契约）互为补充
>
> 本文档回答「接下来要支持什么」：语言支持现状矩阵、分阶段推进路线、明确不做的事项与路标更新机制。

## 1. 语言支持现状矩阵

| 语言 | 状态 | 说明 |
|---|---|---|
| Java | ✅ 已落地 | 9 个 SPI 全部实现并经 `x/java/init.go` 注册，行为由 e2e 黄金基线锚定 |
| Go | 🚧 已声明未实现 | `core.Language` 中已有 `LangGo` 常量，但 `x/go/` 尚不存在，调用 `core.Get("go")` 会返回未注册错误 |

> Java 的能力清单与边界见 [languages/java.md](languages/java.md)；SPI 实现要求见 [language-extension.md](language-extension.md)。

## 2. 分阶段路标

### 阶段 A：Java 边界补齐（当前阶段）

Java 已覆盖主要能力，但存在**代码支持但未验证/未完全实现**的边界，优先补齐：

| 项 | 类型 | 说明 |
|---|---|---|
| 重名普通元素 `$N` 消歧测试 | ✅ 已解决 | `applyUniqueQN` 条件序号分支已由 `collector/dup/DupNameTest.java`（S10 用例）真实触发并断言（见 [qn-format.md](qn-format.md) §4.2） |
| 显式 `super()`/`this()` 构造调用 | 实现边界 | 当前解析为接收者名（Target QN 为 `super`/`this`），评估是否指向具体构造器（见 [languages/java.md](languages/java.md) §6 #1） |
| record 合成方法 | 未实现 | record 的 `equals/hashCode/toString` 由编译器合成，当前 desugar 只生成组件字段/访问器/全参构造器，**不生成**这三个合成方法 |
| 内部类 `this$0` 字段 | 未实现 | Java 编译器为内部类合成指向外部类实例的 `this$0` 字段，当前**不建模** |
| `MethodIsDefault` 常量未设置 | 实现缺口 | `constants.MethodIsDefault`（`java.method.is_default`）已定义但**从未被任何代码设置**，接口 `default` 方法不会打该标记 |
| Java 14+ 模式匹配完整语义 | 未实现 | 完整模式匹配语义、泛型通配符推断等不在当前覆盖 |

### 阶段 B：Go 首个新语言（里程碑）

以 Go 作为第二个语言实现，验证 SPI 契约的多语言泛化能力。步骤（详见 [language-extension.md](language-extension.md) §6）：

1. 创建 `x/go/`，实现 `core/ast.Node` 适配器（包装 tree-sitter-go）。
2. 实现 9 个 SPI：collector / binder / extractor / linker / noise_filter / symbol_resolver / symbol_builder + parser。
3. `x/go/init.go` 调 `core.Register` 注册。
4. 建立 `testdata/go/`，沿用三层测试 + e2e 黄金回归模式。

**Go 特有能力点**（与 Java 差异较大，需专门设计）：

- **Interface 隐式实现**：Go 类型无需显式 `implements`，需按方法集推断实现关系。
- **Struct Embedding**：嵌入字段的方法/字段提升语义，影响继承链回溯与 QN 归属。
- **包内多文件**：同包跨文件符号解析（Java 的 import 表模型不适用）。
- **goroutine/channel**：是否需要建模并发原语之间的关系（当前 SPI 无此概念，需评估扩展）。

### 阶段 C：多语言泛化

多语言并存后的横向能力建设：

- **能力矩阵文档化**：各语言对元素/关系/语法糖的覆盖程度横向对比（扩展 [languages/java.md](languages/java.md) 为 `languages/` 下的统一格式）。
- **降噪语言化**：NoiseFilter 当前为通用过滤，可评估按语言差异化的降噪策略。
- **输出扩展**：当前仅 JSONL 导出，可评估其他消费端格式（如 DOT/CSV）——注意保持简单输出原则。

## 3. 范围边界（明确不做）

以下事项**不在当前路标内**，避免范围蔓延：

| 事项 | 原因 |
|---|---|
| 图数据库存储/查询（Neo4j 等） | 引擎只产出 JSONL 关系流，消费端由外部系统承担 |
| Diff/增量解析 | 当前按完整源码树全量分析，无增量索引需求 |
| 编译期注解处理器执行 | 仅按源码注解文本识别（Lombok 等），不跑注解处理器 |
| 完整泛型推断 | 仅支持"泛型元素类型传播"级别的定向推断（见 [action-resolution.md](action-resolution.md)） |
| 其他语言（Python/TS 等） | 在 Go 落地并验证 SPI 泛化前不扩展 |

## 4. 路标更新机制

- 路标随代码现状演进：**以实际代码为准**，README/文档中的"未勾选项"若代码已实现，以代码为准更新。
- 每完成一个阶段，回填"已落地"项并更新 [languages/java.md](languages/java.md) 与本文档。
- 新增语言时先评审 SPI 契约是否需扩展（如 Go 的 goroutine 关系），避免为单一语言打补丁。
