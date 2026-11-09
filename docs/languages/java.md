# Java 语言实现边界

> 适用范围：`x/java/` · 与 [qn-format.md](../qn-format.md)（QN 契约）、[language-extension.md](../language-extension.md)（SPI 契约）互为补充
>
> 本文档回答「Java 支持到什么程度」：元素/关系/语法糖覆盖清单、实现边界与已知限制。

## 1. 总体状态

Java 是**首个落地**的语言实现（现与 C/C++ 并存，见 [cpp.md](cpp.md)）。全部 9 个 SPI（Parser / Collector / Binder / Extractor / Linker / NoiseFilter / SymbolResolver / SymbolBuilder / ParserFactory）均已实现并经 `x/java/init.go` 注册。行为契约由 `x/java/testdata/e2e/baseline.golden.jsonl`（4412 行 = 1458 元素 + 2954 关系，LevelBalanced）锚定。

> **上下文接口化影响（2026-11-06）**：`FileContext`/`GlobalContext` 改为接口后，Java 全量实现做了机械适配（字段访问 → 方法调用），**语义不变**，e2e 基线内容不变（仅换行符噪声）。
>
> **上下文分层（2026-11-09）**：Java 专属上下文下沉到 `x/java/context`（内嵌 `core.Base*`）。`RegisterPackage` 由 `SymbolResolver` 迁入 `GlobalContext`；重载索引 kind 改由 `GlobalContextRules.OverloadKinds` 注入（Java 只收 `Method`）。**语义不变**，e2e 基线内容不变（仅换行符噪声）。

## 2. 元素覆盖（CodeElement）

| 元素 | QN 格式 | 说明 |
|---|---|---|
| Package | `包.包.包` | 点号包名逐级拆分（`com`、`com.example` 各是独立元素） |
| File | `相对路径` | `QualifiedName == Path`，正斜杠 `/` 分隔（跨平台一致） |
| Class / Interface / Enum / Annotation | `包.类型名` | 含嵌套类（`User.AddonInfo`） |
| Method / Constructor | `父QN.名(参数类型列表)` | 参数列表区分重载，泛型剥离到 `<`，变长参数保留 `...` |
| Field / EnumConstant | `父QN.字段名` | 无括号 |
| Variable | `所在作用域容器QN.变量名` | 容器可为方法/lambda/block/匿名类/构造器/增强 for |
| AnonymousClass | `容器QN.anonymousClass$N` | 名称恒为 `anonymousClass`，序号从 1 递增 |
| Lambda | `容器QN.lambda$N` | 作用域容器，形参/局部变量挂在其下 |
| ScopeBlock | `容器QN.block$N` / `类QN.$static$N` / `类QN.$instance$N` | 普通块递增编号；静态/实例初始化块固定名 |
| MethodRef | `容器QN.method_ref$N` | 名称恒为 `method_ref` |

> 各元素 QN 的完整规则与示例见 [qn-format.md](../qn-format.md) §2。

## 3. 关系覆盖（DependencyRelation）

### 3.1 结构关系（无 Location，raw 文本入 Mores）

| 关系 | 说明 |
|---|---|
| Import | 文件级导入（`raw_import_path`） |
| Contain | 层级拓扑（Linker 产出） |
| Extend | 类继承 / 接口继承 / 匿名类实现接口（`new Runnable(){}` 记 EXTEND） |
| Implement | 类实现接口 |
| Annotation | 注解引用（`java.rel.annotation.target`） |
| Parameter | 方法形参类型（`java.rel.parameter.index`） |
| Return | 方法返回类型（`java.rel.return.dimensions` / `has_type_arguments`） |
| Throw | 异常：签名声明（`is_signature`）或主动抛出（`is_runtime`/`is_rethrow`） |
| TypeArg | 泛型实参（`parent_type` / `depth` / `is_wildcard`） |

### 3.2 动作关系（带 Location，Source 由 `determinePreciseSource` 定位）

| 关系 | Target 期望 Kind | 说明 |
|---|---|---|
| Call | Method | 方法调用、构造调用、方法引用 |
| Create | Class/Interface | `new` 目标类、数组创建 |
| Cast | Class/Interface | 强转、instanceof |
| Use | Variable/Field | 标识符引用（经黑白名单清洗） |
| Assign | Variable/Field | 赋值、复合赋值、自增减、声明初始化 |
| Capture | Variable/Field | 由 USE/ASSIGN 推导（`is_capture` 标记）的跨作用域捕获 |

> 动作解析的三段式机制见 [action-resolution.md](../action-resolution.md)。

