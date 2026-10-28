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
│   第2层 Resolver：三段式符号解析断言           │
│   第3层 Collector：符号定义收集断言            │
├─────────────────────────────────────────────┤
│ testdata/  黄金回归集（源码样例 + 期望断言）    │
└─────────────────────────────────────────────┘
```

## 2. 三层测试详述

三层测试统一位于 `x/java/test/`，通过 `helper.go` 的流水线搭建辅助（`RunCollection` / `RunExtraction` / `RunPipeline`）执行。

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

## 3. 测试数据组织（黄金回归集）

```
testdata/
├── collector/       # Collector 测试样例（base/enum/class/generic/metrics/sugar/lombok/...）
├── extractor/       # Extractor 测试样例（call/assign/cast/create/throw/use/...）
└── e2e/             # 端到端黄金基线（baseline.golden.jsonl）
```

**迁移原则**：
- testdata 样例文件与期望断言作为黄金回归集完整迁移。
- 新架构测试代码重写（因为接口签名变化），但断言语义保持。

## 4. 公共测试辅助（helper.go）

```go
// x/java/test/helper.go
func RunPipeline(t *testing.T, root string, paths []string, level core.FilterLevel) *engine.EngineContext // 完整五阶段
func RunExtraction(t *testing.T, root string, paths []string) (*core.GlobalContext, []*model.DependencyRelation) // Collect+Bind+Extract
func RunCollection(t *testing.T, root string, paths []string) *core.GlobalContext // Collect

func FindDef(t *testing.T, gc *core.GlobalContext, qn string) *model.CodeElement
func FindRel(t *testing.T, rels []*model.DependencyRelation, pred func(*model.DependencyRelation) bool) *model.DependencyRelation
func RelFilter(rels []*model.DependencyRelation, typ model.DependencyType) []*model.DependencyRelation
func PrintCodeElements(t *testing.T, gc *core.GlobalContext)
func PrintRelations(t *testing.T, rels []*model.DependencyRelation)
```

## 5. 端到端测试（e2e）

**目标**：输入完整的 testdata 源码树，断言引擎完整输出（JSONL 确定性快照）。

```go
// x/java/test/e2e_test.go
func TestE2EGolden(t *testing.T) {
    // 1. 收集全部 testdata 下的 .java 文件
    // 2. 跑完整五阶段流水线（LevelBalanced）
    // 3. 元素按 QualifiedName 排序、关系按 (Type,SourceQN,TargetQN,Location) 排序，生成确定性快照
    // 4. 与 baseline.golden.jsonl 比对（-update 标志可刷新）
}
```

**快照确定性策略**：原始输出受 map 迭代顺序影响，因此生成快照前统一排序——元素按 QualifiedName 排序，关系按 `(Type, SourceQN, TargetQN, Location)` 排序，Mores 由 `encoding/json` 按键排序序列化，从而同一输入必然产出同一快照。

**golden 文件**：`x/java/testdata/e2e/baseline.golden.jsonl`，用 `-update` 标志刷新。

## 6. 测试命令

```bash
# 全部测试
go test ./...

# 仅 Java 层
go test ./x/java/...

# Java 三层 + e2e 测试
go test ./x/java/test/...

# 更新 e2e 黄金快照
go test ./x/java/test -run TestE2EGolden -update

# 覆盖率
# go test ./... -coverprofile=cover.out && go tool cover -html=cover.out
```

## 7. 验收标准

1. 黄金回归集（testdata 全部断言）通过。
2. 端到端 CLI 跑通（`go run ./cmd/analyzer` 输出 JSONL）。
3. 文档完备（docs/ 全部章节）。
