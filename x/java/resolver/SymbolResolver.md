基于你提供的 `java_collector.go` 源码，我为你整理、补充并重构了这份**官方技术架构与设计白皮书**。

在本次更新中，我们重点对以下三大核心模块进行了极致的细节补充与对齐：

1. **捕获点位与动作关系的完整映射**（将 Tree-sitter 节点、提取的 AST 关系及语义完全列出）。
2. **分段拓扑化逻辑的深度拆解**（对 `ExpressionChain`、`Head` 类型、`Segments` 以及 `skipParentheses` 进行了工程级呈现）。
3. **`FileContext` 与 `GlobalContext` 符号仓设计及 `CodeElement` 采集模型**（结合你给出的 `java_collector.go` 源码，详尽梳理了 13 种 `ElementKind` 的语法映射、重载签名、作用域修正及语法糖消解机制）。

---

# 🚀 Arch-Lens Java AST 符号解析架构白皮书

*(Arch-Lens Java AST Symbol Resolution Architecture & Design Specification)*

---

## 📖 一、 架构设计哲学与流转链条

Arch-Lens 针对 Java 的静态依赖解析，摒弃了传统的“边遍历、边猜测”的模糊匹配方式，设计了确定性的“前置符号处理 $\rightarrow$ 分段拓扑化 $\rightarrow$ 符号解析求值”流水线。

整个链路的演进过程如下所示：

```
[Tree-sitter 捕获点] (java_queries.go)
       │
       ▼
[双轨一元化过滤] (node_context_resolver.go) ──> 分流出 ExpressNode 与 ContextNode
       │
       ▼
[分段拓扑化拉平] (expression_segmenter.go) ──> 产出 ExpressionChain (Head + Segments)
       │
       ▼
[已知符号链路求值] (chain_resolver.go)     ──> 依赖 MemberResolver 顺藤摸瓜
       │
       ├─ (成功推导) ──> 判定与 DependencyType 合拍 ──> 返回精准 CodeElement
       │
       └─ (推导失败) ──> 触发高阶全局兜底 (generateContextualFallback) ──> 虚拟外部节点

```

---

## 🛠️ 二、 前置符号处理详解 (The Pre-Processing Pipeline)

在前置处理阶段，我们的核心任务是**将动态复杂的 AST 语法树，规整化、降维为适合计算机顺序求值的确定性链条**。

### 1. 捕获点位与动作映射 (`java_queries.go`)

利用 Tree-sitter DSL 查询，将所有可能产生依赖关系的 AST 叶子节点一次性捞出。以下是捕获点位、动作、Tree-sitter 节点类型及其实际 Java 语义的完整映射关系：

| 依赖动作 (`DependencyType`) | Tree-sitter 捕获节点类型 (`Node.Kind()`) | 实际 Java 语法现象示例 | 语义说明与提取目标 |
| --- | --- | --- | --- |
| **`Call`** | `method_invocation` | `obj.doSomething(param)` | 捕获方法调用，链式推导终点通常指向 Method 元素。 |
| **`Call`** | `method_reference` | `String::valueOf` | 捕获方法引用，用于 Lambda/Stream 等高阶函数场景。 |
| **`Create`** | `object_creation_expression` | `new ArrayList<>()` | 捕获对象实例化，推导目标为对应的 Class 及构造函数。 |
| **`Create`** | `array_creation_expression` | `new int[10]` | 捕获数组创建，推导目标为数组的基础或引用类型。 |
| **`Assign`** | `assignment_expression` | `this.value = input` | 捕获赋值行为，用于建立左值变量/字段与右值表达式的依赖流。 |
| **`Assign`** | `update_expression` | `count++` / `--index` | 捕获自增自减，隐式包含读取与写入的双重依赖。 |
| **`Assign`** | `variable_declarator` | `int x = 100` | 捕获变量初始化，建立声明变量与右侧初始值的关联。 |
| **`Cast`** | `cast_expression` | `(SubClass) obj` | 捕获类型强转，强制改变后续链条的 `currType` 上下文。 |
| **`Cast`** | `instanceof_expression` | `obj instanceof String` | 捕获类型检查，将其视为一种隐式的类型转换和断言。 |

