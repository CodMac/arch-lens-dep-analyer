# 三段式动作解析设计

> 适用版本：当前主干

## 1. 概述

动作解析（`ResolveAction`）是引擎的**核心难点**：给定一个 AST 上的动作节点（方法调用、变量使用、赋值、对象创建、类型强转、异常抛出等）与关系类型，解析出该动作指向的目标符号（`*CodeElement`）。

解析采用**三段式**结构，AST 依赖隔离在 `x/java/resolver` 内部：

```
AST 动作节点
  → ① NodeContextResolver  一元化 Context 提取（ExpressNode + ContextNode 双轨）
  → ② ExpressionSegmenter  物理表达树 → 拉平 Head + Segments 符号链
  → ③ ChainResolver        沿链查符号库逐段流转（含继承回溯/重载打分/泛型元素传播）
  → alignElementWithRelType  动作合拍性校正（Call→Method, Use/Assign→Field/Variable...）
  → generateContextualFallback 外部符号兜底（查 import 表拼 QN）
```

## 2. 三段式详解

### 第一步：NodeContextResolver（一元化 Context 提取）

**目标**：把「关系类型 + 动作节点」统一为 `ContextResult{ExpressNode, ContextNode}` 双轨结构，消除不同 relType 的分叉处理。

**核心方法**（x/java/resolver/node_context_resolver.go）：

| 方法 | 适用 relType | 职责 |
|---|---|---|
| `resolveTwoTrackPipeline` | Use / Assign / Call | 整链化向上爬升（findCoreChain）→ 语义微调（specializeSemanticAction）→ 宏观边界（findOuterBoundary） |
| `resolveForCreate` | Create | 定位 new 表达式 |
| `resolveForThrow` | Throw | 定位被抛出的异常表达式 |
| `resolveForCast` | Cast | 定位强转表达式 |

**设计要点**：双轨（ExpressNode=表达式本身，ContextNode=语句容器）使后续 enricher 能同时拿到「动作发生处」与「动作语义容器」两类上下文。

### 第二步：ExpressionSegmenter（拉平符号链）

**目标**：将物理表达式树（`field_access`/`method_invocation`/`method_reference`/`array_access`）递归剥离为**拉平的拓扑链** `ExpressionChain{Head, Segments}`。

```go
// x/java/resolver/expression_model.go
type ExpressionHead struct {
    Type     ExpressionHeadType // HeadThis/HeadSuper/HeadIdent/HeadNewExpr/HeadCastExpr/...
    Name     string
    CastType string
    ASTNode  ast.Node
    RawText  string
}

type ExpressionSegment struct {
    Kind    SegmentKind // SegmentField/SegmentMethod/SegmentArray/SegmentClass
    Name    string
    ASTNode ast.Node
    RawText string
}

type ExpressionChain struct {
    Head     ExpressionHead
    Segments []ExpressionSegment
}
```

**确定性分流**（按 relType + AST kind）：
- `Create` 或 `object_creation_expression` → HeadNewExpr，直接返回。
- `Cast` 或 `cast_expression`/`instanceof_expression` → HeadCastExpr（含 CastType）。
- 其他 → 按 kind 递归拆段：`field_access` → Head + Field 段；`method_invocation` → Head + Method 段；`method_reference` → 用 NamedChild(0)/NamedChild(1) 位置访问。

**示例**：
```
service.getOrder(id).getTotal()
  → Head: service (Ident)
  → Segments: [Method(getOrder), Method(getTotal)]

((User) obj).getName()
  → Head: (CastExpr, CastType=User)
  → Segments: [Method(getName)]

new ArrayList<User>().add(u)
  → Head: (NewExpr, Name=ArrayList<User>)
  → Segments: [Method(add)]
```

### 第三步：ChainResolver（沿链符号推导）

**目标**：从 Head 开始，逐段沿链在全局符号库中查找到目标 `*CodeElement`，同时维护「当前类型」（currType）供下一段解析。

**核心逻辑**（x/java/resolver/chain_resolver.go）：

```go
func (cr *ChainResolver) ResolveChain(chain *ExpressionChain) *model.CodeElement {
    // 1. 处理 Head 节点（NewExpr/CastExpr/This/Super/Ident/ImplicitMethod）
    // 2. 迭代 Segments，逐段流转 currType：
    //    - SegmentClass  → 内部类/嵌套静态类
    //    - SegmentMethod → 方法调用（含重载打分、静态放宽、泛型元素传播）
    //    - SegmentField  → 字段访问（含继承链回溯）
    //    - SegmentArray  → 数组下标读取（消费泛型元素类型）
    // 3. 断链优雅降级：链中途某段解析失败 → 回退到 lastGood（最后一个成功解析的符号）
}
```

