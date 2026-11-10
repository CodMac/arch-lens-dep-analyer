# 语言支持路标

> 适用范围：全引擎 · 与 [languages/java.md](languages/java.md)（实现边界）、[language-extension.md](language-extension.md)（SPI 契约）互为补充
>
> 本文档回答「接下来要支持什么」：语言支持现状矩阵、分阶段推进路线、明确不做的事项与路标更新机制。

## 1. 语言支持现状矩阵

| 语言 | 状态 | 说明 |
|---|---|---|
| Java | ✅ 已落地 | 9 个 SPI 全部实现并经 `x/java/init.go` 注册，行为由 e2e 黄金基线锚定 |
| C | ✅ 已落地 | 与 C++ 共用 `x/cpp/` 实现，`x/cpp/init.go` 注册为 `core.LangC`（`.c`/`.h`） |
| C++ | ✅ 已落地 | 同上，注册为 `core.LangCPP`（`.cpp`/`.hpp`/`.cc`/`.cxx`/`.h`） |
| Go | 🚧 已声明未实现 | `core.Language` 中已有 `LangGo` 常量，但 `x/go/` 尚不存在，调用 `core.Get("go")` 会返回未注册错误 |

> 各语言的能力清单与边界见 [languages/java.md](languages/java.md)、[languages/cpp.md](languages/cpp.md)；SPI 实现要求见 [language-extension.md](language-extension.md)。

### 1.1 已完成的语言扩展里程碑

| 里程碑 | 内容 |
|---|---|
| M1 上下文接口化（2026-11-06） | `FileContext`/`GlobalContext` 改为接口 + Base 实现；`qnIndex` 键加 `|Kind`；新增 `FunctionDecl`/`MethodDecl`；`methodNoParamQN` 放宽至 Function 族；Java 全量机械适配（golden 语义不变） |
| M2 C/C++ 落地（2026-11-09） | `x/cpp/` 全套 SPI + 三层测试 + e2e 黄金基线（2286 行）；C 与 C++ 共用实现，Parser 按扩展名选 grammar |
| M3 上下文分层（2026-11-09） | 语言专属上下文下沉 `x/{lang}/context`：`RegisterPackage` 从 SymbolResolver 迁入 GlobalContext；重载索引 kind 由 `GlobalContextRules` 注入；`PackageName`/`Namespaces` 下沉为可选能力接口（`core.PackageAwareFileContext` / `cppctx.NamespaceAware`）；`ImportInfo` 语言字段改为 `Attrs`；删除 `core.NewFileContext`/`core.NewGlobalContext`，改由 `LanguageImpl.NewGlobalContext` 提供（golden 语义不变） |
| M4 C/C++ 两段式解析（2026-11-10） | `x/cpp/symbol_resolver` 归一为两段式：段 1 `normalize` 把动作 AST 折叠为统一 `expr`（吸收 `field_expression`/`qualified_identifier`/`call_expression`/`new_expression`/`cast_expression` 等形态），段 2 `resolveExpr` 按 `wantValue`/`wantFunc`/`wantType` 查找；`ResolveVar`/`ResolveFunc` 公开 SPI 也走同一路径；删除 12 个分散动作函数（golden 语义不变，仅新增 chain.cpp 用例） |
| M5 C++ Collector 全量测试（2026-11-10） | 新增 `testdata/collector/cpp/` 18 组场景 + `collector_cpp_test.go`（base/namespace/decl_def/class/ctor_dtor/operator/template/inherit/enum/typedef_using/variable/function/param/lambda/scope/macro/cond/edge）；同步修复 8 个 collector 缺陷（reference_declarator 名称丢失、转换运算符、friend 误当成员、struct 继承、模板特化 QN、模板参数误产变量、默认实参进入签名、结构化绑定/range-for 变量缺失） |

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

### 阶段 B：Go 语言（里程碑）

C/C++ 已验证了「同一套 SPI 支撑两种语法相近语言」的路径（见 §1.1 M2）。Go 是**语义差异更大**的第二个独立实现，用于验证 SPI 契约的进一步泛化。步骤（详见 [language-extension.md](language-extension.md) §6）：

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

- **能力矩阵文档化**：各语言对元素/关系/语法糖的覆盖程度横向对比（`languages/` 下已按统一格式提供 `java.md` 与 `cpp.md`）。
- **降噪语言化**：NoiseFilter 当前为通用过滤（C/C++ 复用 `DefaultNoiseFilter`，仅额外恒保留 `ImplLink`），可评估按语言差异化的降噪策略。
- **输出扩展**：当前仅 JSONL 导出，可评估其他消费端格式（如 DOT/CSV）——注意保持简单输出原则。

### 阶段 D：C/C++ 边界补齐

C/C++ 首版已落地，以下为已知边界（详见 [languages/cpp.md](languages/cpp.md) §7）：

| 项 | 类型 | 说明 |
|---|---|---|
| 运算符重载调用 | 未实现 | `a + b` 不建模为 `operator+` 调用（成员声明仍收集） |
| 宏展开 | 不做 | 只收集宏名+宏体，不参与符号解析 |
| 模板实例化/特化 | 不建模 | `Box<int>`/`Box<double>` 共用 `Box`；显式特化与主模板同 QN |
| 条件编译求解 | 不做 | 全分支保留 + 分支标注，不做宏环境求解 |
| 多继承语义 | 简化 | 多个基类各产一条 EXTEND，不区分 `virtual` 继承 |
| `extern "C"` 链接名 | 不区分 | 声明照常收集 |

## 3. 范围边界（明确不做）

以下事项**不在当前路标内**，避免范围蔓延：

| 事项 | 原因 |
|---|---|
| 图数据库存储/查询（Neo4j 等） | 引擎只产出 JSONL 关系流，消费端由外部系统承担 |
| Diff/增量解析 | 当前按完整源码树全量分析，无增量索引需求 |
| 编译期注解处理器执行 | 仅按源码注解文本识别（Lombok 等），不跑注解处理器 |
| 完整泛型推断 | 仅支持"泛型元素类型传播"级别的定向推断（见 [action-resolution.md](action-resolution.md)） |
| 其他语言（Python/TS 等） | 在 Go 落地并验证 SPI 泛化前不扩展 |
| C/C++ 编译期语义（宏展开/模板实例化/重载决议） | 静态架构分析只需符号与依赖，不追求编译器级语义 |

## 4. 路标更新机制

- 路标随代码现状演进：**以实际代码为准**，README/文档中的"未勾选项"若代码已实现，以代码为准更新。
- 每完成一个阶段，回填"已落地"项并更新对应 `languages/*.md` 与本文档。
- 新增语言时先评审 SPI 契约是否需扩展（如 Go 的 goroutine 关系），避免为单一语言打补丁。