---

### 2. 双轨一元化上下文提取 (`node_context_resolver.go`)

捕获的叶子节点只是“引子”，我们需要利用 `NodeContextResolver` 向上或向下爬升，拆分出**两个核心轨道（Result）**：

* **`ExpressNode` (物理表达式范围)**：向下收敛，找回这一段链条或操作的**完整物理表达式**。
* *示例*：从方法调用 `perform()`，向上爬升还原为 `((Sub) obj).perform()`。


* **`ContextNode` (宏观边界)**：向上拓扑，锚定其所在的最大语句或表达式边界。
* *示例*：锚定到整个 `assignment_expression`、`throw_statement` 或 `instanceof_expression`，用于做多轨变量依赖或图拓扑分析。



---

### 3. 分段拓扑化 (`expression_segmenter.go`)

分段拓扑化的核心任务是**将一个嵌套、复杂的 `ExpressNode`（如括号、链式调用、类型转换）彻底降维扁平化**，生成一个标准求值链条 `ExpressionChain`。

#### A. 括号穿透机制 (`skipParentheses`)

在 Java 中，括号 `(expr)` 会严重干扰 AST 树的深度层次（例如 `((A) obj).foo()`）。

* **处理逻辑**：遇到 `parenthesized_expression` 时，循环向内提取其 `expression` 子节点，直到剥离出最核心的物理表达式。

#### B. 符号链的数据结构 (`ExpressionChain`)

```go
type ExpressionChain struct {
    Head     ExpressionHead   // 链条起点（定位第一步去哪里查）
    Segments []ChainSegment   // 后续递进的求值段
}

```

#### C. 起点类型 (`Head`) 的判定与映射

`Head` 决定了整个链式推导的**初始作用域类型（`currType`）**。

| 起点类型 (`Head.Type`) | 对应 Java 语法现象 | 符号解析的核心逻辑 |
| --- | --- | --- |
| **`HeadThis`** | `this` | 从**当前类**的作用域开始查找。 |
| **`HeadSuper`** | `super` | 从**当前类的直接父类**作用域开始查找。 |
| **`HeadNewExpr`** | `new A.B()` | 直接触发**类符号检索**，获取该类的符号节点。 |
| **`HeadCastExpr`** | `((Sub) obj)` | 绕过 `obj` 原本声明类型，直接以 `CastType` 作为类型上下文。 |
| **`HeadIdent`** | 纯标识符 `obj` / `MyClass` | 依次检索：局部变量 $\rightarrow$ 实例/静态字段 $\rightarrow$ 外部 `Imports` / 同包类。 |
| **`HeadImplicitMethod`** | 隐式方法 `foo()` | 隐式调用。直接查找**当前类及其继承链**中的成员方法。 |
| **`HeadLiteral`** | `"hello"` / `100` | 映射为 Java 基础类型或 `java.lang.String`。 |

#### D. 后续分段 (`Segments`) 的类型与递进

一旦通过 `Head` 拿到了 `currType`，后续的 `Segments` 将以**管道（Pipeline）方式**依次向后推导：

* **`SegmentField`**：表示获取属性（如 `.value`）。
* *求值公式*：$\text{currType} = \text{MemberResolver.ResolveField}(\text{currType}, \text{fieldName})$。


* **`SegmentMethod`**：表示方法调用（如 `.toString()`）。
* *求值公式*：$\text{currType} = \text{MemberResolver.ResolveMethod}(\text{currType}, \text{methodName, params})$。


* **`SegmentClass`**：表示内部类或静态成员访问（如 `.InnerClass`）。
* *求值公式*：通过 `currType.QualifiedName + "$" + innerClassName` 在全局符号空间检索。


