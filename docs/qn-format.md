# QN（QualifiedName）格式规范与边界

> 适用范围：Java 语言实现（`x/java/`）· 与 [languages/java.md](languages/java.md) 互为补充
>
> QN 是 arch-lens-dep-analyer 中代码实体的**全局唯一标识**，贯穿 Collector（定义）、Binder/Resolver（解析）、Extractor（关系 Source/Target）、Linker（Contain 拓扑）、Exporter（JSONL）全流程。本文档罗列各类 Element 提取出的 QN 格式、拼接规则与边界情况，作为行为契约与测试校准依据。

## 1. 拼接基础

QN 由 `SymbolBuilder.BuildQualifiedName(parentQN, name)` 拼接（`x/java/symbol_builder/java_symbol_builder.go`）：

```go
func (b *SymbolBuilder) BuildQualifiedName(parentQN, name string) string {
    if parentQN == "" || parentQN == "." {
        return name
    }
    return parentQN + "." + name
}
```

- 父 QN 为空（`""` 或 `"."`）时直接返回 `name`（顶层元素）。
- 否则用 **`.`**（点号）连接，**不使用 `$`**（`$` 仅用于匿名/重名消歧，见 §4）。
- QN 中的括号是**方法参数签名**，属于 QN 的一部分（如 `main(String[])`），不是调用标记。

### 1.1 命名层级

```
包.包.包 ... 类[.内部类]... .方法(参数) | 字段 | 变量 | 匿名类 | lambda | block | ...
```

```
com.example.base.ScopeTest.test().lambda$1.a
└─包───────┘└─类─────┘└方法─┘└lambda消歧┘└变量
```

## 2. 各 Element 的 QN 格式

> 所有示例均取自 e2e 黄金基线 `x/java/testdata/e2e/baseline.golden.jsonl`（实际输出）。

### 2.1 Package（包）

**格式**：逐级包名，用 `.` 连接。

| 场景 | 示例 |
|---|---|
| 顶层包 | `com` |
| 嵌套包 | `com.example`、`com.example.base` |

- 由 `SymbolResolver.RegisterPackage` 将点号包名**逐级拆分为包元素**（`com`、`com.example`、`com.example.base` 各是一个独立元素）。
- `IsFormSource=true`，`Path=""`。

### 2.2 File（文件）

**格式**：相对项目根的路径，**反斜杠分隔**（Windows）。

| 场景 | 示例 |
|---|---|
| 常规文件 | `MyClass.java` |
| 子目录文件 | `collector\\sugar\\RecordTest.java` |

- `QualifiedName == Path`，即文件元素本身以文件路径为 QN。
- `Kind=FILE`，`IsFormSource=true`。

### 2.3 Class / Interface / Enum / Annotation（类型声明）

**格式**：`包名.类型名`。

| 场景 | 示例 |
|---|---|
| 顶层类 | `com.example.base.OuterClass` |
| 嵌套类（静态内部类） | `com.example.base.User.AddonInfo` |
| 接口 | `com.example.base.MyInterface` |
| 枚举 | `com.example.app.Status` |
| 注解类型 | `com.example.app.MyClassAnnotation1` |

### 2.4 Method（方法）与 Constructor（构造器）

**格式**：`父QN.方法名(参数类型列表)`。

- **方法**：参数类型列表是 QN 的一部分（`main(String[])`），用于区分重载。
- **构造器**：方法名为类名，同样带参数列表（`User.User(String)`、`Main.Main()`）。

| 场景 | 示例 |
|---|---|
| 无参方法 | `com.example.Main.test()` |
| 有参方法 | `com.example.Main.main(String[])`、`com.example.app.MyClass.run(int)` |
| 多参数 | `com.example.base.ConfigService.updateConfigs(String[],Object...)` |
| 无参构造器 | `com.example.Main.Main()` |
| 有参构造器 | `com.example.app.MyClass.MyClass(String)` |
| 内部类构造器 | `com.example.app.MyClass.MyInnerClass.MyInnerClass()` |
| 注解类型元素方法 | `com.example.app.MyClassAnnotation1.value()` |
| 枚举隐式方法（desugar） | `com.example.app.Status.values()`、`com.example.app.Status.valueOf(String)` |
| 默认构造器（desugar） | `com.example.base.AbstractBaseEntity.AbstractBaseEntity()` |

