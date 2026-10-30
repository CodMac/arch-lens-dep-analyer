// Package model 定义引擎的领域数据模型（纯数据结构，零依赖）。
package model

import "strings"

// ElementKind 表示代码实体的类型。
type ElementKind string

const (
	// --- 基本结构体 ---
	File      ElementKind = "FILE"      // 源文件
	Package   ElementKind = "PACKAGE"   // 包 (Go, Java, C++)
	Module    ElementKind = "MODULE"    // 模块 (Python, Rust)
	Namespace ElementKind = "NAMESPACE" // 命名空间 (C++, C#)

	// --- 面向对象/复合类型 ---
	AnonymousClass  ElementKind = "ANONYMOUS_CLASS"  // 匿名内部类 (Java)
	Class           ElementKind = "CLASS"            // 类 (Java, C++, Python, JS)
	Interface       ElementKind = "INTERFACE"        // 接口 (Java, Go, TS)
	AnonymousStruct ElementKind = "ANONYMOUS_STRUCT" // 匿名结构体 (Go)
	Struct          ElementKind = "STRUCT"           // 结构体 (Go, C)
	Enum            ElementKind = "ENUM"             // 枚举 (Java, C++, Rust)
	EnumConstant    ElementKind = "ENUM_CONSTANT"    // 枚举常量
	KAnnotation     ElementKind = "ANNOTATION"       // 注解 (Java, C++, Python, JS)
	Trait           ElementKind = "TRAIT"            // 特质 (Rust, Scala)

	// --- 可执行体 ---
	Lambda     ElementKind = "LAMBDA"      // Lambda 表达式 (Java)
	MethodRef  ElementKind = "METHOD_REF"  // 方法引用 (Java)
	ScopeBlock ElementKind = "SCOPE_BLOCK" // 代码块（用于分析局部变量作用域）
	Function   ElementKind = "FUNCTION"    // 独立函数 (C, Go, JS)
	Method     ElementKind = "METHOD"      // 类/结构体的方法
	Macro      ElementKind = "MACRO"       // 预处理器宏 (C/C++)

	// --- 存储和声明 ---
	Variable ElementKind = "VARIABLE" // 局部/全局变量
	Constant ElementKind = "CONSTANT" // 常量
	Field    ElementKind = "FIELD"    // 类/结构体/枚举的成员或字段
	Type     ElementKind = "TYPE"     // 自定义类型别名或基本类型引用

	// --- 基本类型 ---
	Primitive ElementKind = "PRIMITIVE" // Java 基本类型 (int/long/boolean/...)

	// --- 其他 ---
	Unknown ElementKind = "UNKNOWN" // 未知类型
)

// Location 描述代码元素或依赖关系在源码中的位置。
type Location struct {
	FilePath    string `json:"FilePath"`
	StartLine   int    `json:"StartLine"`
	EndLine     int    `json:"EndLine"`
	StartColumn int    `json:"StartColumn"`
	EndColumn   int    `json:"EndColumn"`
}

// CodeElement 描述源码中的一个可识别实体（Source 或 Target）。
type CodeElement struct {
	Kind           ElementKind `json:"Kind"`                // 元素类型 (e.g., CLASS, METHOD)
	Name           string      `json:"Name"`                // 短名称 (e.g., "main")
	QualifiedName  string      `json:"QualifiedName"`       // 限定名 (e.g., "com.demo.OrderService.getTotal")
	Path           string      `json:"Path"`                // 源文件路径（相对项目根）
	Signature      string      `json:"Signature,omitempty"` // 完整签名（方法含参数与返回值）
	Location       *Location   `json:"Location,omitempty"`  // 源码位置
	Doc            string      `json:"Doc,omitempty"`       // 文档注释
	Comment        string      `json:"Comment,omitempty"`   // 普通注释
	Extra          *Extra      `json:"Extra,omitempty"`     // 额外元数据
	IsFormSource   bool        `json:"IsFormSource"`        // 源码产生的实体
	IsFormSugar    bool        `json:"IsFormSugar"`         // 语法糖产生的实体
	IsFormExternal bool        `json:"IsFormExternal"`      // 外部导入产生的实体
	// PackageName 元素所属包：内部元素=包 QN；外部元素=import 包前缀或 <external>。
	PackageName string `json:"PackageName,omitempty"`
}

// Extra 是 CodeElement 的额外信息，包含跨语言通用的关键元数据。
type Extra struct {
	Modifiers   []string               `json:"Modifiers,omitempty"`  // 修饰符列表 (public/private/static/...)
	Annotations []string               `json:"Annotation,omitempty"` // 注解列表 (@Service/...)
	Mores       map[string]interface{} `json:"Mores,omitempty"`      // 语言特化键值字典
}

// NewExternalElement 构造一个外部/虚拟符号（IsFormExternal=true），用于兜底节点。
// PackageName 依据 qualifiedName 提取：有点号取包前缀，否则归 <external>。
func NewExternalElement(kind ElementKind, name, qualifiedName string) *CodeElement {
	pkg := "<external>"
	if i := strings.LastIndex(qualifiedName, "."); i > 0 {
		pkg = qualifiedName[:i]
	}
	return &CodeElement{
		Kind:           kind,
		Name:           name,
		QualifiedName:  qualifiedName,
		PackageName:    pkg,
		IsFormExternal: true,
	}
}

// GetMores 惰性初始化并返回元素的 Mores 字典。
func (e *CodeElement) GetMores() map[string]interface{} {
	if e.Extra == nil {
		e.Extra = &Extra{}
	}
	if e.Extra.Mores == nil {
		e.Extra.Mores = make(map[string]interface{})
	}
	return e.Extra.Mores
}
