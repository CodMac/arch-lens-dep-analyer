# 数据模型设计

> 适用版本：当前主干

## 1. 设计原则

1. **纯数据结构**：model 包零依赖，不 import 任何非标准库包。
2. **三件套保留**：`CodeElement` / `DependencyRelation` / `Mores` 字典为已验证的核心资产，完整保留，仅清理字段与常量。
3. **Mores 纯净**：语言特化元数据入 Mores；**临时传递数据不入 Mores**（见 §5）。
4. **Key 集中**：Mores key 全部定义为字符串常量，语言无关 key 在 model 层，语言特化 key 在各语言 constants 包，禁止散落魔法字符串。

## 2. CodeElement（代码元素）

```go
type CodeElement struct {
    Kind           ElementKind // 元素类型：CLASS/METHOD/FIELD/VARIABLE/...
    Name           string      // 短名称："main"
    QualifiedName  string      // 限定名："com.demo.OrderService.getTotal"
    Path           string      // 源文件路径（相对项目根）
    Signature      string      // 完整签名（方法含参数与返回值）
    Location       *Location   // 源码位置
    Doc            string      // 文档注释
    Comment        string      // 普通注释
    Extra          *Extra      // 元数据
    IsFormSource   bool        // 源码产生的实体
    IsFormSugar    bool        // 语法糖产生的实体
    IsFormExternal bool        // 外部导入产生的实体
}
```

### ElementKind 分类

| 分类 | 种类 |
|---|---|
| 基本结构体 | FILE / PACKAGE / MODULE / NAMESPACE |
| 面向对象/复合类型 | CLASS / INTERFACE / ENUM / ANONYMOUS_CLASS / ANONYMOUS_STRUCT / STRUCT / TRAIT / ANNOTATION |
| 可执行体 | METHOD / FUNCTION / LAMBDA / METHOD_REF / SCOPE_BLOCK / MACRO |
| 声明（无实现体，与定义成对） | FUNCTION_DECL / METHOD_DECL |
| 存储和声明 | FIELD / VARIABLE / CONSTANT / ENUM_CONSTANT / TYPE |
| 其他 | UNKNOWN |

**声明/定义成对 kind（2026-11-06 新增）**：

| 声明 kind | 对应定义 kind | 用途 |
|---|---|---|
| `FUNCTION_DECL` | `FUNCTION` | C/C++ 头文件函数原型（无函数体） |
| `METHOD_DECL` | `METHOD` | C++ 类成员声明（无函数体） |

- 声明与定义 **QN 相同、Kind 不同**，在 `BaseGlobalContext` 中按 `QN|Kind` 共存。
- `FindByQualifiedName` 定义优先；`ImplLink` 关系从定义指向声明。
- 其他声明形态（`extern` 变量、struct 前向声明）不独立成 kind，归 `VARIABLE`/`STRUCT` 并记 `cpp.is_declaration=true`。

### Extra 元数据

```go
type Extra struct {
    Modifiers   []string               // "public"/"private"/"static"/...
    Annotations []string               // "@Service"/...
    Mores       map[string]interface{} // 语言特化键值字典
}
```

### 三形态标记

一个 CodeElement 用三个布尔字段标记来源：

| 字段 | 含义 | 示例 |
|---|---|---|
| `IsFormSource` | 源码真实声明 | `class OrderService` |
| `IsFormSugar` | 语法糖合成 | Lombok 生成的 `getX()`、Record 的 `x()` |
| `IsFormExternal` | 外部/虚拟符号 | `java.lang.String` 兜底节点 |

三者互斥且至少一个为真（用于 NoiseFilter 判定）。

## 3. DependencyRelation（依赖关系）

```go
type DependencyRelation struct {
    Type     DependencyType // 关系类型：CALL/CREATE/USE/ASSIGN/...
    Source   *CodeElement   // 发起方
    Target   *CodeElement   // 指向方
    Location *Location      // 关系发生位置
    Details  string         // 附加描述（调用签名等）
    Mores    map[string]interface{} // 关系特化元数据
}
```

### DependencyType 分组

| 分组 | 类型 |
|---|---|
| 组织与层级 | IMPORT / EXPORT / CONTAIN |
| 继承与实现 | EXTEND / IMPLEMENT / OVERRIDE |
| 类型绑定与元数据 | ANNOTATION / PARAMETER / RETURN / THROW / TYPE_ARG |
| 行为与执行流 | CALL / CREATE / CAST |
| 数据流与状态 | USE / ASSIGN / CAPTURE |
| 跨语言/特殊 | IMPL_LINK / MIXIN |

> `IMPL_LINK` 当前由 C/C++ Linker 产出（定义 → 声明，见 [languages/cpp.md](languages/cpp.md) §4）。`OVERRIDE` / `EXPORT` / `MIXIN` 常量已定义但尚无语言实现产出。

## 4. Mores 字典与 Key 常量

Mores 是 `map[string]interface{}`，key 按语言无关 / 语言特化分层定义：