## 4. 语法糖支持（desugar，IsFormSugar=true）

| 语法糖 | 生成元素 | 示例 |
|---|---|---|
| 默认构造器 | `类名()` | `AbstractBaseEntity.AbstractBaseEntity()` |
| Record 组件字段 | `组件名` | `User.id` |
| Record 访问器 | `组件名()` | `User.id()` |
| Record 全参构造器 | `类名(类型,类型)` | `User.User(Long,String)` |
| 枚举 `values()` | `values()` | `Status.values()` |
| 枚举 `valueOf(String)` | `valueOf(String)` | `Status.valueOf(String)` |
| Lombok `@Slf4j` | 字段 `log` | `Clazz.log` |
| Lombok getter/setter | `getX()` / `setX(类型)` | `getName()`、`setName(String)` |
| Lombok Builder 内部类 | `父QN.Builder` | `ChainedCallExample.Builder` |
| Lombok Builder 链式 setter | `Builder.字段名(类型)` | `Builder.name(String)` |
| Lombok Builder `build()` | `Builder.build()` | `Builder.build()` |
| Lombok Builder 私有构造器 | `类名(Builder)` | `ChainedCallExample(Builder)` |

**Lombok 注解支持面**（testdata 覆盖）：`@Data` / `@Builder` / `@Slf4j` / `@Getter` / `@Setter` / `@ToString` / `@EqualsAndHashCode` / `@Value` / `@NoArgsConstructor` / `@AllArgsConstructor` / `@RequiredArgsConstructor`（`collector/sugar/lombok/` 下样例）。

## 5. 已支持的语言特性（语义层面）

- **方法重载消解**：参数数量 + 类型精确匹配 + 子类/接口 + 装箱，多级打分选最佳。
- **继承链回溯**：沿 `extends` / `implements` 向上递归查找目标符号。
- **上下文感知解析**：按 receiver（`this`/`super`/变量实例）自动切换搜索容器。
- **泛型元素类型传播**：`List<Order>` 链式调用在 `Order` 上解析。
- **闭包捕获**：Lambda/匿名类对外部变量的捕获（`capture_depth` / `is_effectively_final`）。
- **流式链调用**：链式调用逐段解析与 Source 定位。
- **作用域细化**：变量重定位到最近 block（`block$N.item`）、嵌套块链式编号。
- **圈复杂度/LOC 度量**：`java.method.metrics.complexity`、`java.file.metrics.loc`。

## 6. 已知边界与限制

### 6.1 通用边界

| # | 边界 | 说明 |
|---|---|---|
| 1 | 显式 `super()`/`this()` | 解析为接收者名，Target QN 为 `super`/`this`（外部 Method 符号），不指向具体构造器 |
| 2 | 重名普通元素 `$N` 消歧 | 同容器重名普通元素（字段/变量/类）追加 `$N` 的规则**代码支持但未被 e2e 基线触发**（属未验证盲区，待补测试） |
| 3 | 泛型在 QN 中 | 参数类型列表剥离泛型实参只保留原始类型；泛型实参经 TYPE_ARG 关系递归提取 |
| 4 | 文件路径分隔符 | 文件元素 QN/Path 统一为**正斜杠** `/` 相对项目根（`engine/stage.go` 的 `normalizeFilePath` 负责相对化 + `filepath.ToSlash`），跨平台一致；仅 CLI 扫描输入路径可用平台分隔符 |
| 5 | 外部符号兜底 | 未命中源码符号库时由 `SymbolResolver` 兜底：基本类型产 PRIMITIVE 元素；泛型类型变量（单大写字母）/通配符 `?` 不产外部；其余按 import 表升级完整 QN 并分配包归属（无 import 归 `<external>`）。不补 `java.lang.*` 前缀，隐式 java.lang 符号 QN 保留裸名。详见 6.2 |
| 6 | 未实现语法糖 | Java 14+ 模式匹配完整语义、泛型通配符推断未覆盖；record 的 `equals/hashCode/toString` 合成**已实现**（`desugar.go:195-230`）；`sealed`/`permits` 等新语法未覆盖 |
| 7 | 注解处理器 | 不执行编译期注解处理器；仅按源码注解文本识别 Lombok 等 |

### 6.2 外部符号兜底行为（2026-11-05 重评估）

外部符号兜底由 `SymbolResolver`（`java_symbol_resolver.go:63-93` `ResolveType`、`649-702` `generateContextualFallback`）统一处理，行为分四类：

