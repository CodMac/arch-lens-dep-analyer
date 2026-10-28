# Arch-Lens-Dep-Analyer

**Arch-Lens-Dep-Analyer** 是一款基于 **tree-sitter** 的多语言静态架构依赖分析引擎。它读取源码文件，解析出代码元素（类、方法、字段、变量等）及其之间的依赖关系（调用、创建、使用、赋值、继承、实现、泛型、闭包捕获等），以 JSONL 形式输出，用于架构可视化、依赖审计与演化追踪。

设计文档位于 [docs/](docs/overview.md)，含架构、数据模型、流水线、动作解析、测试体系与语言扩展等专题。

---

## 核心能力

* **15 类依赖关系**：Contain / Import / Extend / Implement / Call / Create / Assign / Use / Capture / TypeArg / Parameter / Return / Throw / Annotation / Cast，经 `Mores` 字典提供细粒度元数据（见[数据模型速查](#依赖关系速查)）。
* **多阶段消解管线**：Collector → Binder → Extractor → Linker → NoiseFilter 五阶段，阶段间经 `EngineContext` 传递数据，Collector / Extractor 文件级并行（见 [pipeline.md](docs/pipeline.md)）。
* **三段式动作解析**：NodeContextResolver → ExpressionSegmenter → ChainResolver，从 AST 动作节点推导目标符号，含继承链回溯、重载打分、泛型元素传播与断链兜底（见 [action-resolution.md](docs/action-resolution.md)）。
* **闭包捕获**：Lambda / 匿名类对外部变量的捕获（`capture_depth`、`is_effectively_final`）。
* **语法糖合成**：Record 组件/访问器/构造器、枚举 `values()`/`valueOf()`、Lombok 系列（`@Data`/`@Builder`/`@Slf4j` 等）。
* **SPI 多语言扩展**：core 定义契约、各语言经注册表接入（见 [language-extension.md](docs/language-extension.md)）。
* **三层测试 + 黄金回归**：collector / resolver / extractor 三层单测 + e2e 黄金基线（见 [testing.md](docs/testing.md)）。

---

## 架构总览

六层架构，依赖方向严格自上而下（cmd → engine → core → model），语言无关的 `core` 层不依赖任何 AST 库，语言实现通过 SPI 接入（见 [architecture.md](docs/architecture.md)）：

```
cmd/analyzer        CLI 入口（薄层：参数解析、文件扫描、调用引擎）
engine/             流水线编排（五阶段 Stage 抽象 + 并发调度）
core/               核心 SPI（语言注册表 + 各阶段接口 + AST 抽象）
model/              数据模型（纯数据结构，零依赖）
x/{lang}/           语言实现（collector/binder/extractor/... 经 SPI 注册）
output/             导出器（jsonl）
docs/               设计文档
```

引擎将「文件 → 关系」处理流程抽象为**五阶段流水线**（见 [pipeline.md](docs/pipeline.md)）：

```
Collector → Binder → Extractor → Linker → NoiseFilter
  并行收集     符号绑定     并行提取     层级拓扑     三级降噪
```

---

## 依赖关系速查

引擎可识别并提取以下维度的依赖关系，细粒度元数据由 `Mores` 字典承载（完整规范见 [data-model.md](docs/data-model.md)）：

| 关系类型 | 说明 |
| --- | --- |
| **Contain** | 拓扑包含关系（包、文件、成员） |
| **Import** | 源码级导入依赖 |
| **Extend** | 类/接口继承、匿名类实现接口 |
| **Implement** | 类实现接口 |
| **Call** | 方法调用、构造函数、方法引用 |
| **Create** | 对象实例化、数组创建 |
| **Assign** | 变量赋值、复合赋值、自增减 |
| **Use** | 标识符引用、字段访问 |
| **Capture** | 闭包捕获：Lambda 引用外部变量 |
| **TypeArg** | 泛型参数依赖 |
| **Parameter** | 方法形参类型依赖 |
| **Return** | 方法返回值类型依赖 |
| **Throw** | 异常抛出（声明或主动抛出） |
| **Annotation** | 注解引用 |
| **Cast** | 类型转换、Instanceof 检查 |

---

## 语言支持

| 语言 | 状态 | 说明 |
| --- | --- | --- |
| Java | ✅ 已落地 | 15 类关系 + 语法糖，能力边界见 [languages/java.md](docs/languages/java.md) |
| Go | 🚧 已声明未实现 | `core` 已有 `LangGo` 常量，`x/go/` 尚未落地，推进计划见 [roadmap.md](docs/roadmap.md) |

---

## 快速开始

### 编译 (Build)

Arch-Lens-Dep-Analyer 依赖 `tree-sitter` 动态库，编译前请确保环境已配置相关依赖：

```bash
# 克隆项目
git clone https://github.com/CodMac/arch-lens-dep-analyer.git
cd arch-lens-dep-analyer

# 下载依赖
go mod download

# 编译 CLI 工具
CGO_ENABLED=1 go build -o arch-lens-dep-analyer .
```

### 基础使用 (Usage)

Arch-Lens-Dep-Analyer 提供命令行接口，支持多种输出格式和过滤等级。

#### 1. 基础 JSONL 导出

分析指定目录，并将结果导出为可供后续处理的 `jsonl` 格式：

```bash
./arch-lens-dep-analyer -path ./my-project -lang java -level 1 -out-dir ./result
```

输出：`<out-dir>/element.jsonl` 与 `<out-dir>/relation_{type}.jsonl`。

#### 2. 核心参数说明

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `-lang` | `java` | 分析的目标语言 |
| `-path` | `.` | 源码根路径 |
| `-level` | `1` | **过滤等级**：0(Raw), 1(Balanced), 2(Pure) |
| `-jobs` | CPU 核数 | 并发执行的任务数 |
| `-filter` | `.*\.<lang>$` | 文件过滤正则（默认按语言扩展名） |
| `-out-dir` | `./output` | 输出目录 |

---

## 文档索引

| 文档 | 内容 |
| --- | --- |
| [overview.md](docs/overview.md) | 设计总览：定位、目标、原则、分层 |
| [architecture.md](docs/architecture.md) | 分层架构详述：各层职责、依赖方向、模块清单 |
| [data-model.md](docs/data-model.md) | 数据模型：CodeElement / DependencyRelation / Mores / Location |
| [pipeline.md](docs/pipeline.md) | 五阶段流水线：Collector → Binder → Extractor → Linker → NoiseFilter |
| [action-resolution.md](docs/action-resolution.md) | 三段式动作解析：Context → Segment → Chain |
| [language-extension.md](docs/language-extension.md) | 语言扩展 SPI：注册表、接入新语言的步骤 |
| [languages/java.md](docs/languages/java.md) | Java 语言实现边界：能力清单、语法糖支持、已知限制 |
| [roadmap.md](docs/roadmap.md) | 语言支持路标：现状矩阵、分阶段规划、范围边界 |
| [testing.md](docs/testing.md) | 测试体系：三层单测 + 黄金回归集 |
| [qn-format.md](docs/qn-format.md) | QN 格式规范：各元素 QN 形态、消歧规则、边界情况 |

---

## 测试

三层单元测试 + 端到端黄金回归集，详见 [testing.md](docs/testing.md)：

```bash
# 全部测试
go test ./...

# 仅 Java 层
go test ./x/java/...

# Java 三层 + e2e 测试
go test ./x/java/test/...

# 更新 e2e 黄金快照
go test ./x/java/test -run TestE2EGolden -update
```

---

## 贡献与许可

* 欢迎提交 PR 增加新语言支持（如 Go, C++, Python），接入步骤见 [language-extension.md](docs/language-extension.md)。
* 如果你发现了特定的 Java 语法糖未能解析，请提交 Issue 并附带代码片段。
* 本项目基于 **MIT License** 开源。
