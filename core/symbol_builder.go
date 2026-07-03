package core

import "fmt"

type SymbolBuilder interface {
	// BuildQualifiedName 根据父节点和当前名构建 QN
	// (Java 用 ".", C++ 用 "::")
	BuildQualifiedName(parentQN, name string) string

	// IsPrimitive 是否为基础类型
	IsPrimitive(t string) bool
}

var symbolBuilderMap = make(map[Language]SymbolBuilder)

// RegisterSymbolBuilder 注册一个语言与其对应的 SymbolBuilder 工厂函数。
func RegisterSymbolBuilder(lang Language, builder SymbolBuilder) {
	symbolBuilderMap[lang] = builder
}

// GetSymbolBuilder 根据语言类型获取对应的 SymbolBuilder 实例。
func GetSymbolBuilder(lang Language) (SymbolBuilder, error) {
	builder, ok := symbolBuilderMap[lang]
	if !ok {
		return nil, fmt.Errorf("no SymbolResolver for language: %s", lang)
	}

	return builder, nil
}