* **`SegmentArray`**：表示数组索引（如 `[0]`）。
* *求值公式*：剥离数组维度，将 `T[]` 降维为 `T`。



---

## 💾 三、 符号仓存储架构：`FileContext` 与 `GlobalContext`

Arch-Lens 内部维护着两级符号缓存。基于 `java_collector.go` 的设计，我们对其存储的 **CodeElement (ELE)** 的格式与种类进行深度拆解。

### 1. `CodeElement` (ELE) 统一实体格式

所有的 AST 声明（类、方法、变量、Lambda 等）在被收集后，都会被抽象并规整为一元化的 `CodeElement` 结构体：

```go
type CodeElement struct {
    Kind          ElementKind     // 元素种类（见下文 13 种）
    Name          string          // 简短名称（如 "userService"、"save"）
    QualifiedName string          // 唯一限定名（如 "com.demo.UserService.save(User)")
    Path          string          // 所在的源文件绝对路径
    Location      model.Location  // 节点在源文件中的行列范围
    IsFormSource  bool            // 是否来自源码（若为假，则为虚拟兜底节点或外部依赖）
    Metadata      map[string]any  // 增强元数据（如修饰符、泛型签名、注解列表等）
}

```

---

### 2. 核心收集生命周期 (The Collector Workflow)

在 `java_collector.go` 中，一个文件从原始 AST 到生成 `FileContext` 的生命周期非常清晰：

```
[源文件] ──> 注入 File 节点 (initFileElem)
            │
            ▼
        提取 Package 与 Imports (processTopLevelDeclarations)
            │
            ▼
        递归深度遍历 AST 构建基础树 (collectBasicDefinitions)
            │
            ▼
        变量特殊作用域修正 (refineVariableScopes)
            │
            ▼
        修饰符、注解等元数据增强 (enrichMetadata)
            │
            ▼
        语法糖消解 (applySyntacticSugar: Records, Lombok, Constructors)

```

---

### 3. `ElementKind`（13 种元素）与 Java AST 节点映射表

`java_collector.go` 中的 `_identifyElements` 确定了 13 种最核心的 CodeElement：

| ElementKind (ELE 种类) | Tree-sitter AST 对应节点 (`Node.Kind()`) | 命名生成规则 (`_applyUniqueQN`) | 核心处理逻辑与细节 |
| --- | --- | --- | --- |
| **`File`** | （虚拟顶级节点） | `filepath.Base(filePath)` | 文件自身节点，全路径为 `QualifiedName`。 |
| **`Class`** | `class_declaration`<br>

<br>`record_declaration` | `parentQN + "." + Name` | 支持 Java 16+ Record 声明。 |
| **`Interface`** | `interface_declaration` | `parentQN + "." + Name` | 接口声明。 |
| **`Enum`** | `enum_declaration` | `parentQN + "." + Name` | 枚举声明。 |
| **`EnumConstant`** | `enum_constant` | `parentQN + "." + Name` | 枚举常量字段。 |
| **`KAnnotation`** | `annotation_type_declaration` | `parentQN + "." + Name` | 注解类型声明本身。 |
| **`Method`** | `method_declaration`<br>

<br>`constructor_declaration`<br>

<br>`annotation_type_element_declaration` | `parentQN + "." + Name + (ParamTypes)` | 带**参数类型签名**（如 `foo(String,int)`），完美支持方法重载。对于无名构造，回溯类名。 |
| **`Field`** | `field_declaration` | `parentQN + "." + Name` | 类成员变量，支持一行多变量声明（如 `int a, b;` 拆为两个 Field）。 |
| **`Variable`** | `local_variable_declaration`<br>

<br>`formal_parameter`<br>

<br>`catch_formal_parameter`<br>

<br>`enhanced_for_statement`<br>

