# Java 语言实现边界

> 适用范围：`x/java/` · 与 [qn-format.md](../qn-format.md)（QN 契约）、[language-extension.md](../language-extension.md)（SPI 契约）互为补充
>
> 本文档回答「Java 支持到什么程度」：元素/关系/语法糖覆盖清单、实现边界与已知限制。

## 1. 总体状态

Java 是当前**唯一已落地**的语言实现。全部 9 个 SPI（Parser / Collector / Binder / Extractor / Linker / NoiseFilter / SymbolResolver / SymbolBuilder / ParserFactory）均已实现并经 `x/java/init.go` 注册。行为契约由 `x/java/testdata/e2e/baseline.golden.jsonl`（2914 行 = 927 元素 + 1987 关系）锚定。

## 2. 元素覆盖（CodeElement）

| 元素 | QN 格式 | 说明 |
|---|---|---|
| Package | `包.包.包` | 点号包名逐级拆分（`com`、`com.example` 各是独立元素） |
| File | `相对路径` | `QualifiedName == Path`，反斜杠分隔（Windows） |
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

| # | 边界 | 说明 |
|---|---|---|
| 1 | 显式 `super()`/`this()` | 解析为接收者名，Target QN 为 `super`/`this`（外部 Method 符号），不指向具体构造器 |
| 2 | 重名普通元素 `$N` 消歧 | 同容器重名普通元素（字段/变量/类）追加 `$N` 的规则**代码支持但未被 e2e 基线触发**（属未验证盲区，待补测试） |
| 3 | 泛型在 QN 中 | 参数类型列表剥离泛型实参只保留原始类型；泛型实参经 TYPE_ARG 关系递归提取 |
| 4 | 文件路径分隔符 | 文件元素 QN/Path 用平台分隔符（Windows `\`），跨平台比较需注意 |
| 5 | 外部符号兜底 | 未命中符号库时按 import 表/包名升级 QN（`List` → `java.util.List`），`IsFormExternal=true` |
| 6 | 未实现语法糖 | Java 14+ 模式匹配完整语义、record 的 `equals/hashCode/toString` 合成、泛型通配符推断等不在当前覆盖 |
| 7 | 注解处理器 | 不执行编译期注解处理器；仅按源码注解文本识别 Lombok 等 |

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
| SymbolBuilder | `x/java/symbol_builder/` | QN 拼接 |

## 8. 测试现状

- 三层单测 + e2e 黄金回归集（见 [testing.md](../testing.md)）。
- e2e 基线：`x/java/testdata/e2e/baseline.golden.jsonl`（927 元素 + 1987 关系）。
- 覆盖关系类型：Call/Create/Assign/Cast/Use/Extend/Implement/Parameter/Return/Throw/Annotation/TypeArg/Capture 共 13 类。