**参数类型列表规则**（`extractParameterTypesOnly`）：
- 仅保留**参数类型**，不含参数名。
- 泛型参数**只取原始类型**（`List<String>` → `List`），截断到 `<`。
- 变长参数保留 `...`（`updateConfigs(String[],Object...)`）。
- 无参数 → `()`。

### 2.5 Field（字段）与 EnumConstant（枚举常量）

**格式**：`父QN.字段名`（无括号）。

| 场景 | 示例 |
|---|---|
| 实例字段 | `com.example.base.User.AddonInfo.tableName` |
| record 组件字段（desugar） | `com.example.sugar.User.id`、`com.example.sugar.User.name` |
| 枚举常量 | `com.example.app.Status.ACTIVE`、`com.example.sugar.Color.RED` |
| 内部类内枚举常量 | `com.example.base.test.AnonymousClassTest.Inner.Color.RED` |

### 2.6 Variable（局部变量/形参）

**格式**：`所在作用域容器QN.变量名`。

作用域容器可以是方法、lambda、scope block、匿名类方法、构造器、增强 for 等。

| 场景 | 示例 |
|---|---|
| 方法内局部变量 | `com.example.Main.main(String[]).args`、`com.example.app.MyClass.run(int).times` |
| 构造器内形参 | `com.example.app.MyClass.MyClass(String).name` |
| 方法形参 | `com.example.base.MethodTest.exec(String).s` |
| lambda 参数 | `com.example.base.ScopeTest.test().lambda$1.a` |
| lambda 内变量 | `com.example.rel.CallRelationSuite.executeAll().lambda$1.s` |
| block 内变量 | `com.example.base.ScopeTest.test().block$1.x` |
| 匿名类方法参数 | `com.example.base.test.AnonymousClassTest.run().anonymousClass$1.compareTo(String).o` |
| 泛型方法参数 | `com.example.base.MethodTest.exec(int,String).i` |
| 增强 for 变量 | `com.example.base.test.ScopeVariableTest.test().block$4.item` |

### 2.7 AnonymousClass（匿名内部类）

**格式**：`所在方法QN.anonymousClass$序号`。

| 场景 | 示例 |
|---|---|
| 方法内匿名类 | `com.example.base.CallbackManager.register().anonymousClass$1` |
| 匿名类方法 | `com.example.base.CallbackManager.register().anonymousClass$1.run()` |
| 匿名类重载方法 | `com.example.base.test.AnonymousClassTest.run().anonymousClass$1.compareTo(String)` |

- 名称恒为 `anonymousClass`，序号从 **1** 开始，按出现顺序递增（同一作用域容器内）。
- 匿名类是作用域容器，其成员（方法、字段、变量）挂在 `...anonymousClass$N` 之下。

### 2.8 Lambda（Lambda 表达式）

**格式**：`所在容器QN.lambda$序号`。

| 场景 | 示例 |
|---|---|
| 方法内 lambda | `com.example.base.ScopeTest.test().lambda$1` |
| lambda 内变量 | `com.example.base.ScopeTest.test().lambda$1.a` |
| lambda 参数 | `com.example.base.ScopeTest.test().lambda$2.p1` |

- 名称恒为 `lambda`，序号从 **1** 开始（同一作用域容器内）。
- Lambda 是作用域容器，其形参/局部变量挂在 `...lambda$N` 之下。
- 注意：lambda 的**序号作用域独立于**匿名类/block（各自独立计数）。

### 2.9 ScopeBlock（代码块）

**格式**：`所在容器QN.block$序号` 或 `$static` / `$instance`。

| 场景 | 示例 |
|---|---|
| 普通代码块 | `com.example.base.ScopeTest.test().block$1`、`block$2` |
| 嵌套 block | `com.example.metrics.ComplexityTest.medium(int).block$1.block$1.i`（嵌套块各自编号） |
| try/catch 块 | `com.example.metrics.ComplexityTest.withTryCatch().block$2.e` |
| 静态初始化块 | `com.example.base.OuterClass.$static$1`、`com.example.rel.AssignRelationSuite.$static$1` |
| 实例初始化块 | `com.example.base.OuterClass.$instance$1` |