<br>`instanceof_expression` | `parentQN + "." + Name` | 局部变量、方法形参、Catch 异常变量、foreach 迭代变量、Java 14+ 模式匹配变量。 |
| **`Lambda`** | `lambda_expression` | `parentQN + ".lambda$N"` | 自动追加递增计数器后缀（如 `lambda$1`），防止重名。 |
| **`MethodRef`** | `method_reference` | `parentQN + ".method_ref$N"` | 方法引用节点，追加递增计数器后缀（如 `method_ref$1`）。 |
| **`AnonymousClass`** | `object_creation_expression` (含 `class_body`) | `parentQN + ".anonymousClass$N"` | 匿名内部类，追加递增计数器（如 `anonymousClass$1`）。 |
| **`ScopeBlock`** | `static_initializer`<br>

<br>`block` (类体内或独立块) | 静态块：`$static$N`<br>

<br>实例块：`$instance$N`<br>

<br>普通块：`block$N` | 专门处理静态代码块、实例初始化块以及方法体内的独立局部块。 |

---

### 4. 核心处理机制深度拆解

#### A. 方法重载唯一 QN 生成机制 (`_extractParameterTypesOnly`)

为了区分重载方法，`java_collector.go` 的 QN 构建不仅仅使用方法名，还会递归提取其**参数类型列表**：

1. 解析 `parameters` 节点的 Named 节点。
2. 通过 `_extractTypeString` 获取参数类型，丢弃泛型信息（如 `List<String>` 提取为 `List`）。
3. 处理变长参数：`spread_parameter` 会自动追加 `...`（如 `String...`）。
4. 若遇到 Lambda 的隐式/推导参数（如 `(a, b) -> ...`），则标记为 `inferred`。
5. *最终格式*：`com.demo.Calculator.add(int,int)`。

#### B. 变量作用域精准修正机制 (`refineVariableScopes`)

在 Java 中，定义在 `if`、`for`、`try-catch` 代码块中的局部变量，其物理 AST 父亲往往是方法节点，这会导致变量生命周期 QN 层次不准确。
`java_collector.go` 设计了精妙的 **Refine** 流程：

1. **向上攀爬**：调用 `_findNearestBlockParent` 寻找变量定义节点上方最近的逻辑容器（如 `enhanced_for_statement`、`catch_clause` 等）。
2. **定位 Block**：遍历该容器的子节点，找到与其平级的 `block` 节点。
3. **重新定位 QN**：通过 Location 匹配（`MatchLocation`）找到该 block 在 `FileContext` 中注册的 `ScopeBlock` 元素。
4. **修正归属**：将变量的 `ParentQN` 修正为该 block 的 QN，并重新生成子 QN（例如由 `method.x` 修正为 `method.block$1.x`）。

#### C. 语法糖消解与补全 (`applySyntacticSugar`)

为了保证依赖分析的完整性，`Collector` 在收集完毕后会利用 `DeSugar` 执行**无损语法糖补全**：

* **Records 消解**：自动为 `record` 生成所有组件属性对应的 `read()` 方法、全参构造函数（Canonical Constructor）。
* **默认构造函数生成**：若一个 Class 没有任何 `constructor_declaration`，自动为其补全一个无参默认构造函数，以便 `new Class()` 的求值器能够精准命中。
* **Lombok 支持**：识别类上的 `@Data`、`@Getter`、`@Setter` 等注解，自动在内存中为该类注入对应的 `getXxx()`、`setXxx()` 方法。
* **Enums 方法补全**：自动为所有 `enum` 注入静态的 `values()` 和 `valueOf(String)` 方法。

---

## 🎯 四、 符号解析与类型推导 (`SymbolResolver` & `ChainResolver`)

进入求值阶段，我们依靠内存中的符号空间进行高精度的流转推导。

### 1. 核心数据存储

* **`FileContext` (`core/file_context.go`)**：单文件符号仓，保存文件内定义（`Definitions`）、导入信息（`Imports`），并通过单机读写锁保障并发安全。
* **`GlobalContext` (`core/global_context.go`)**：全局符号仓，汇聚所有 `FileContext` 的定义，构建出全局的符号检索平面。