**关键子能力**（MemberResolver，x/java/resolver/member_resolver.go）：

| 能力 | 说明 |
|---|---|
| `ResolveMethod` | 收集候选（含继承链回溯 collectMethodCandidates）→ 重载打分 pickBestOverload |
| `ResolveField` | 字段/枚举常量解析，含静态上下文校验、可见性检查、继承链回溯 |
| 重载打分 | 参数数量（+100/+50）→ 类型精确匹配（+50）→ 子类/接口（+40）→ 基本类型装箱（+30） |
| 可见性 | 同包 / public / protected（子类）判定 |

**泛型元素类型传播**：
```
List<Order> list = ...;
list.get(0).getTotal();
    ↑          ↑
 genericElem=Order  在 Order 上解析 getTotal
```
- `extractGenericElementType` 从方法返回类型提取 `List<Order>` → `Order`（命中符号库才传播）。
- `isCollectionAccessor`（get/getOrDefault/getFirst/peek/remove/poll...）+ `buildCollectionAccessorFallback` 消费元素类型。

### 第四步：alignElementWithRelType（动作合拍性校正）

**目标**：校验推导出的 CodeElement 是否符合关系类型预期，必要时做语义修正。

| relType | 目标 Kind 期望 | 语义修正 |
|---|---|---|
| Call | Method | 若得 Class/Interface → 视为未注册构造的实例化，构建外部 Method |
| Use / Assign | Field / Variable / EnumConstant | 直接返回 |
| Cast / Create / Throw / Return | Class / Interface / AnonymousClass | 若得 Field/Variable → 提取声明类型；若得 Method → 提取返回值类型；完全无类型信息 → 回溯 Owner Class |

### 第五步：generateContextualFallback（外部符号兜底）

**目标**：三段式全部失败时，依据拉平链拼装最吻合的外部 QualifiedName（IsFormExternal=true）。

- 用 `headName`（Cast 取 CastType，NewExpr 取类型名）为基底。
- 查 import 表得到完整 QN（如 `List` → `java.util.List`）。
- 累加 Segments 拼路径（`$` 分隔内部类，`.` 分隔成员）。
- 按 relType 修正 Kind（Call → Method，Use/Assign → Field，其他 → Class）。

## 3. 动作捕获（JavaActionQuery）

用 tree-sitter query 全量捕获动作点，Go 逻辑做黑白名单拦截。

```go
// x/java/query/java_query.go - 捕获规则
(method_invocation)          @call_target        // Call
(method_reference)           @ref_target         // Call（方法引用）
(object_creation_expression) @create_target      // Create
(cast_expression)            @cast_target        // Cast
(instanceof_expression)      @cast_target        // Cast
(identifier)                 @id_atom            // Use（全量捞取，Go 侧黑白名单拦截）
(assignment_expression)      @assign_target      // Assign
(update_expression)          @assign_target      // Assign
(enhanced_for_statement)     @assign_target      // Assign
(variable_declarator)        @assign_target      // Assign
(throw_statement)            @throw_target       // Throw
```

**Go 侧拦截**：`isStaticUseNode` / `useFilterList` 清洗 Use 捕获点，避免过度捕获。

## 4. 设计要点

| 维度 | 设计 |
|---|---|
| AST 依赖 | resolver 仅依赖 `core/ast.Node` 接口，不直接触碰具体 AST 库 |
| 临时传递 | 节点经 Relation 私有临时字段传递，不写入 Mores |
| 模块边界 | resolver 只做符号推导，symbol_resolver 只做外部符号兜底 |
| 泛型传播 | 沿链消费泛型元素类型，支持 `List<Order>` 链式调用 |

## 5. 测试策略

- **第 2 层测试**（resolver）：`TestExpectation` + `runSymbolAsserts` 矩阵，`line:text:seq` 三元组断言目标符号。
- 覆盖：Head 全类型（This/Super/New/Cast/Ident/ImplicitMethod）、链段全类型（Field/Method/Array/Class）、泛型传播、断链降级、重载消解、继承回溯。
- e2e 黄金回归集（`x/java/testdata/e2e/baseline.golden.jsonl`）为整体行为的权威校准源。