1. **基本类型** → 产 `PRIMITIVE` 元素（`int`/`long`/`boolean`/…），QN 保留原名。
2. **泛型类型变量**（单大写字母 `T`/`E`/`S`/`K`/`V`）与**通配符 `?`** → 不产外部元素（非真实类型依赖）。
3. **有 import** → 升级为完整 QN（`List` → `java.util.List`），`PackageName` = import 包前缀。
4. **无 import**（含同包符号、隐式 java.lang）→ QN 保留原文（裸名），`PackageName` 归 `<external>`，不补 `java.lang.*` 前缀。

以下 5 类缺陷/现象均来自 spring-master 50 个最复杂文件的逐文件审计（LevelRaw 全量抽取值核对）。

| # | 缺陷模式 | 现象 | 案例 |
|---|---|---|---|
| E1 | 方法引用目标裸名（最系统） | 方法引用 `A::b` 的目标 QN 退化为裸 `b()`，仅 `PackageName` 保留接收者类型原文 | 6.2.1 |
| E2 | 文本块字面量接收者污染 | 文本块 `"""...""".formatted()` 的接收者取整段字面量，目标 QN 含换行/缩进 | 6.2.2 |
| E3 | 泛型参数逗号截断 | 带泛型 receiver 的方法调用，实参含嵌套逗号时按逗号错位分割，QN 混入 `>` 截断垃圾 | 6.2.3 |
| E4 | 外部接口嵌套类型构造器裸名 | 外部接口的嵌套类型 `new Entry()` 解析为裸 `Entry()`，丢失外层接口 QN | 6.2.4 |
| E5 | 链式外层断链 | 链式调用外层方法 receiver 为内层调用表达式，目标 QN 退化为 `内层方法.外层方法()` | 6.2.5 |

#### 6.2.1 方法引用目标裸名（E1）

方法引用（静态方法/实例方法/构造器引用）目标解析分三种情形（2026-11-05 重评估后）：

1. **命中源码符号库**（同文件/同包类在源码树中）→ 解析为完整 QN（如 `C8FullProbe::matches` → `com.example.rel.C8FullProbe.matches()`、`MergedAnnotation::isPresent` → `com.example.rel.MergedAnnotation.isPresent()`）。
2. **外部类且有显式 import** → 按 import 升级完整 QN（如 `acceptedMediaType::isCompatibleWith` → `org.springframework.util.MimeType.isCompatibleWith(MimeType)`、`AnnotationUtils::invokeAnnotationMethod` → `org.springframework.core.annotation.AnnotationUtils.invokeAnnotationMethod(Method,Object)`）。
3. **同包但无显式 import 的外部类** → 兜底生成**裸名**，`PackageName=<external>`，无法与同名/重载方法区分。

裸名案例（spring-master 已审计文件，行号为源码）：

- `spring-tx/.../reactive/AbstractReactiveTransactionManager.java:395` `TransactionSynchronization::suspend` → 目标 QN 仅 `suspend()`（`TransactionSynchronization` 与所在文件同包 `org.springframework.transaction`，无 import）
- 同型：同包无 import 的父类/兄弟类方法引用均落此路径

> 注：方法引用**构造器引用**（`Inner::new`）与同文件类型方法引用通常能命中源码库，解析为完整 QN，不属缺陷。

#### 6.2.2 文本块字面量接收者污染（E2）

Java 15 文本块 `"""..."""` 作为字符串字面量被当作调用接收者，目标 QN 取整段字面量原文（含换行/制表符），`PackageName` 同样被污染；与相邻的 `IllegalArgumentException()` 构造调用并列产出两条 CALL。

案例：

```java
// spring-context/.../cache/interceptor/CacheAspectSupport.java:705-708
throw new IllegalArgumentException("""
        Null key returned for cache operation [%s]. If you are using named parameters, \
        ensure that the compiler uses the '-parameters' flag."""
        .formatted(context.metadata.operation));
```

产出：目标 QN `"""...""".formatted()`，`PackageName` 为整段文本块；同时 L705 正常产出 `IllegalArgumentException()` 构造 CALL。同型案例：`AnnotationTypeMapping.java:325`、`ConfigurationClassPostProcessor.java:694`。

#### 6.2.3 泛型参数逗号截断（E3）

带泛型 receiver 的方法调用，当类型实参含**嵌套逗号**时按逗号分割错位，receiver 类型在 `>` 处截断，目标 QN 混入 `>` 与后续实参的残片。

案例：

```java
// spring-context/.../cache/interceptor/CacheAspectSupport.java:719,730
private final MultiValueMap<Class<? extends CacheOperation>, CacheOperationContext> contexts;
...
this.contexts.add(op.getClass(), getOperationContext(op, method, args, target, targetClass));
```

