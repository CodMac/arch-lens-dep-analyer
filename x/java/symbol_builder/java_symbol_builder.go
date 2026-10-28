// Package symbol_builder 实现 Java 语言的符号构建能力（QualifiedName 拼接、基本类型判定）。
package symbol_builder

// SymbolBuilder 实现 core.SymbolBuilder：用点号拼接 QN，识别 Java 基本类型。
type SymbolBuilder struct{}

// New 创建 Java SymbolBuilder。
func New() *SymbolBuilder { return &SymbolBuilder{} }

// BuildQualifiedName 拼接父符号 QN 与子符号名称为完整 QN（Java 用点号）。
func (b *SymbolBuilder) BuildQualifiedName(parentQN, name string) string {
	if parentQN == "" || parentQN == "." {
		return name
	}
	return parentQN + "." + name
}

// IsPrimitive 判断类型名是否为 Java 基本类型。
func (b *SymbolBuilder) IsPrimitive(typeName string) bool {
	switch typeName {
	case "int", "long", "short", "byte", "char", "boolean", "float", "double":
		return true
	}
	return false
}
