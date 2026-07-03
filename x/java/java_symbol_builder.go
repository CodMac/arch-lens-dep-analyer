package java

import "slices"

type SymbolBuilder struct{}

func NewSymbolBuilder() *SymbolBuilder {
	return &SymbolBuilder{}
}

func (jbr *SymbolBuilder) IsPrimitive(t string) bool {
	pt := []string{"int", "long", "double", "float", "boolean", "char", "byte", "short", "void"}
	return slices.Contains(pt, t)
}

func (jbr *SymbolBuilder) BuildQualifiedName(parentQN, name string) string {
	if parentQN == "" || parentQN == "." {
		return name
	}
	return parentQN + "." + name
}