- 普通 block 名称恒为 `block`，序号从 **1** 开始（同一容器内）。
- `$static` 对应 `static_initializer`，`$instance` 对应类体内无名的 `class_body` 顶层块。
- 嵌套 block 会形成**链式编号**：`block$1.block$1`（外层第 1 个块的内部第 1 个块）。
- 变量重定位：`refineVariableScopes` 会把 for/try/if/catch 等逻辑容器中的变量**重新归属**到最近的 block 之下（`block$4.item`）。

### 2.10 MethodRef（方法引用）

**格式**：`所在容器QN.method_ref$序号`。

| 场景 | 示例 |
|---|---|
| 方法引用 | `com.example.sugar.MethodRefTest.testAllMethodReferences().method_ref$1` ~ `method_ref$6` |

- 名称恒为 `method_ref`，序号从 **1** 开始（同一容器内）。

### 2.11 语法糖生成元素（IsFormSugar=true）

由 desugar 生成，QN 遵循"宿主 QN + 对应格式"。

| 语法糖 | 生成的元素 | 示例 |
|---|---|---|
| 默认构造器 | `类名()` | `com.example.base.AbstractBaseEntity.AbstractBaseEntity()` |
| Record 组件字段 | `组件名` | `com.example.sugar.User.id` |
| Record 访问器 | `组件名()` | `com.example.sugar.User.id()` |
| Record 全参构造器 | `类名(类型,类型)` | `com.example.sugar.User.User(Long,String)` |
| 枚举 `values()` | `values()` | `com.example.app.Status.values()` |
| 枚举 `valueOf(String)` | `valueOf(String)` | `com.example.app.Status.valueOf(String)` |
| Lombok `@Slf4j` | 字段 `log` | `...Clazz.log` |
| Lombok getter/setter | `getX()` / `setX(类型)` | `com.example.lombok.DataComprehensive.getName()`、`setName(String)` |
| Lombok Builder 内部类 | `父QN.Builder` | `com.example.chained.ChainedCallExample.Builder` |
| Lombok Builder 链式 setter | `Builder.字段名(类型)` | `com.example.chained.ChainedCallExample.Builder.name(String)` |
| Lombok Builder `build()` | `Builder.build()` | `com.example.chained.ChainedCallExample.Builder.build()` |
| Lombok Builder 私有构造器 | `类名(Builder)` | `com.example.chained.ChainedCallExample.ChainedCallExample(Builder)` |

**Lombok Builder 内部类层级**：Builder 是宿主类的**内部类**（`Class.Builder`），其方法挂在 Builder 之下，方法参数中的字段类型作为 QN 一部分：

```
com.example.chained.ChainedCallExample.Builder.name(String)
com.example.chained.ChainedCallExample.Builder.build()
com.example.chained.ChainedCallExample.Builder.age(int).age   ← Builder 方法内变量
```

## 3. 关系中的 QN 语义

### 3.1 Source QN

动作关系的 Source 由 `determinePreciseSource` 向上定位（`java_extractor.go:306`）：

| 定位到的容器 kind | Source QN 形态 |
|---|---|
| method_declaration / constructor_declaration | `类QN.方法(参数)` |
| static_initializer | `类QN.$static$N` |
| lambda_expression | `容器QN.lambda$N` |
| field_declaration | `类QN.字段名` |

- 定位规则：从触发节点向上找最近的容器节点，按**行号 + kind** 在文件定义中匹配（同一容器起始行唯一）。
- 匿名类内部的动作关系，Source 会落在匿名类的方法上（如 `anonymousClass$1.run()`）。

### 3.2 Target QN

动作关系的 Target 由 `SymbolResolver.ResolveAction` 三段式解析得出：
1. **NodeContextResolver** 提取双轨上下文（接收者/符号）。
2. **ExpressionSegmenter** 拉平符号链（`a.b().c` → Head `a` + Segments `b`、`c`）。
3. **ChainResolver** 沿链推导目标元素。

| 关系类型 | Target 期望 kind | Target QN 来源 |
|---|---|---|
| CALL | Method | 方法 QN（`类QN.方法(参数)`）或外部方法符号 |
| CREATE | Class/Interface | 类型 QN（`new` 目标类） |
| CAST | Class/Interface | 类型 QN（强转目标类型） |
| USE | Variable/Field | 变量/字段 QN（`容器QN.名`） |
| ASSIGN | Variable/Field | 变量/字段 QN |
| THROW | Class | 异常类型 QN |
| PARAMETER/RETURN/TYPE_ARG | Class/Interface | 类型 QN |

