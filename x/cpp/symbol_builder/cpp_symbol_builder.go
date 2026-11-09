// Package symbol_builder 实现 C/C++ 语言的符号构建能力（QualifiedName 拼接、基本类型判定）。
package symbol_builder

// SymbolBuilder 实现 core.SymbolBuilder：用 :: 拼接 QN，识别 C/C++ 基本类型。
type SymbolBuilder struct{}

// New 创建 C/C++ SymbolBuilder。
func New() *SymbolBuilder { return &SymbolBuilder{} }

// BuildQualifiedName 拼接父符号 QN 与子符号名称为完整 QN（C/C++ 用 ::）。
func (b *SymbolBuilder) BuildQualifiedName(parentQN, name string) string {
	if parentQN == "" || parentQN == "::" {
		return name
	}
	return parentQN + "::" + name
}

// IsPrimitive 判断类型名是否为 C/C++ 基本类型。
func (b *SymbolBuilder) IsPrimitive(typeName string) bool {
	switch typeName {
	case "void", "int", "long", "short", "char", "unsigned", "signed",
		"float", "double", "bool", "wchar_t", "size_t", "ssize_t":
		return true
	}
	return false
}