`contexts` 类型实参 `Class<? extends CacheOperation>` 内嵌套逗号（`? extends CacheOperation` 与下一实参分隔），解析错位后目标 QN 为 `CacheOperation>, CacheOperationContext.add()`，`PackageName` = `CacheOperation>, CacheOperationContext`。

#### 6.2.4 外部接口嵌套类型构造器裸名（E4）

外部接口的嵌套类型即使**在源码树内**，若仅经 import 引入外层接口（而非嵌套类型本身），构造器调用仍解析为裸类名，丢失外层接口 QN。

案例：

```java
// spring-context/.../context/annotation/ConfigurationClassParser.java:52,839
import org.springframework.context.annotation.DeferredImportSelector.Group;   // 只 import Group
...
this.imports.add(new Entry(metadata, importClassName));   // Entry = Group.Entry，未 import 嵌套类型
```

`Entry` 是 `DeferredImportSelector.Group` 接口的嵌套类（spring-master 源码树内 `DeferredImportSelector.java`），但文件只 `import Group`，`new Entry()` 解析为裸 `Entry()`，无 `DeferredImportSelector.Group` 前缀。

#### 6.2.5 链式外层断链（E5）

链式调用中，外层方法的 receiver 是内层调用表达式（如 `StringBuilder.append(...).append(...)`），外层目标 QN 退化为 `内层方法.外层方法()` 拼接，无法解析到真实方法。

案例：

```java
// spring-web/.../web/util/HierarchicalUriComponents.java:478,483,489,501,504（toUriString() 内）
uriBuilder.append(getScheme()).append(':');
uriBuilder.append(this.userInfo).append('@');
```

外层 `.append(':')` 的 receiver 为内层表达式 `uriBuilder.append(getScheme())`，目标 QN = `StringBuilder.append.append()`，`PackageName` = `StringBuilder.append`（接收者类型未沿链式返回类型解析）。

### 6.3 合理裸名（非缺陷）

以下裸名是**正确**解析结果，不属于缺陷：

- **隐式 java.lang 构造器**：`IllegalStateException()` / `IllegalArgumentException()` / `StringBuilder()` / `String()` / `Thread()` / `Object()`（QN 保留裸名即正确，不补 `java.lang.` 前缀）
- **外部父类继承方法**：内部类继承外部父类后调用父类方法，如 `DataBufferUtils` 的 `request(1)` = `BaseSubscriber.request(long)`、`ConcurrentReferenceHashMap.Segment` 的 `lock()/unlock()` = `ReentrantLock`、`enqueue()` = `ReferenceQueue`（判断裸名前须先确认类继承边界）
- **`super()` 构造器**：解析为接收者名 `super`
- **`getClass()` / `toString()`**：Object 方法
- **throws 子句宽泛异常**：`Throwable` / `Exception`（接口契约声明，非具体抛出）

## 7. 与 SPI 契约的对应

| SPI | Java 实现 | 说明 |
|---|---|---|
| Parser / ParserFactory | `x/java/java_parser.go` | 包装 tree-sitter-java |
| Collector | `x/java/collector/` | 定义收集 + 作用域细化 + 元数据增强 + desugar |
| Binder | `x/java/binder/` | raw 类型 → QN |
| Extractor | `x/java/extractor/` | 结构 + 动作 + Capture 关系提取 |
| Linker | `x/java/linker/` | Contain 拓扑 |
| NoiseFilter | `x/java/noise_filter/` | 三级降噪 |
| SymbolResolver | `x/java/symbol_resolver/` | 三段式动作解析 + 类型解析 |
| SymbolBuilder | `x/java/symbol_builder/` | QN 拼接（点号） |
| FileContext | `x/java/context/` | 内嵌 `core.BaseFileContext`（无命名空间能力） |
| GlobalContext | `x/java/context/` | 内嵌 `core.BaseGlobalContext`；`RegisterPackage` 按 `.` 注册 PACKAGE；重载索引只收 `Method` |

> SPI 签名中的上下文类型为接口（`core.FileContext` / `core.GlobalContext`），通用索引与锁由 `core.Base*` 提供，Java 专属语义在 `x/java/context` 中实现。

## 8. 测试现状

- 三层单测 + e2e 黄金回归集（见 [testing.md](../testing.md)）。
- e2e 基线：`x/java/testdata/e2e/baseline.golden.jsonl`（1458 元素 + 2954 关系，LevelBalanced）。
- 覆盖关系类型：Call/Create/Assign/Cast/Use/Extend/Implement/Parameter/Return/Throw/Annotation/TypeArg/Capture 共 13 类。