**Target 的两种来源**：
- **源码符号**（`IsFormSource=true`）：命中符号库中的定义，QN 与定义一致。
- **外部符号**（`IsFormExternal=true`）：未命中时按 import 表/包名升级 QN 的兜底符号（见 §5.4）。

### 3.3 结构关系的 Source/Target

结构关系（Extend/Implement/Annotation/Parameter/Return/Throw/TypeArg）直接以**定义元素本身**为 Source/Target，QN 与定义 QN 完全一致；无 Location（L0）。

## 4. 消歧：`$N` 序号后缀

`applyUniqueQN`（`java_collector.go:366`）负责同容器内同名元素的 QN 消歧：

### 4.1 强制序号（匿名/作用域类元素）

以下元素**名称固定**，每次出现都递增序号，**总是带 `$N`**：

| 元素 | 固定名 | 序号作用域 |
|---|---|---|
| AnonymousClass | `anonymousClass` | 所在作用域容器内 |
| Lambda | `lambda` | 所在作用域容器内 |
| ScopeBlock | `block` | 所在作用域容器内 |
| MethodRef | `method_ref` | 所在作用域容器内 |
| 静态初始化块 | `$static` | 类内 |
| 实例初始化块 | `$instance` | 类内 |

```
test().lambda$1   test().lambda$2   test().lambda$3 ...
```

### 4.2 条件序号（重名消歧）

普通元素（字段、变量、类等）**只在同容器内重名时**追加 `$N`（`applyUniqueQN` 中 `occurrences[identity] > 1` 时触发）：

- 首个出现**不带** `$N`，从第 2 个开始 `$2`、`$3`…
- **方法/构造器不会触发条件序号**：因为参数列表已区分重载，`identity` 含参数签名，同容器内方法签名唯一。
- 序号计数**每个作用域容器独立**（`childOccurrences` 在进入容器时重置，见 `collectBasicDefinitions`）。

> ✅ **已覆盖**：`collector/dup/DupNameTest.java`（S10 盲区用例）真实触发该规则并断言——同容器内嵌套类与同名字段（`DupNameTest.Inner` 与 `DupNameTest.Inner$2`、`DupNameTest.Config` 与 `DupNameTest.Config$2`）。不同容器（`block$1.dup` / `block$2.dup`）各自独立计数，不触发。断言见 `TestCollectorS10DedupBlindspot`。

## 5. 边界情况

### 5.1 构造器 vs 方法

构造器 QN 的**方法名即类名**：

```
class User {
    User(String name) {}      →  com.example.User.User(String)
    String getName() {}       →  com.example.User.getName()
}
```

注意 `User.User(String)`（构造器）与 `User.User`（类自身）是**不同 QN**——类 QN 无括号，构造器 QN 带括号。

### 5.2 泛型参数在 QN 中

参数类型列表**剥离泛型实参**，只保留原始类型：

```java
void process(List<String> items)   →  Class.process(List)   // 不是 process(List<String>)
```

但**字段/变量的原始类型**保留在 Mores（`FieldRawType`/`VariableRawType`）中，TYPE_ARG 关系从这些原始类型中递归提取泛型实参（`List<String>` → TYPE_ARG `-> String`）。

### 5.3 变长参数与数组

- 变长参数：`void log(String, Object...)` → `log(String,Object...)`。
- 数组：`String[]` → `String[]`；`int[][]` → `int[][]`。
- 方法参数中的数组类型**作为参数类型一部分**保留（`main(String[])`）。

### 5.4 外部符号兜底 QN

未在符号库命中时（如 JDK 类、未导入类型），`ResolveType`/`generateContextualFallback` 按以下优先级拼装 QN：

1. **import 表精确升级**：`List` → `java.util.List`（`fc.Imports[base]`）。
2. **包内查找**：`包名.符号`。
3. **通配符导入**：`java.util.*` → `java.util.<符号>`。
4. **裸符号**：原样保留（`Runnable`、`String`）。

外部符号 `IsFormExternal=true`，`Kind` 由关系类型决定（CALL→Method，USE→Field，其余→Class）。

**已知边界**：显式 `super()`/`this()` 构造调用，解析为接收者名，Target QN 为 `super` / `this`（外部 Method 符号）。

### 5.5 匿名类实现接口分类

