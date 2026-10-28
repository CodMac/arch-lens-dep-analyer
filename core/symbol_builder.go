package core

// SymbolBuilder 负责语言特有的符号构建能力（QualifiedName 拼接、基本类型判定）。
type SymbolBuilder interface {
	// BuildQualifiedName 拼接父符号 QN 与子符号名称为完整 QN。
	// (Java 用点号，Go 也用点号，但实现可自定义分隔规则)
	BuildQualifiedName(parentQN, name string) string

	// IsPrimitive 判断类型名是否为语言基本类型。
	IsPrimitive(typeName string) bool
}