### 2. 精准链式推导流程 (`ChainResolver`)

`ChainResolver` 拿着 `ExpressionChain`，根据 `Head` 的性质获取初始的 `currType`（当前推导类）：

1. **处理 `Head**`：如果是 `HeadNewExpr`，则尝试将其与右侧的 `SegmentClass`（如内部类）进行合并收敛，并结合方法参数推断定位出最契合的**构造函数**。
2. **流转 `Segments**`：
* **遇 `SegmentClass**`：利用 `currType.QualifiedName + "$" + seg.Name` 拼接精准寻找内部类。
* **遇 `SegmentMethod**`：由 `MemberResolver` 解析出具体 Method 符号，并提取其返回值类型（`MethodReturnTypeWithQN`）作为下一步的 `currType`。
* **遇 `SegmentField**`：由 `MemberResolver` 解析出具体 Field 符号，并提取其声明类型（`VariableTypeWithQN`）作为下一步的 `currType`。


3. **阻断机制**：任何一步中途断链（返回 `nil`），说明此链条超出了源码解析范围，整个 `ResolveChain` 立即返回 `nil`，交由外层兜底。

### 3. 重载与继承求值算法 (`MemberResolver`)

解析类中的 Field 或 Method 时，不仅需要处理当前类，还必须处理复杂的 Java 特征：

* **继承链深度搜索 (`searchFieldInInheritance` / `collectMethodCandidates`)**：若当前类找不到，则会顺着其父类（`ClassSuperClass`）和接口（`ClassImplementedInterfaces`）向上递归检索。
* **精确可见性检查 (`checkVisibility`)**：基于 `public`, `protected`, `private` 和包作用域，过滤不符合可见性的候选符号。
* **重载评分匹配 (`pickBestOverload`)**：当方法存在重载时，计算最佳匹配得分：
* 参数个数相同：$+100$ 分。
* 参数类型完全一致或后缀匹配：$+50$ 分。
* 父子类/接口继承兼容：$+40$ 分。
* 装箱拆箱/基本类型向上兼容（如 `int -> long`）：$+30$ 分。



---

## 🛡️ 五、 高阶全局兜底与对齐策略 (Fallback & Alignment)

静态分析无法保证 $100\%$ 的源码都在 `GlobalContext` 中，必须有完美的容错方案。

### 1. 动作与符号的合拍性检查 (Alignment)

推导出的符号 `targetEle` 需要和当时的依赖动作 `relType` 进行合拍性校正：

* **`Call` 动作**：期望对应 `Method` 符号。如果推导出了 `Class`，则说明可能是未注册具体构造方法的实例化过程，自动修正为虚拟的外部构造方法符号（`createExternalFallbackWithParent`）。
* **`Use` / `Assign` 动作**：期望对应 `Field` 或 `Variable`。
* **宏观动作（`Create`, `Cast`, `Throw`, `Return`）**：期望对应 `Class` 或 `Interface`。如果意外定位到了变量或方法，则自动回溯获取其声明所属的 `Owner Class`。

### 2. 分段上下文保底 (`generateContextualFallback`)

若 `ChainResolver` 彻底断链或解析失败，`SymbolResolver` 不会抛出空指针，而是利用 `generateContextualFallback` 自动合成一个最吻合的虚拟外部节点：

1. **路径解析**：从 `Head` 对应的物理位置还原导入路径起点（检索 `Imports`），得到 `baseQN`。
2. **链条拼装**：顺着 `Segments` 累加名字。如果是 `SegmentClass` 拼 `$`, 其他拼 `.`。
3. **动作修饰**：如果是 `Call` 依赖，则在尾部补充一对括号 `()`。
* *示例*：对于 `((ArrayList) obj).size()`，若找不到 `ArrayList` 源码，保底生成的节点 QualifiedName 为：`java.util.ArrayList.size()`，其 Kind 为 `Method`，且 `IsFormExternal = true`。