`new Runnable(){}` 匿名类：
- 元素 QN：`方法QN.anonymousClass$1`。
- 结构关系：**EXTEND** `-> Runnable`（不是 IMPLEMENT，见 `extractStructural` 对 AnonymousClass 的处理）。
- 同时产生 CREATE `-> Runnable` 与 CALL `-> Runnable()`（构造调用）。

### 5.6 作用域容器与变量归属

- 方法、lambda、block、匿名类都是**作用域容器**，变量挂在其下。
- `refineVariableScopes` 会把 for/try/catch/if 内的变量**重定位**到最近的 block：
  - `for(String s : list)` → `block$N.s`
  - `try(...)`/`catch(...)` → `block$N.e`
- 嵌套块链式编号：`block$1.block$1`。
- 局部类（方法内 class）QN：`方法QN.局部类名`，其构造器/方法挂在其下：
  - `com.example.base.OuterClass.scopeTest().LocalClass`
  - `com.example.base.OuterClass.scopeTest().LocalClass.LocalClass()`
  - `com.example.base.OuterClass.scopeTest().LocalClass.action()`

### 5.7 文件路径 QN 的反斜杠

文件元素与 `Path` 字段使用**平台分隔符**（Windows 下为 `\`）：`collector\\sugar\\RecordTest.java`。测试比较时需注意路径分隔符差异。

### 5.8 顶层元素判定（Linker）

Linker 将 `ParentQN == ""` 或 `ParentQN == 包名` 的元素视为**顶层元素**，直接挂到文件下（Contain）：`File.java -> 包.类`。方法、内部类、变量等**不会**平铺在文件下（它们的 ParentQN 是类/方法）。

## 6. QN 索引与查找

GlobalContext / FileContext 提供两类 QN 索引：

| 索引 | 键 | 说明 |
|---|---|---|
| `qualifiedNameMap`（文件内） | 精确 QN | 唯一映射 |
| `shortNameMap`（文件内） | 短名（Name） | 可多个（重载/重名） |
| `qnIndex`（全局） | 精确 QN | 唯一映射 |
| `methodNoParamQN`（全局） | 去参 QN | 重载消解用（`类QN.方法名` → 多候选） |

查找策略（`PreciseResolve`）：
1. 文件内短名 → 全局精确 QN → 包内 QN → 通配符导入 QN。
2. 方法解析先用 `methodNoParamQN`（去参 QN）收集候选，再用参数数量/类型打分选最佳重载。

## 7. 测试校准

- e2e 黄金基线（`x/java/testdata/e2e/baseline.golden.jsonl`）是 QN 行为的**权威校准源**。
- 修改 QN 生成逻辑后，须 `go test ./x/java/test -run TestE2EGolden -update` 刷新基线，并 review diff 确认符合本文档规范。
- 三层测试中的 Source/Target 断言使用**宽松匹配**（`sourceLooseMatch`/`targetLooseMatch`）：
  - Source 忽略调用括号段（`a.m()` vs `a.m`）。
  - Target 比较最后一个点段简单名（去包/括号/泛型）。
  - `super`/`this` 与构造器调用特殊放行。

## 附：QN 快速速查表

| 元素 | 格式 | 示例 |
|---|---|---|
| Package | `包` | `com.example.base` |
| File | `相对路径` | `collector\\sugar\\RecordTest.java` |
| Class/Interface/Enum/Annotation | `包.名` | `com.example.base.OuterClass` |
| Method | `包.类.方法(参数)` | `com.example.Main.main(String[])` |
| Constructor | `包.类.类(参数)` | `com.example.app.MyClass.MyClass(String)` |
| Field | `包.类.字段` | `com.example.base.User.AddonInfo.tableName` |
| EnumConstant | `包.枚举.常量` | `com.example.app.Status.ACTIVE` |
| Variable | `容器QN.变量` | `com.example.Main.main(String[]).args` |
| AnonymousClass | `容器QN.anonymousClass$N` | `...register().anonymousClass$1` |
| Lambda | `容器QN.lambda$N` | `...test().lambda$1` |
| ScopeBlock | `容器QN.block$N` | `...test().block$1` |
| StaticBlock | `类QN.$static$N` | `...OuterClass.$static$1` |
| InstanceBlock | `类QN.$instance$N` | `...OuterClass.$instance$1` |
| MethodRef | `容器QN.method_ref$N` | `...testAllMethodReferences().method_ref$1` |