| 定义位置 | 前缀 | 用途 | 示例 |
|---|---|---|---|
| `model/mores.go` | `ele.` / `rel.` | 语言无关通用 key | `ele.raw_name`、`rel.raw_text`、`rel.node_ast_kind` |
| `x/java/constants/` | `java.*` | Java 语言特化 key | `java.class.superclass`、`java.method.return_type`、`java.rel.call.receiver` |
| `x/cpp/constants/` | `cpp.*` | C/C++ 语言特化 key | `cpp.function.return_type`、`cpp.class.superclass`、`cpp.condition`、`cpp.rel.impl_link.signature` |

**示例（Java 语言特化）**：
```go
// x/java/constants/ele_const.go
const (
    ClassSuperClass         = "java.class.superclass"      // 父类名（含泛型）
    ClassImplementedInterfaces = "java.class.interfaces"   // 实现的接口列表
    MethodReturnType        = "java.method.return_type"    // 返回值原始类型文本
    MethodParameters        = "java.method.parameters"     // 参数列表（类型+名称）
    MethodThrowsTypes       = "java.method.throws"         // 声明抛出的异常类型
    VariableRawType         = "java.variable.raw_type"     // 局部变量类型文本
    FieldRawType            = "java.field.raw_type"        // 字段类型文本
)

// x/java/constants/rel_const.go
const (
    RelRawText       = "java.rel.raw_text"             // 动作完整源码文本
    RelCallReceiver  = "java.rel.call.receiver"        // 调用接收者
    RelAssignIsCapture = "java.rel.assign.is_capture"  // 跨作用域捕获赋值
    RelTypeArgDepth  = "java.rel.type_arg.depth"       // 泛型嵌套深度
)
```

**示例（C/C++ 语言特化）**：
```go
// x/cpp/constants/cpp_constants.go
const (
    FunctionReturnType   = "cpp.function.return_type"       // 返回类型原始文本
    FunctionParameters   = "cpp.function.parameters"        // 参数列表（类型+名称）
    FunctionIsDeclaration = "cpp.function.is_declaration"   // 是否为声明（无函数体）
    FunctionIsPureVirtual = "cpp.function.is_pure_virtual"  // 纯虚函数（= 0）
    VariableRawType      = "cpp.variable.raw_type"          // 变量类型文本
    VariableIsGlobal     = "cpp.variable.is_global"         // 全局变量
    VariableLambdaQN     = "cpp.variable.lambda_qn"         // 变量持有的 Lambda QN
    FieldRawType         = "cpp.field.raw_type"             // 字段类型文本
    ClassSuperClass      = "cpp.class.superclass"           // 基类列表
    TypeUnderlying       = "cpp.type.underlying"            // typedef/using 底层类型
    MacroBody            = "cpp.macro.body"                 // 宏体原文
    Condition            = "cpp.condition"                  // 条件编译分支标签
    Visibility           = "cpp.visibility"                 // public/private/protected
    IsDeclaration        = "cpp.is_declaration"             // 通用声明标记

    RelRawText           = "cpp.rel.raw_text"               // 关系原始文本
    RelAstKind           = "cpp.rel.ast_kind"               // 触发关系的 AST 节点类型
    RelIncludeIsSystem   = "cpp.rel.include.is_system"      // 系统头文件
    RelImplLinkSignature = "cpp.rel.impl_link.signature"    // ImplLink 配对签名
)
```

**约定**：新增 Mores key 必须：
1. 定义在对应 constants 文件，使用带语义的分组前缀。
2. 不在业务代码中直接写裸字符串。

## 5. 临时字段（私有传递通道）

`DependencyRelation` 上保留**私有不导出临时字段**，供语言实现内部各子阶段（extract→enricher）传递 AST 节点：

```go
// model/relation.go
type DependencyRelation struct {
    // ...公开字段...

    // 私有临时字段：仅语言实现内部各子阶段传递 AST 节点，
    // 引擎在流水线结束时统一清空，绝不进入 JSON 输出。
    // 值类型为 core/ast.Node 接口（语言无关）；model 保持零依赖，故用 any。
    tmpNode any // 触发该关系的 AST 节点
    tmpExpr any // 表达式节点
    tmpCtx  any // 上下文容器节点
}
```

**设计要点**：
- 类型用 `ast.Node` 接口（落地为 `any`）而非具体 `*sitter.Node`，保持 model 对 AST 库零依赖。
- 临时字段不导出、无 JSON tag，天然不会进入输出。
- 引擎（或 extractor 出口）在流水线结束时调用 `ClearTemp()` 统一置 nil，杜绝泄漏。
- 通过导出访问器（`SetTmpNode`/`TmpNode`/`SetTmpExpr`/`TmpExpr`/`SetTmpCtx`/`TmpCtx`）操作。

## 6. Location（位置）

```go
type Location struct {
    FilePath    string // 文件路径
    StartLine   int
    EndLine     int
    StartColumn int
    EndColumn   int
}
```

## 7. 序列化契约

- JSON 字段名使用 Go 结构体字段名（PascalCase）：`Kind`/`Name`/`QualifiedName`/`Type`/`Source`/`Target`/...
- 可省略字段（`Signature`/`Location`/`Doc`/`Comment`/`Extra`/`Mores`/`Details`）带 `omitempty`。
- 临时字段无 JSON tag，永不序列化。
